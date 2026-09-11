package middleware

import (
	"context"

	coremiddleware "github.com/jzero-io/agentrazor/core-engine/middleware"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/server/internal/global"
)

func Register(server *rest.Server) {
	httpx.SetOkHandler(okHandler)
	httpx.SetErrorHandlerCtx(global.ServiceContext.Error)
	httpx.SetValidator(global.ServiceContext.Validate)
	server.Use(global.ServiceContext.I18n)
}

func okHandler(_ context.Context, data any) any {
	return coremiddleware.Body{
		Data: data,
		Code: 0,
		Msg:  "success",
	}
}

func NewMiddleware(svcCtx *svc.ServiceContext) svc.Middleware {
	return svc.Middleware{
		Agent: NewAgentMiddleware(svcCtx).Handle,
	}
}
