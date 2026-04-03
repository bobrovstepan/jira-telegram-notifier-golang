package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"net/http"
	"regexp"
	"strings"

	"jira-telegram-notifier/internal/telegram"
)

type JiraHandler struct{}

func NewJiraHandler() *JiraHandler {
	return &JiraHandler{}
}

func (h *JiraHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	projectKey := getProjectKey(payload)
	secret := getWebhookSecret(projectKey)

	if !verifySignature(r.Header.Get("X-Hub-Signature"), body, secret) {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	message := buildMessage(payload)
	if message != "" {
		if sender := newSenderForProject(projectKey); sender != nil {
			sender.Send(message)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"message":"ok"}`)
}

func getProjectKey(payload map[string]any) string {
	issue, _ := payload["issue"].(map[string]any)
	fields, _ := issue["fields"].(map[string]any)
	project, _ := fields["project"].(map[string]any)
	key, _ := project["key"].(string)
	return key
}

func newSenderForProject(projectKey string) *telegram.Sender {
	botToken := os.Getenv(projectKey + "_TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv(projectKey + "_TELEGRAM_CHAT_ID")
	threadID := os.Getenv(projectKey + "_TELEGRAM_THREAD_ID")
	if botToken == "" || chatID == "" {
		return nil
	}
	return telegram.NewSender(botToken, chatID, threadID)
}

func verifySignature(header string, body []byte, secret string) bool {
	if header == "" || !strings.Contains(header, "=") {
		return false
	}

	parts := strings.SplitN(header, "=", 2)
	theirSig := parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	ourSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(ourSig), []byte(theirSig))
}

func buildMessage(payload map[string]any) string {
	eventType, _ := payload["webhookEvent"].(string)
	if eventType == "" {
		eventType = "unknown"
	}

	issue, _ := payload["issue"].(map[string]any)
	user, _ := payload["user"].(map[string]any)
	comment, _ := payload["comment"].(map[string]any)
	changelog, _ := payload["changelog"].(map[string]any)

	if issue == nil && comment == nil && changelog == nil {
		return ""
	}

	var lines []string

	lines = append(lines, "<b>Jira: "+formatEventType(eventType)+"</b>")

	if issue != nil {
		key, _ := issue["key"].(string)
		fields, _ := issue["fields"].(map[string]any)
		summary := ""
		status := ""
		if fields != nil {
			summary, _ = fields["summary"].(string)
			if s, ok := fields["status"].(map[string]any); ok {
				status, _ = s["name"].(string)
			}
		}
		self, _ := issue["self"].(string)

		keyTag := "<b>" + key + "</b>"
		if self != "" {
			re := regexp.MustCompile(`/rest/api/.*`)
			taskURL := re.ReplaceAllString(self, "/browse/"+key)
			keyTag = fmt.Sprintf(`<a href="%s"><b>%s</b></a>`, taskURL, key)
		}

		lines = append(lines, "🎫 "+keyTag+": "+summary)
		if status != "" {
			lines = append(lines, "📌 Status: "+status)
		}
	}

	if user != nil {
		displayName, _ := user["displayName"].(string)
		if displayName == "" {
			displayName, _ = user["name"].(string)
		}
		lines = append(lines, "👤 By: "+displayName)
	}

	if comment != nil {
		body, _ := comment["body"].(string)
		re := regexp.MustCompile(`\[~accountid:[^\]]+\]`)
		body = re.ReplaceAllString(body, "")
		body = strings.TrimSpace(body)
		if len([]rune(body)) > 300 {
			body = string([]rune(body)[:300]) + "..."
		}
		author := ""
		if a, ok := comment["author"].(map[string]any); ok {
			author, _ = a["displayName"].(string)
		}
		line := "💬 Comment"
		if author != "" {
			line += " (" + author + ")"
		}
		lines = append(lines, line+": "+body)
	}

	if changelog != nil {
		items, _ := changelog["items"].([]any)
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			field, _ := item["field"].(string)
			fromStr, _ := item["fromString"].(string)
			toString, _ := item["toString"].(string)
			if fromStr == "" {
				fromStr = "—"
			}
			if toString == "" {
				toString = "—"
			}
			lines = append(lines, "🔄 "+field+": "+fromStr+" → "+toString)
		}
	}

	return strings.Join(lines, "\n")
}

func formatEventType(event string) string {
	switch {
		case strings.Contains(event, "created"):
			return "Created"
		case strings.Contains(event, "updated"):
			return "Updated"
		case strings.Contains(event, "deleted"):
			return "Deleted"
		case strings.Contains(event, "comment"):
			return "Comment"
		case strings.Contains(event, "worklog"):
			return "Worklog"
		case strings.Contains(event, "sprint"):
			return "Sprint"
		default:
			s := strings.ReplaceAll(event, "jira:", "")
			s = strings.ReplaceAll(s, "_", " ")
			return strings.Title(s)
	}
}

func getProjectKeyFromPayload(payload map[string]any) string {
	issue, _ := payload["issue"].(map[string]any)
	if issue == nil {
		return ""
	}
	fields, _ := issue["fields"].(map[string]any)
	if fields == nil {
		return ""
	}
	project, _ := fields["project"].(map[string]any)
	if project == nil {
		return ""
	}
	key, _ := project["key"].(string)
	return key
}

func getWebhookSecret(projectKey string) string {
	return os.Getenv(projectKey + "_JIRA_WEBHOOK_SECRET")
}