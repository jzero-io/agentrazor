package svc

import (
	"context"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"telegram_chat_bot/internal/config"
	"telegram_chat_bot/internal/pluginconfiguration"
)

func (svcCtx *ServiceContext) GetConfig() (config.Config, error) {
	cfg, err := svcCtx.ConfigCenter.GetConfig()
	if err != nil {
		return config.Config{}, err
	}
	if svcCtx.ServiceContext != nil {
		cfg.Config = svcCtx.ServiceContext.Config
	}
	return cfg, nil
}

func (svcCtx *ServiceContext) MustGetConfig() config.Config {
	return svcCtx.ConfigCenter.MustGetConfig()
}

func (svcCtx *ServiceContext) EffectiveConfig(ctx context.Context) (config.Config, error) {
	base, err := svcCtx.GetConfig()
	if err != nil {
		return config.Config{}, err
	}
	record, err := svcCtx.PluginConfiguration.Load(ctx)
	if errors.Is(err, sqlx.ErrNotFound) {
		return pluginconfiguration.Defaults().Apply(base), nil
	}
	if err != nil {
		return config.Config{}, fmt.Errorf("load Telegram plugin configuration: %w", err)
	}
	settings, err := pluginconfiguration.Decrypt(record.EncryptedValue, base.Jwt.AccessSecret)
	if err != nil {
		return config.Config{}, err
	}
	return settings.Apply(base), nil
}

func (svcCtx *ServiceContext) SetConfigListener() {
	svcCtx.ConfigCenter.AddListener(func() {
		c, err := svcCtx.GetConfig()
		if err != nil {
			logx.Errorf("reload config error: %v", err)
			return
		}

		logx.Infof("reload config successfully")
		switch c.Log.Level {
		case "debug":
			logx.SetLevel(logx.DebugLevel)
		case "info":
			logx.SetLevel(logx.InfoLevel)
		case "error":
			logx.SetLevel(logx.ErrorLevel)
		case "severe":
			logx.SetLevel(logx.SevereLevel)
		}

		// add custom logic here
	})
}
