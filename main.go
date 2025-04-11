// Copyright (C) 2025 Roman Gorshunov
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License Version 3.0.
//
// Anonymous Telegram Bot "Cry Out"
// Description: A Telegram bot that verifies and publishes anonymous messages in a channel
//              using the OpenAI API, with webhook/polling support and rate limiting.

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	openai "github.com/sashabaranov/go-openai"
)

var (
	userLastMessageTime = make(map[int]time.Time)
	userLastMessage     = make(map[int]string)
	mu                  sync.Mutex
	rateLimitDuration   = 20 * time.Minute
)

func verifyMessageWithOpenAI(client *openai.Client, message string) (bool, string, error) {
	prompt := os.Getenv("OPENAI_PROMPT")
	if prompt == "" {
		prompt = "Проверь, соответствует ли сообщение следующим критериям: написано преимущественно на русском языке, не содержит грубых ругательств (допускаются слова с символами '*'), а цель сообщения — пожаловаться, выплакаться, выговориться публично. Ответь только 'да' или 'нет'. Сообщение:\n\n"
	}

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: prompt + message},
		},
		MaxTokens: 10,
	})

	if err != nil {
		return false, "", err
	}

	answer := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))
	return answer == "да", answer, nil
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	openaiToken := os.Getenv("OPENAI_API_KEY")
	channelName := os.Getenv("TELEGRAM_BOT_CHANNEL_NAME")
	webhookURL := os.Getenv("WEBHOOK_URL")

	if botToken == "" || openaiToken == "" || channelName == "" {
		log.Fatal("Missing required environment variables")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	openaiClient := openai.NewClient(openaiToken)

	var updates tgbotapi.UpdatesChannel

	if webhookURL != "" {
		_, err = bot.SetWebhook(tgbotapi.NewWebhook(webhookURL))
		if err != nil {
			log.Fatal("Webhook error:", err)
		}
		info, err := bot.GetWebhookInfo()
		if err != nil || info.LastErrorDate != 0 {
			log.Fatal("Webhook setup error:", info.LastErrorMessage)
		}
		log.Println("Running in webhook mode.")
		updates = bot.ListenForWebhook("/")
		go http.ListenAndServe(":8080", nil)
	} else {
		log.Println("Running in polling mode.")
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		updates, err = bot.GetUpdatesChan(u)
		if err != nil {
			log.Fatal("Polling setup error:", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go handleSignals(cancel)

	processUpdates(ctx, bot, openaiClient, channelName, updates)

	log.Println("Bot shut down gracefully.")
}

func handleSignals(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	log.Println("Shutdown signal received, exiting...")
	cancel()
}

func processUpdates(ctx context.Context, bot *tgbotapi.BotAPI, openaiClient *openai.Client, channelName string, updates tgbotapi.UpdatesChannel) {
	rulesMessage := "📝 Правила отправки сообщений:\n\n" +
		"1. Цель сообщения – выплеснуть эмоции и получить поддержку.\n" +
		"2. Сообщение должно быть преимущественно на русском языке.\n" +
		"3. Запрещены грубые матерные выражения."

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping update processing.")
			return
		case update := <-updates:
			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			userID := update.Message.From.ID
			userMessage := update.Message.Text

			if userMessage == "/start" {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, rulesMessage))
				continue
			}

			mu.Lock()
			lastTime, exists := userLastMessageTime[userID]
			lastMsg := userLastMessage[userID]
			if exists && time.Since(lastTime) < rateLimitDuration {
				mu.Unlock()
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "⏳ Вы можете отправлять одно сообщение раз в 20 минут."))
				log.Printf("UserID: %d, Username: %s, Msg: '%s', User error: RateLimited message", userID, update.Message.From.UserName, userMessage)
				continue
			}
			if lastMsg == userMessage {
				mu.Unlock()
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "⚠️ Вы уже отправляли такое сообщение ранее."))
				log.Printf("UserID: %d, Username: %s, Msg: '%s', User error: Duplicated message", userID, update.Message.From.UserName, userMessage)
				continue
			}
			mu.Unlock()

			valid, openaiResponse, err := verifyMessageWithOpenAI(openaiClient, userMessage)
			if err != nil {
				log.Println("OpenAI error:", err)
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🚫 Ошибка проверки. Попробуйте позже."))
				continue
			}

			log.Printf("UserID: %d, Username: %s, Msg: '%s', OpenAI: '%s'", userID, update.Message.From.UserName, userMessage, openaiResponse)

			if !valid {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🚫 Сообщение не соответствует правилам.\n\n"+rulesMessage))
				continue
			}

			channelMsg, err := bot.Send(tgbotapi.NewMessageToChannel(channelName, userMessage))
			if err != nil {
				log.Println("Telegram error:", err)
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🚫 Ошибка публикации. Попробуйте позже."))
				continue
			}

			mu.Lock()
			userLastMessageTime[userID] = time.Now()
			userLastMessage[userID] = userMessage
			mu.Unlock()

			link := "https://t.me/" + strings.TrimPrefix(channelName, "@") + "/" + strconv.Itoa(channelMsg.MessageID)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Сообщение опубликовано: "+link))
		}
	}
}
