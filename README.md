# Anonymous Telegram Bot "Cry Out"

This is a Telegram bot for publishing anonymous messages from Russian-speaking users who want to vent, complain, or share their pain. The bot uses the OpenAI API to ensure messages meet community guidelines and then posts them in a Telegram channel, where others can express support through reactions and comments.

---

## 🔧 Features

- 📥 Accepts user messages
- 🤖 Verifies messages using OpenAI (GPT-3.5)
- 📜 Checks compliance with posting rules (in Russian, no harsh swearing, etc.)
- 📢 Publishes to the specified Telegram channel
- 🔗 Replies to the user with a direct link to the published message
- ⏳ Rate limiting (1 message per user per 20 min)
- 📛 Deduplication (prevents duplicate messages)
- 🔗 Supports polling and webhook modes
- 🔄 Implements handling for SIGINT/SIGTERM signals to allow graceful shutdowns

---

## 🚀 Quick Start (Docker)

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/tg-anon-bot.git
   cd tg-anon-bot
   ```

1. Create an .env file with environment variables:

   ```bash
   TELEGRAM_BOT_TOKEN="your_telegram_bot_token"
   TELEGRAM_BOT_CHANNEL_NAME="@your_channel_name"
   OPENAI_API_KEY="your_openai_api_key"
   WEBHOOK_URL="your_webhook_url"  # leave empty for polling mode
   OPENAI_PROMPT="Проверь, соответствует ли сообщение следующим критериям: написано преимущественно на русском языке, не содержит грубых ругательств (допускаются слова с символами '*'), а цель сообщения — пожаловаться, выплакаться, выговориться публично. Ответь только 'да' или 'нет'. Сообщение:"
   ```

1. Build and run the Docker container:

   ```bash
   docker build -t tg-anon-bot .
   docker run -d --restart=always -p 8080:8080 --env-file .env tg-anon-bot
   ```

---

## 📋 Message Rules

1. The message should be mostly in Russian.
1. Harsh obscene language is not allowed.
1. Mild swearing is allowed if letters are replaced with * so that the meaning is still clear.
1. The message should aim to express emotion, vent, and seek support.

---

## 🔒 License

This project is licensed under the [GPL-3.0](LICENSE). See the full [LICENSE](LICENSE) file for more details.

---

## 🤝 Acknowledgements

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) – Telegram Bot API wrapper for Go
- [sashabaranov/go-openai](https://github.com/sashabaranov/go-openai) – OpenAI GPT client for Go

---

## 🚧 Future Improvements

- 💾 **Backup Strategy:** Consider persisting important data (e.g., using SQLite) to maintain state across restarts

---

## 📫 Contact

If you have suggestions or questions — feel free to open an issue or submit a pull request!

---

## 👤 Author

- Roman – [gorshunovr](https://github.com/gorshunovr)