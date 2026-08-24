package middleware

import (
	"github.com/jzero-io/agentrazor/server/internal/svc"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/server/internal/global"
)

func Register(server *rest.Server) {
	httpx.SetOkHandler(global.ServiceContext.Ok)
	httpx.SetErrorHandlerCtx(global.ServiceContext.Error)
	httpx.SetValidator(global.ServiceContext.Validate)
	server.Use(global.ServiceContext.I18n)
}

func NewMiddleware() svc.Middleware {
	return svc.Middleware{
		Agent: NewAgentMiddleware().Handle,
	}
}
