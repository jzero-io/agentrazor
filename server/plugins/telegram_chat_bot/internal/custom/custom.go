package custom

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	conversation "github.com/jzero-io/agentrazor/core-engine/sdk/agent"
	"github.com/zeromicro/go-zero/core/logx"

	"telegram_chat_bot/internal/config"
	"telegram_chat_bot/internal/qa"
	"telegram_chat_bot/internal/svc"
	"telegram_chat_bot/internal/telegrambot"
)

type Custom struct {
	svcCtx *svc.ServiceContext

	mu             sync.RWMutex
	started        bool
	running        bool
	cancel         context.CancelFunc
	done           chan struct{}
	botName        string
	groups         int
	run            func(context.Context) error
	telegramClient *telegrambot.Client
}

func New(svcCtx *svc.ServiceContext) *Custom {
	return &Custom{svcCtx: svcCtx}
}

func (c *Custom) Init() error {
	cfg, err := c.svcCtx.EffectiveConfig(context.Background())
	if err != nil {
		return err
	}
	if err := cfg.Telegram.Validate(); err != nil {
		return err
	}
	if !cfg.Telegram.Enabled() {
		logx.Info("Telegram chat bot is disabled because no bot token is configured")
		return nil
	}
	if strings.TrimSpace(cfg.Conversation.APIKey) == "" {
		return errors.New("AgentRazor API key is required when Telegram is enabled")
	}
	timeout := time.Duration(cfg.Conversation.TimeoutSeconds) * time.Second
	conversationClient, err := conversation.NewClient(conversation.Config{
		BaseURL:    strings.TrimRight(strings.TrimSpace(cfg.Conversation.BaseURL), "/"),
		APIKey:     strings.TrimSpace(cfg.Conversation.APIKey),
		HTTPClient: &http.Client{Timeout: timeout},
	})
	if err != nil {
		return err
	}
	telegramClient, err := telegrambot.New(
		cfg.Telegram,
		&http.Client{Timeout: time.Duration(cfg.Telegram.RequestTimeoutSeconds+5) * time.Second},
	)
	if err != nil {
		return err
	}
	service, err := qa.New(conversationClient, telegramClient, c.svcCtx.TelegramConversation, timeout)
	if err != nil {
		return err
	}
	lastUpdateID, err := c.svcCtx.TelegramConversation.LastProcessedUpdateID(context.Background())
	if err != nil {
		return err
	}
	groups, err := cfg.Telegram.AllowedGroupIDs()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.groups = len(groups)
	c.telegramClient = telegramClient
	c.run = func(ctx context.Context) error {
		return telegramClient.Run(ctx, lastUpdateID, service.Handle)
	}
	c.mu.Unlock()
	return nil
}

func (c *Custom) ValidateConfig(cfg config.Config) error {
	if err := cfg.Telegram.Validate(); err != nil {
		return err
	}
	if err := cfg.Conversation.Validate(); err != nil {
		return err
	}
	if cfg.Telegram.Enabled() {
		if strings.TrimSpace(cfg.Conversation.APIKey) == "" {
			return errors.New("AgentRazor API key is required when Telegram is enabled")
		}
		if _, err := conversation.NewClient(conversation.Config{
			BaseURL: strings.TrimRight(strings.TrimSpace(cfg.Conversation.BaseURL), "/"),
			APIKey:  strings.TrimSpace(cfg.Conversation.APIKey),
			HTTPClient: &http.Client{
				Timeout: time.Duration(cfg.Conversation.TimeoutSeconds) * time.Second,
			},
		}); err != nil {
			return err
		}
		if _, err := telegrambot.New(cfg.Telegram, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Custom) Start() {
	c.mu.Lock()
	c.started = true
	run := c.run
	if run == nil || c.running {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	c.cancel = cancel
	c.done = done
	c.running = true
	c.mu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			c.mu.Lock()
			c.running = false
			c.cancel = nil
			c.done = nil
			c.mu.Unlock()
		}()
		if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logx.Errorf("Telegram chat bot stopped: %v", err)
		}
	}()
}

func (c *Custom) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	done := c.done
	c.cancel = nil
	c.done = nil
	c.running = false
	c.run = nil
	c.telegramClient = nil
	c.botName = ""
	c.groups = 0
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			logx.Error("timed out stopping Telegram chat bot")
		}
	}
}

func (c *Custom) Reload() error {
	c.Stop()
	if err := c.Init(); err != nil {
		return err
	}
	c.mu.RLock()
	started := c.started
	c.mu.RUnlock()
	if started {
		c.Start()
	}
	return nil
}

func (c *Custom) Status() svc.RuntimeInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	username := c.botName
	if c.telegramClient != nil {
		username = c.telegramClient.Bot().Username
	}
	return svc.RuntimeInfo{Running: c.running, BotUsername: username, AllowedGroups: c.groups}
}
