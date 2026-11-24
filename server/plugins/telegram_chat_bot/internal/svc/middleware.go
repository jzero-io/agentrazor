package svc

import "github.com/zeromicro/go-zero/rest"

type Middleware struct {
	Authx rest.Middleware
}

func (svcCtx *ServiceContext) NewMiddleware() Middleware {
	return Middleware{Authx: svcCtx.ServiceContext.Authx}
}
