package svc

import (
	"net/http"
	"time"

	"github.com/jzero-io/agentrazor/core-engine/svc"
	"github.com/jzero-io/jzero/core/configcenter"
	"github.com/jzero-io/jzero/core/stores/modelx"

	"github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/config"
	"github.com/jzero-io/agentrazor/server/internal/model"
)

type ServiceContext struct {
	*svc.ServiceContext
	ConfigCenter configcenter.ConfigCenter[config.Config]
	Model        model.Model
	AgentThreads *agent.ThreadService
	Middleware
}

func NewServiceContext(cc configcenter.ConfigCenter[config.Config], route2code func(r *http.Request) string) *ServiceContext {
	svcCtx := &ServiceContext{
		ConfigCenter: cc,
	}
	svcCtx.SetConfigListener()

	svcCtx.ServiceContext = svc.NewServiceContext(svcCtx.ConfigCenter.MustGetConfig().Config, route2code)
	svcCtx.Model = model.NewModel(svcCtx.SqlxConn, modelx.WithCachedConn(modelx.NewConnWithCache(svcCtx.SqlxConn, svcCtx.Cache)))
	svcCtx.Middleware = NewMiddleware(svcCtx)
	agentConfig := cc.MustGetConfig().Agent
	if agentConfig.Enabled {
		runtime, err := agent.NewCodexAppServerRuntime(agent.CodexAppServerOptions{
			Binary:             agentConfig.BinaryPath,
			CodexHome:          agentConfig.CodexHome,
			Workspace:          agentConfig.Workspace,
			Sandbox:            agentConfig.Sandbox,
			ServiceName:        agentConfig.ServiceName,
			DisableApps:        agentConfig.DisableApps,
			DisabledMCPServers: agentConfig.DisabledMCPServers,
			MaxEvents:          10_000,
			StartTimeout:       time.Duration(agentConfig.StartTimeoutSeconds) * time.Second,
		})
		if err != nil {
			panic(err)
		}
		svcCtx.AgentThreads = agent.NewThreadService(runtime, time.Duration(agentConfig.RunTimeoutSeconds)*time.Second)
	}
	return svcCtx
}
