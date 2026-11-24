package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/zeromicro/go-zero/core/logx"

	"telegram_chat_bot/internal/config"
)

const (
	maxMessageUnits    = 4096
	maxConcurrentTurns = 8
)

type BotInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Entity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type Message struct {
	MessageID int64    `json:"message_id"`
	Text      string   `json:"text"`
	Entities  []Entity `json:"entities"`
	From      *User    `json:"from"`
	Chat      Chat     `json:"chat"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Question struct {
	UpdateID        int64
	ChatID          int64
	ChatTitle       string
	MessageID       int64
	UserID          int64
	Username        string
	UserDisplayName string
	Text            string
}

type QuestionHandler func(context.Context, Question) error

type Client struct {
	token        string
	baseURL      string
	allowedChats map[int64]struct{}
	pollTimeout  time.Duration
	http         *http.Client
	botMu        sync.RWMutex
	bot          BotInfo
}

func New(cfg config.TelegramConf, httpClient *http.Client) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	allowedChats, err := cfg.AllowedGroupIDs()
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second}
	}
	return &Client{
		token:        strings.TrimSpace(cfg.BotToken),
		baseURL:      strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/"),
		allowedChats: allowedChats,
		pollTimeout:  time.Duration(cfg.PollTimeoutSeconds) * time.Second,
		http:         httpClient,
	}, nil
}

func (c *Client) Bot() BotInfo {
	c.botMu.RLock()
	defer c.botMu.RUnlock()
	return c.bot
}

func (c *Client) Run(ctx context.Context, afterUpdateID int64, handler QuestionHandler) error {
	if handler == nil {
		return errors.New("Telegram question handler is required")
	}
	if err := c.deleteWebhook(ctx); err != nil {
		return err
	}
	bot, err := c.getMe(ctx)
	if err != nil {
		return err
	}
	c.botMu.Lock()
	c.bot = bot
	c.botMu.Unlock()
	logx.Infof("Telegram bot @%s connected for %d allowlisted group(s)", bot.Username, len(c.allowedChats))

	offset := afterUpdateID + 1
	sem := make(chan struct{}, maxConcurrentTurns)
	for {
		updates, err := c.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logx.Errorf("poll Telegram updates: %v", err)
			if !sleepContext(ctx, 3*time.Second) {
				return nil
			}
			continue
		}
		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			question, ok := c.question(update, bot.Username)
			if !ok {
				continue
			}
			sem <- struct{}{}
			go func(question Question) {
				defer func() { <-sem }()
				if err := handler(ctx, question); err != nil {
					logx.Errorf("handle Telegram update %d in chat %d: %v", question.UpdateID, question.ChatID, err)
				}
			}(question)
		}
	}
}

func (c *Client) SendMessage(ctx context.Context, chatID, replyToMessageID int64, content string) error {
	content = strings.TrimSpace(content)
	if chatID == 0 || content == "" {
		return errors.New("Telegram chat ID and message content are required")
	}
	chunks := SplitMessage(content, maxMessageUnits)
	for index, chunk := range chunks {
		payload := map[string]any{
			"chat_id": chatID,
			"text":    chunk,
		}
		if index == 0 && replyToMessageID > 0 {
			payload["reply_parameters"] = map[string]any{
				"message_id":                  replyToMessageID,
				"allow_sending_without_reply": true,
			}
		}
		if err := c.call(ctx, "sendMessage", nil, payload, nil); err != nil {
			return fmt.Errorf("send Telegram message %d/%d: %w", index+1, len(chunks), err)
		}
	}
	return nil
}

func SplitMessage(content string, maxUnits int) []string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) == 0 {
		return nil
	}
	if maxUnits <= 0 {
		return []string{string(runes)}
	}
	chunks := make([]string, 0, (len(runes)+maxUnits-1)/maxUnits)
	for len(runes) > 0 {
		splitAt := preferredSplit(runes, maxUnits)
		chunks = append(chunks, string(runes[:splitAt]))
		runes = runes[splitAt:]
	}
	return chunks
}

func preferredSplit(runes []rune, maxUnits int) int {
	hardLimit, newlineAt, spaceAt, units := 0, 0, 0, 0
	for index, value := range runes {
		units += utf16.RuneLen(value)
		if units > maxUnits {
			break
		}
		hardLimit = index + 1
		if value == '\n' {
			newlineAt = hardLimit
		} else if unicode.IsSpace(value) {
			spaceAt = hardLimit
		}
	}
	if hardLimit == 0 {
		return 1
	}
	if hardLimit == len(runes) {
		return hardLimit
	}
	if newlineAt > 0 {
		return newlineAt
	}
	if spaceAt > 0 {
		return spaceAt
	}
	return hardLimit
}

func (c *Client) question(update Update, botUsername string) (Question, bool) {
	message := update.Message
	if message == nil || message.From == nil || message.From.IsBot {
		return Question{}, false
	}
	if message.Chat.Type != "group" && message.Chat.Type != "supergroup" {
		return Question{}, false
	}
	if _, ok := c.allowedChats[message.Chat.ID]; !ok {
		return Question{}, false
	}
	text, ok := stripBotMention(message.Text, message.Entities, botUsername)
	if !ok {
		return Question{}, false
	}
	return Question{
		UpdateID:        update.UpdateID,
		ChatID:          message.Chat.ID,
		ChatTitle:       strings.TrimSpace(message.Chat.Title),
		MessageID:       message.MessageID,
		UserID:          message.From.ID,
		Username:        message.From.Username,
		UserDisplayName: displayName(message.From),
		Text:            text,
	}, true
}

func stripBotMention(text string, entities []Entity, botUsername string) (string, bool) {
	target := "@" + strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if target == "@" || strings.TrimSpace(text) == "" {
		return "", false
	}
	units := utf16.Encode([]rune(text))
	for _, entity := range entities {
		if entity.Type != "mention" || entity.Offset < 0 || entity.Length <= 0 {
			continue
		}
		end := entity.Offset + entity.Length
		if end > len(units) {
			continue
		}
		mention := string(utf16.Decode(units[entity.Offset:end]))
		if !strings.EqualFold(mention, target) {
			continue
		}
		cleaned := make([]uint16, 0, len(units)-entity.Length)
		cleaned = append(cleaned, units[:entity.Offset]...)
		cleaned = append(cleaned, units[end:]...)
		return strings.TrimSpace(string(utf16.Decode(cleaned))), true
	}
	return "", false
}

func displayName(user *User) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name != "" {
		return name
	}
	if user.Username != "" {
		return user.Username
	}
	return strconv.FormatInt(user.ID, 10)
}

func (c *Client) getMe(ctx context.Context) (BotInfo, error) {
	var bot BotInfo
	if err := c.call(ctx, "getMe", nil, nil, &bot); err != nil {
		return BotInfo{}, fmt.Errorf("get Telegram bot identity: %w", err)
	}
	if strings.TrimSpace(bot.Username) == "" {
		return BotInfo{}, errors.New("Telegram getMe returned an empty username")
	}
	return bot, nil
}

func (c *Client) deleteWebhook(ctx context.Context) error {
	if err := c.call(ctx, "deleteWebhook", nil, map[string]any{"drop_pending_updates": false}, nil); err != nil {
		return fmt.Errorf("disable Telegram webhook before polling: %w", err)
	}
	return nil
}

func (c *Client) getUpdates(ctx context.Context, offset int64) ([]Update, error) {
	query := url.Values{}
	query.Set("offset", strconv.FormatInt(offset, 10))
	query.Set("timeout", strconv.Itoa(int(c.pollTimeout/time.Second)))
	query.Set("allowed_updates", `["message"]`)
	var updates []Update
	if err := c.call(ctx, "getUpdates", query, nil, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
}

func (c *Client) call(ctx context.Context, method string, query url.Values, payload any, result any) error {
	endpoint := c.baseURL + "/bot" + c.token + "/" + method
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	var envelope apiResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode Telegram API response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !envelope.OK {
		if envelope.Description != "" {
			return errors.New(envelope.Description)
		}
		return fmt.Errorf("Telegram API returned HTTP %d", response.StatusCode)
	}
	if result != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return fmt.Errorf("decode Telegram API result: %w", err)
		}
	}
	return nil
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
