package serverless

import (
	"path/filepath"

	coresvc "github.com/jzero-io/agentrazor/core-engine/svc"
	"github.com/jzero-io/jzero/core/configcenter"
	"github.com/jzero-io/jzero/core/configcenter/subscriber"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"

	adminlocales "telegram_chat_bot/admin/locales"
	"telegram_chat_bot/internal/adminpage"
	"telegram_chat_bot/internal/config"
	"telegram_chat_bot/internal/custom"
	"telegram_chat_bot/internal/global"
	"telegram_chat_bot/internal/handler"
	"telegram_chat_bot/internal/svc"
)

type Serverless struct {
	SvcCtx        *svc.ServiceContext
	HandlerFunc   func(server *rest.Server, svcCtx *svc.ServiceContext)
	RouteCodesMap map[string]string
}

func (*Serverless) PluginLocales() map[string][]byte {
	return adminlocales.Bundles()
}

func New(coreSvcCtx *coresvc.ServiceContext) *Serverless {
	cc := configcenter.MustNewConfigCenter[config.Config](configcenter.Config{
		Type: "yaml",
	}, subscriber.MustNewFsnotifySubscriber(filepath.Join("plugins", "telegram_chat_bot", "etc", "etc.yaml"), subscriber.WithUseEnv(true)))

	svcCtx := svc.NewServiceContext(cc, coreSvcCtx)
	global.ServiceContext = svcCtx
	customServer := custom.New(svcCtx)
	svcCtx.SetRuntimeReloader(customServer)
	logx.Must(customServer.Init())

	group := service.NewServiceGroup()
	group.Add(customServer)
	group.Start()

	return &Serverless{
		SvcCtx: svcCtx,
		HandlerFunc: func(server *rest.Server, svcCtx *svc.ServiceContext) {
			handler.RegisterHandlers(server, svcCtx)
			adminpage.RegisterHandlers(server)
		},
		RouteCodesMap: map[string]string{
			"GET:/api/v1/plugin/admin/telegram_chat_bot/status":        "v1:plugin:telegramChatBot:getStatus",
			"GET:/api/v1/plugin/admin/telegram_chat_bot/configuration": "v1:plugin:telegramChatBot:getConfiguration",
			"PUT:/api/v1/plugin/admin/telegram_chat_bot/configuration": "v1:plugin:telegramChatBot:saveConfiguration",
		},
	}
}
