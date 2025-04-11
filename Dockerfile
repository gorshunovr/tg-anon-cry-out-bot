# Build stage
FROM golang:1.24-alpine AS builder

RUN apk update && apk add --no-cache git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot main.go

# Final minimal stage
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/bot .

# Expose port for webhook
EXPOSE 8080

CMD ["./bot"]
