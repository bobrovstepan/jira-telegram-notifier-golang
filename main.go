package main

import (
	"log"
	"net/http"
	"os"

	"jira-telegram-notifier/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	webhookPath := os.Getenv("WEBHOOK_PATH")
	if webhookPath == "" {
		webhookPath = "/api/webhooks/jira"
	}

	jiraHandler := handler.NewJiraHandler()

	http.HandleFunc(webhookPath, jiraHandler.Handle)

	log.Printf("Listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
