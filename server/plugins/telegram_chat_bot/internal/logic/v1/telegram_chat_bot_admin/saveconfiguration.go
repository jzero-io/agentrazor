package telegram_chat_bot_admin

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	adminlogic "telegram_chat_bot/internal/logic/admin"
	"telegram_chat_bot/internal/svc"
	types "telegram_chat_bot/internal/types"
)

type SaveConfiguration struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveConfiguration(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveConfiguration {
	return &SaveConfiguration{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveConfiguration) SaveConfiguration(req *types.SaveConfigurationRequest) (resp *types.SaveConfigurationResponse, err error) {
	return adminlogic.SaveConfiguration(l.ctx, l.svcCtx, req)
}
