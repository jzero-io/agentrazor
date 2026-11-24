package pluginconfiguration

import (
	"testing"

	"telegram_chat_bot/internal/config"
)

func TestEncryptRoundTrip(t *testing.T) {
	settings := Settings{
		Telegram: config.TelegramConf{
			BotToken:              "secret-token",
			GroupChatIDs:          "-1001",
			APIBaseURL:            "https://api.telegram.org",
			PollTimeoutSeconds:    25,
			RequestTimeoutSeconds: 40,
		},
		Conversation: config.ConversationConf{
			BaseURL:        "http://server:8001",
			APIKey:         "agentrazor-key",
			TimeoutSeconds: 600,
		},
	}
	encrypted, err := Encrypt(settings, "master-secret")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := Decrypt(encrypted, "master-secret")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted.Telegram.BotToken != settings.Telegram.BotToken ||
		decrypted.Conversation.APIKey != settings.Conversation.APIKey ||
		decrypted.Telegram.GroupChatIDs != settings.Telegram.GroupChatIDs {
		t.Fatalf("round trip mismatch: %#v", decrypted)
	}
}
