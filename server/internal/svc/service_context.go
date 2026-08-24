package svc

import (
	"net/http"

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
	agentOptions := svcCtx.AgentOptionsFromConfig(cc.MustGetConfig().Agent)
	runtimeFactory := func() (agent.ThreadRuntime, error) {
		return agent.NewCodexAppServerRuntime(agentOptions)
	}
	runtime, err := runtimeFactory()
	if err != nil {
		panic(err)
	}
	svcCtx.AgentThreads = agent.NewThreadService(runtime, runtimeFactory)
	svcCtx.installAgentTokenUsageRecorder()
	return svcCtx
}
