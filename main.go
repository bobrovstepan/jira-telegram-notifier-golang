package main

import (
	"log"
	"net/http"
	"os"

	"jira-telegram-notifier/internal/handler"
	"jira-telegram-notifier/internal/telegram"
)

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	threadID := os.Getenv("TELEGRAM_THREAD_ID")
	webhookSecret := os.Getenv("JIRA_WEBHOOK_SECRET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if botToken == "" || chatID == "" || webhookSecret == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID and JIRA_WEBHOOK_SECRET are required")
	}

	sender := telegram.NewSender(botToken, chatID, threadID)
	jiraHandler := handler.NewJiraHandler(webhookSecret, sender)

	http.HandleFunc("/api/webhooks/jira", jiraHandler.Handle)

	log.Printf("Listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
