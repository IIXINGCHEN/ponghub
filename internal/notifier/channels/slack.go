package channels

import (
	"fmt"
	"os"
	"time"

	"github.com/wcy-dt/ponghub/internal/types/structures/configure"
)

// SlackNotifier implements Slack webhook notifications
type SlackNotifier struct {
	config *configure.SlackConfig
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(config *configure.SlackConfig) *SlackNotifier {
	return &SlackNotifier{config: config}
}

// Send sends a Slack webhook notification with enhanced features
//
//goland:noinspection GoErrorStringFormat
func (s *SlackNotifier) Send(title, message string) error {
	webhookURL := s.config.WebhookURL
	if webhookURL == "" {
		webhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	}

	if webhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	var payload map[string]interface{}

	if s.config.UseBlocks {
		payload = s.buildBlockPayload(title, message)
	} else {
		payload = s.buildAttachmentPayload(title, message)
	}

	// Add basic configuration
	if s.config.Channel != "" {
		payload["channel"] = s.config.Channel
	}
	if s.config.Username != "" {
		payload["username"] = s.config.Username
	}
	if s.config.IconEmoji != "" {
		payload["icon_emoji"] = s.config.IconEmoji
	}
	if s.config.IconURL != "" {
		payload["icon_url"] = s.config.IconURL
	}

	// Execute request with retry logic
	return s.sendWithRetry(webhookURL, payload)
}

// buildBlockPayload creates a Slack Block Kit payload
func (s *SlackNotifier) buildBlockPayload(title, message string) map[string]interface{} {
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": title,
			},
		},
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": message,
			},
		},
	}

	// Add context block with timestamp
	contextBlock := map[string]interface{}{
		"type": "context",
		"elements": []map[string]interface{}{
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Timestamp:* %s", time.Now().Format("2006-01-02 15:04:05 UTC")),
			},
		},
	}
	blocks = append(blocks, contextBlock)

	// Add divider
	blocks = append(blocks, map[string]interface{}{
		"type": "divider",
	})

	payload := map[string]interface{}{
		"blocks": blocks,
	}

	// Add mentions in text field if configured
	if len(s.config.Mentions) > 0 {
		mentionText := ""
		for _, mention := range s.config.Mentions {
			if mention != "" {
				// Support different mention formats
				if mention[0] == '@' || mention[0] == '#' || mention[0] == '!' {
					mentionText += mention + " "
				} else {
					mentionText += "@" + mention + " "
				}
			}
		}
		if mentionText != "" {
			payload["text"] = mentionText + title
		}
	}

	return payload
}

// buildAttachmentPayload creates a Slack attachment payload (legacy format)
func (s *SlackNotifier) buildAttachmentPayload(title, message string) map[string]interface{} {
	attachment := map[string]interface{}{
		"title":     title,
		"text":      message,
		"color":     s.getAttachmentColor(),
		"timestamp": time.Now().Unix(),
		"fields": []map[string]interface{}{
			{
				"title": "Service Status",
				"value": "Alert triggered",
				"short": true,
			},
			{
				"title": "Timestamp",
				"value": time.Now().Format("2006-01-02 15:04:05 UTC"),
				"short": true,
			},
		},
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{attachment},
	}

	// Add mentions in main text if configured
	if len(s.config.Mentions) > 0 {
		mentionText := ""
		for _, mention := range s.config.Mentions {
			if mention != "" {
				if mention[0] == '@' || mention[0] == '#' || mention[0] == '!' {
					mentionText += mention + " "
				} else {
					mentionText += "@" + mention + " "
				}
			}
		}
		if mentionText != "" {
			payload["text"] = mentionText + title
		}
	}

	return payload
}

// getAttachmentColor returns the color for Slack attachments
func (s *SlackNotifier) getAttachmentColor() string {
	if s.config.Color != "" {
		return s.config.Color
	}
	return "danger" // Default to red for alerts
}

// sendWithRetry sends the Slack webhook with retry logic
func (s *SlackNotifier) sendWithRetry(webhookURL string, payload map[string]interface{}) error {
	maxRetries := 0
	if s.config.Retries > 0 {
		maxRetries = s.config.Retries
	}

	timeout := 30
	if s.config.Timeout > 0 {
		timeout = s.config.Timeout
	}

	headers := make(map[string]string)
	if s.config.UserAgent != "" {
		headers["User-Agent"] = s.config.UserAgent
	} else {
		headers["User-Agent"] = "PongHub-Slack-Notifier/1.0"
	}

	return sendHTTPRequest(webhookURL, "POST", payload, headers, maxRetries, timeout, false)
}
