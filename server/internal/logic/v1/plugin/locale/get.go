package locale

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/plugin/locale"
)

type Get struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGet(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Get {
	return &Get{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Get) Get() (resp *types.GetResponse, err error) {
	locales := l.svcCtx.PluginRegistry.AdminLocales()
	return &types.GetResponse{
		Locales: locales,
	}, nil
}
