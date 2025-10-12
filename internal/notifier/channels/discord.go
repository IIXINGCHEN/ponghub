package channels

import (
	"fmt"
	"os"
	"time"

	"github.com/wcy-dt/ponghub/internal/types/structures/configure"
)

// DiscordNotifier implements Discord webhook notifications
type DiscordNotifier struct {
	config *configure.DiscordConfig
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(config *configure.DiscordConfig) *DiscordNotifier {
	return &DiscordNotifier{config: config}
}

// Send sends a Discord webhook notification with enhanced features
//
//goland:noinspection GoErrorStringFormat
func (d *DiscordNotifier) Send(title, message string) error {
	webhookURL := d.config.WebhookURL
	if webhookURL == "" {
		webhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
	}

	if webhookURL == "" {
		return fmt.Errorf("Discord webhook URL not configured")
	}

	var payload map[string]interface{}

	if d.config.UseEmbeds {
		payload = d.buildEmbedPayload(title, message)
	} else {
		payload = d.buildTextPayload(title, message)
	}

	// Add username and avatar if configured
	if d.config.Username != "" {
		payload["username"] = d.config.Username
	}
	if d.config.AvatarURL != "" {
		payload["avatar_url"] = d.config.AvatarURL
	}

	// Execute request with retry logic
	return d.sendWithRetry(webhookURL, payload)
}

// buildEmbedPayload creates a rich embed payload
func (d *DiscordNotifier) buildEmbedPayload(title, message string) map[string]interface{} {
	embed := map[string]interface{}{
		"title":       title,
		"description": message,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	// Add color if configured
	if d.config.Color > 0 {
		embed["color"] = d.config.Color
	} else {
		embed["color"] = 0xFF0000 // Default red color for alerts
	}

	// Add fields for additional context
	fields := []map[string]interface{}{
		{
			"name":   "Service Status",
			"value":  "Alert triggered",
			"inline": true,
		},
		{
			"name":   "Timestamp",
			"value":  time.Now().Format("2006-01-02 15:04:05 UTC"),
			"inline": true,
		},
	}
	embed["fields"] = fields

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}

	// Add mentions if configured
	if len(d.config.Mentions) > 0 {
		content := ""
		for _, mention := range d.config.Mentions {
			if mention != "" {
				// Support both user IDs and role IDs
				if len(mention) > 15 && mention[0] != '<' {
					// Assume it's a raw ID, format it
					if mention[0] == '&' {
						content += fmt.Sprintf("<@&%s> ", mention[1:]) // Role mention
					} else {
						content += fmt.Sprintf("<@%s> ", mention) // User mention
					}
				} else {
					content += fmt.Sprintf("%s ", mention) // Pre-formatted mention
				}
			}
		}
		if content != "" {
			payload["content"] = content
		}
	}

	return payload
}

// buildTextPayload creates a simple text payload
func (d *DiscordNotifier) buildTextPayload(title, message string) map[string]interface{} {
	content := fmt.Sprintf("**%s**\n```\n%s\n```", title, message)

	// Add mentions if configured
	if len(d.config.Mentions) > 0 {
		mentionContent := ""
		for _, mention := range d.config.Mentions {
			if mention != "" {
				if len(mention) > 15 && mention[0] != '<' {
					if mention[0] == '&' {
						mentionContent += fmt.Sprintf("<@&%s> ", mention[1:])
					} else {
						mentionContent += fmt.Sprintf("<@%s> ", mention)
					}
				} else {
					mentionContent += fmt.Sprintf("%s ", mention)
				}
			}
		}
		if mentionContent != "" {
			content = mentionContent + "\n" + content
		}
	}

	return map[string]interface{}{
		"content": content,
	}
}

// sendWithRetry sends the Discord webhook with retry logic
func (d *DiscordNotifier) sendWithRetry(webhookURL string, payload map[string]interface{}) error {
	maxRetries := 0
	if d.config.Retries > 0 {
		maxRetries = d.config.Retries
	}

	timeout := 30
	if d.config.Timeout > 0 {
		timeout = d.config.Timeout
	}

	headers := make(map[string]string)
	headers["User-Agent"] = "PongHub-Discord-Notifier/1.0"

	return sendHTTPRequest(webhookURL, "POST", payload, headers, maxRetries, timeout, false)
}
