package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Sender struct {
	botToken string
	chatID   string
	threadID string
}

func NewSender(botToken, chatID, threadID string) *Sender {
	return &Sender{botToken: botToken, chatID: chatID, threadID: threadID}
}

func (s *Sender) Send(text string) {
	params := map[string]any{
		"chat_id":    s.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	if s.threadID != "" {
		params["message_thread_id"] = s.threadID
	}

	body, err := json.Marshal(params)
	if err != nil {
		log.Printf("telegram: failed to marshal params: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("telegram: request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("telegram: unexpected status %d", resp.StatusCode)
	}
}
