package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"telegram_chat_bot/internal/config"
	"telegram_chat_bot/internal/pluginconfiguration"
	"telegram_chat_bot/internal/svc"
	"telegram_chat_bot/internal/types"
)

func GetConfiguration(ctx context.Context, svcCtx *svc.ServiceContext) (*types.GetConfigurationResponse, error) {
	cfg, err := svcCtx.EffectiveConfig(ctx)
	if err != nil {
		return nil, err
	}
	updatedAt := ""
	record, err := svcCtx.PluginConfiguration.Load(ctx)
	if err == nil {
		updatedAt = record.UpdateTime.UTC().Format(time.RFC3339)
	} else if !errors.Is(err, sqlx.ErrNotFound) {
		return nil, err
	}
	return &types.GetConfigurationResponse{
		Config:    responseConfiguration(cfg),
		Secrets:   configurationSecrets(cfg),
		UpdatedAt: updatedAt,
	}, nil
}

func SaveConfiguration(ctx context.Context, svcCtx *svc.ServiceContext, req *types.SaveConfigurationRequest) (*types.SaveConfigurationResponse, error) {
	if req == nil {
		return nil, errors.New("configuration is required")
	}
	user, err := auth.Info(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(user.Uuid) == "" {
		return nil, errors.New("authenticated user is required")
	}
	base, err := svcCtx.GetConfig()
	if err != nil {
		return nil, err
	}
	current, err := svcCtx.EffectiveConfig(ctx)
	if err != nil {
		return nil, err
	}
	candidate := mergeConfiguration(current, req.Config)
	if err := svcCtx.ValidateRuntimeConfig(candidate); err != nil {
		return nil, fmt.Errorf("invalid Telegram plugin configuration: %w", err)
	}

	previous, previousErr := svcCtx.PluginConfiguration.Load(ctx)
	if previousErr != nil && !errors.Is(previousErr, sqlx.ErrNotFound) {
		return nil, previousErr
	}
	encrypted, err := pluginconfiguration.Encrypt(pluginconfiguration.FromConfig(candidate), base.Jwt.AccessSecret)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.PluginConfiguration.Save(ctx, encrypted, user.Uuid); err != nil {
		return nil, err
	}
	if err := svcCtx.ReloadRuntime(); err != nil {
		if previousErr == nil {
			_ = svcCtx.PluginConfiguration.Save(ctx, previous.EncryptedValue, previous.UpdatedBy)
		} else {
			_ = svcCtx.PluginConfiguration.Delete(ctx)
		}
		_ = svcCtx.ReloadRuntime()
		return nil, fmt.Errorf("apply Telegram plugin configuration: %w", err)
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	if record, recordErr := svcCtx.PluginConfiguration.Load(ctx); recordErr == nil {
		updatedAt = record.UpdateTime.UTC().Format(time.RFC3339)
	}
	return &types.SaveConfigurationResponse{
		Config:    responseConfiguration(candidate),
		Secrets:   configurationSecrets(candidate),
		UpdatedAt: updatedAt,
	}, nil
}

func GetStatus(ctx context.Context, svcCtx *svc.ServiceContext) (*types.GetStatusResponse, error) {
	cfg, err := svcCtx.EffectiveConfig(ctx)
	if err != nil {
		return nil, err
	}
	count, err := svcCtx.TelegramConversation.CountConversations(ctx)
	if err != nil {
		return nil, err
	}
	runtime := svcCtx.RuntimeStatus()
	return &types.GetStatusResponse{
		BotRunning:        runtime.Running,
		BotUsername:       runtime.BotUsername,
		TokenConfigured:   strings.TrimSpace(cfg.Telegram.BotToken) != "",
		AllowedGroups:     runtime.AllowedGroups,
		ConversationCount: count,
	}, nil
}

func responseConfiguration(cfg config.Config) types.PluginConfiguration {
	return types.PluginConfiguration{
		Telegram: types.TelegramConfiguration{
			GroupChatIDs:          cfg.Telegram.GroupChatIDs,
			APIBaseURL:            cfg.Telegram.APIBaseURL,
			PollTimeoutSeconds:    cfg.Telegram.PollTimeoutSeconds,
			RequestTimeoutSeconds: cfg.Telegram.RequestTimeoutSeconds,
		},
		Conversation: types.ConversationConfiguration{
			BaseURL:        cfg.Conversation.BaseURL,
			TimeoutSeconds: cfg.Conversation.TimeoutSeconds,
		},
	}
}

func configurationSecrets(cfg config.Config) types.ConfigurationSecrets {
	return types.ConfigurationSecrets{
		TelegramBotToken:   strings.TrimSpace(cfg.Telegram.BotToken) != "",
		ConversationAPIKey: strings.TrimSpace(cfg.Conversation.APIKey) != "",
	}
}

func mergeConfiguration(current config.Config, input types.PluginConfiguration) config.Config {
	candidate := current
	candidate.Telegram.GroupChatIDs = input.Telegram.GroupChatIDs
	candidate.Telegram.APIBaseURL = input.Telegram.APIBaseURL
	candidate.Telegram.PollTimeoutSeconds = input.Telegram.PollTimeoutSeconds
	candidate.Telegram.RequestTimeoutSeconds = input.Telegram.RequestTimeoutSeconds
	if value := strings.TrimSpace(input.Telegram.BotToken); value != "" {
		candidate.Telegram.BotToken = value
	}
	candidate.Conversation.BaseURL = input.Conversation.BaseURL
	candidate.Conversation.TimeoutSeconds = input.Conversation.TimeoutSeconds
	if value := strings.TrimSpace(input.Conversation.APIKey); value != "" {
		candidate.Conversation.APIKey = value
	}
	return candidate
}
