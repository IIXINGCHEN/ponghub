package channels

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wcy-dt/ponghub/internal/types/structures/configure"
)

// TelegramNotifier implements Telegram bot notifications
type TelegramNotifier struct {
	config *configure.TelegramConfig
}

// NewTelegramNotifier creates a new Telegram notifier
func NewTelegramNotifier(config *configure.TelegramConfig) *TelegramNotifier {
	return &TelegramNotifier{config: config}
}

// Send sends a Telegram bot notification with enhanced features
//
//goland:noinspection GoErrorStringFormat
func (t *TelegramNotifier) Send(title, message string) error {
	botToken := t.config.BotToken
	if botToken == "" {
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	chatID := t.config.ChatID
	if chatID == "" {
		chatID = os.Getenv("TELEGRAM_CHAT_ID")
	}

	if botToken == "" || chatID == "" {
		return fmt.Errorf("Telegram bot token or chat ID not configured")
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := t.buildPayload(chatID, title, message)

	// Execute request with retry logic
	return t.sendWithRetry(telegramURL, payload)
}

// buildPayload constructs the Telegram API payload
func (t *TelegramNotifier) buildPayload(chatID, title, message string) map[string]interface{} {
	// Format text based on parse mode
	text := t.formatMessage(title, message)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	// Set parse mode
	parseMode := t.config.ParseMode
	if parseMode == "" {
		parseMode = "Markdown" // Default to Markdown
	}
	payload["parse_mode"] = parseMode

	// Set other options
	if t.config.DisableWebPagePreview {
		payload["disable_web_page_preview"] = true
	}
	if t.config.DisableNotification {
		payload["disable_notification"] = true
	}

	// Add reply to message if configured
	if t.config.ReplyToMessageID > 0 {
		payload["reply_to_message_id"] = t.config.ReplyToMessageID
	}

	return payload
}

// formatMessage formats the message based on parse mode
func (t *TelegramNotifier) formatMessage(title, message string) string {
	parseMode := t.config.ParseMode
	if parseMode == "" {
		parseMode = "Markdown"
	}

	switch parseMode {
	case "HTML":
		return t.formatHTML(title, message)
	case "MarkdownV2":
		return t.formatMarkdownV2(title, message)
	case "Markdown":
		return t.formatMarkdown(title, message)
	default:
		// Plain text
		return fmt.Sprintf("%s\n\n%s", title, message)
	}
}

// formatMarkdown formats message using Markdown (legacy)
func (t *TelegramNotifier) formatMarkdown(title, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05 UTC")

	formatted := fmt.Sprintf("*%s*\n\n", title)
	formatted += fmt.Sprintf("```\n%s\n```\n\n", message)
	formatted += fmt.Sprintf("🕒 *Timestamp:* %s", timestamp)

	return formatted
}

// formatMarkdownV2 formats message using MarkdownV2
func (t *TelegramNotifier) formatMarkdownV2(title, message string) string {
	// Escape special characters for MarkdownV2
	title = t.escapeMarkdownV2(title)
	message = t.escapeMarkdownV2(message)
	timestamp := t.escapeMarkdownV2(time.Now().Format("2006-01-02 15:04:05 UTC"))

	formatted := fmt.Sprintf("*%s*\n\n", title)
	formatted += fmt.Sprintf("```\n%s\n```\n\n", message)
	formatted += fmt.Sprintf("🕒 *Timestamp:* %s", timestamp)

	return formatted
}

// formatHTML formats message using HTML
func (t *TelegramNotifier) formatHTML(title, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05 UTC")

	formatted := fmt.Sprintf("<b>%s</b>\n\n", title)
	formatted += fmt.Sprintf("<pre>%s</pre>\n\n", message)
	formatted += fmt.Sprintf("🕒 <b>Timestamp:</b> %s", timestamp)

	return formatted
}

// escapeMarkdownV2 escapes special characters for MarkdownV2
func (t *TelegramNotifier) escapeMarkdownV2(text string) string {
	// Characters that need to be escaped in MarkdownV2: '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!'
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}

	return text
}

// sendWithRetry sends the Telegram message with retry logic
func (t *TelegramNotifier) sendWithRetry(telegramURL string, payload map[string]interface{}) error {
	maxRetries := 0
	if t.config.Retries > 0 {
		maxRetries = t.config.Retries
	}

	timeout := 30
	if t.config.Timeout > 0 {
		timeout = t.config.Timeout
	}

	headers := make(map[string]string)
	if t.config.UserAgent != "" {
		headers["User-Agent"] = t.config.UserAgent
	} else {
		headers["User-Agent"] = "PongHub-Telegram-Notifier/1.0"
	}

	return sendHTTPRequest(telegramURL, "POST", payload, headers, maxRetries, timeout, false)
}
