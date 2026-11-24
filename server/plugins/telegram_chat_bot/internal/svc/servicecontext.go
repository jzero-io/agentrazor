package svc

import (
	"sync"

	coresvc "github.com/jzero-io/agentrazor/core-engine/svc"
	"github.com/jzero-io/jzero/core/configcenter"

	"telegram_chat_bot/internal/config"
	pluginconfigurationmodel "telegram_chat_bot/internal/model/pluginconfiguration"
	telegramconversationmodel "telegram_chat_bot/internal/model/telegramconversation"
)

type RuntimeReloader interface {
	ValidateConfig(config.Config) error
	Reload() error
}

type RuntimeInfo struct {
	Running       bool
	BotUsername   string
	AllowedGroups int
}

type RuntimeStatusProvider interface {
	Status() RuntimeInfo
}

type ServiceContext struct {
	*coresvc.ServiceContext
	ConfigCenter         configcenter.ConfigCenter[config.Config]
	PluginConfiguration  *pluginconfigurationmodel.Model
	TelegramConversation *telegramconversationmodel.Model
	Middleware
	runtimeMu sync.RWMutex
	runtime   RuntimeReloader
}

func NewServiceContext(cc configcenter.ConfigCenter[config.Config], coreSvcCtx *coresvc.ServiceContext) *ServiceContext {
	svcCtx := &ServiceContext{
		ServiceContext: coreSvcCtx,
		ConfigCenter:   cc,
	}

	svcCtx.SetConfigListener()
	svcCtx.PluginConfiguration = pluginconfigurationmodel.NewModel(svcCtx.SqlxConn)
	svcCtx.TelegramConversation = telegramconversationmodel.NewModel(svcCtx.SqlxConn)
	svcCtx.Middleware = svcCtx.NewMiddleware()
	return svcCtx
}

func (sc *ServiceContext) SetRuntimeReloader(runtime RuntimeReloader) {
	sc.runtimeMu.Lock()
	defer sc.runtimeMu.Unlock()
	sc.runtime = runtime
}

func (sc *ServiceContext) ValidateRuntimeConfig(cfg config.Config) error {
	sc.runtimeMu.RLock()
	runtime := sc.runtime
	sc.runtimeMu.RUnlock()
	if runtime == nil {
		return nil
	}
	return runtime.ValidateConfig(cfg)
}

func (sc *ServiceContext) ReloadRuntime() error {
	sc.runtimeMu.RLock()
	runtime := sc.runtime
	sc.runtimeMu.RUnlock()
	if runtime == nil {
		return nil
	}
	return runtime.Reload()
}

func (sc *ServiceContext) RuntimeStatus() RuntimeInfo {
	sc.runtimeMu.RLock()
	runtime := sc.runtime
	sc.runtimeMu.RUnlock()
	if provider, ok := runtime.(RuntimeStatusProvider); ok {
		return provider.Status()
	}
	return RuntimeInfo{}
}
