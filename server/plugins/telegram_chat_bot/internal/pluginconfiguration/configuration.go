package pluginconfiguration

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"telegram_chat_bot/internal/config"
)

const encryptedValuePrefix = "v1:"

type Settings struct {
	Telegram     config.TelegramConf     `json:"telegram"`
	Conversation config.ConversationConf `json:"conversation"`
}

func Defaults() Settings {
	return Settings{
		Telegram: config.TelegramConf{
			APIBaseURL:            "https://api.telegram.org",
			PollTimeoutSeconds:    25,
			RequestTimeoutSeconds: 40,
		},
		Conversation: config.ConversationConf{
			BaseURL:        "http://127.0.0.1:8001",
			TimeoutSeconds: 600,
		},
	}
}

func FromConfig(cfg config.Config) Settings {
	return Settings{
		Telegram:     cfg.Telegram,
		Conversation: cfg.Conversation,
	}
}

func (s Settings) Apply(base config.Config) config.Config {
	base.Telegram = s.Telegram
	base.Conversation = s.Conversation
	return base
}

func Encrypt(settings Settings, masterSecret string) (string, error) {
	plain, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("marshal Telegram plugin configuration: %w", err)
	}
	block, err := aes.NewCipher(derivedKey(masterSecret))
	if err != nil {
		return "", fmt.Errorf("create configuration cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create configuration cipher mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("create configuration nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	return encryptedValuePrefix + base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func Decrypt(value, masterSecret string) (Settings, error) {
	if !strings.HasPrefix(value, encryptedValuePrefix) {
		return Settings{}, errors.New("unsupported Telegram plugin configuration version")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, encryptedValuePrefix))
	if err != nil {
		return Settings{}, fmt.Errorf("decode Telegram plugin configuration: %w", err)
	}
	block, err := aes.NewCipher(derivedKey(masterSecret))
	if err != nil {
		return Settings{}, fmt.Errorf("create configuration cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Settings{}, fmt.Errorf("create configuration cipher mode: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return Settings{}, errors.New("invalid Telegram plugin configuration")
	}
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return Settings{}, fmt.Errorf("decrypt Telegram plugin configuration: %w", err)
	}
	var settings Settings
	if err := json.Unmarshal(plain, &settings); err != nil {
		return Settings{}, fmt.Errorf("decode Telegram plugin configuration: %w", err)
	}
	return settings, nil
}

func derivedKey(masterSecret string) []byte {
	sum := sha256.Sum256([]byte("agentrazor/telegram-plugin-configuration/v1:" + masterSecret))
	return sum[:]
}
