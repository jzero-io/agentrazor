package cmd

import (
	"net/http"

	coresvc "github.com/jzero-io/agentrazor/core-engine/svc"
	"github.com/jzero-io/jzero/core/configcenter"
	"github.com/jzero-io/jzero/core/configcenter/subscriber"
	"github.com/jzero-io/jzero/core/swaggerv2"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"

	"telegram_chat_bot/internal/config"
	"telegram_chat_bot/internal/custom"
	"telegram_chat_bot/internal/global"
	"telegram_chat_bot/internal/handler"
	"telegram_chat_bot/internal/middleware"
	"telegram_chat_bot/internal/svc"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "telegram_chat_bot server",
	Long:  "telegram_chat_bot server",
	Run: func(cmd *cobra.Command, args []string) {
		cc := configcenter.MustNewConfigCenter[config.Config](configcenter.Config{
			Type: "yaml",
		}, subscriber.MustNewFsnotifySubscriber(cmd.Flag("config").Value.String(), subscriber.WithUseEnv(true)))

		// set up logger
		logx.Must(logx.SetUp(cc.MustGetConfig().Log.LogConf))

		// print version
		printVersion()

		// create service context
		coreSvcCtx := coresvc.NewServiceContext(cc.MustGetConfig().Config, func(*http.Request) string { return "" })
		svcCtx := svc.NewServiceContext(cc, coreSvcCtx)
		global.ServiceContext = svcCtx

		// create rest server
		restServer := rest.MustNewServer(svcCtx.ConfigCenter.MustGetConfig().Rest.RestConf)
		// create custom server
		customServer := custom.New(svcCtx)
		svcCtx.SetRuntimeReloader(customServer)
		logx.Must(customServer.Init())

		// register auto generated routes
		handler.RegisterHandlers(restServer, svcCtx)
		// register swagger routes
		swaggerv2.RegisterRoutes(restServer)
		// register middleware
		middleware.Register(restServer)

		group := service.NewServiceGroup()
		group.Add(restServer)
		group.Add(customServer)

		logx.Infof("Starting rest server at %s:%d...", cc.MustGetConfig().Rest.Host, cc.MustGetConfig().Rest.Port)
		group.Start()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
