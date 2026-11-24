package telegram_chat_bot_admin

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	adminlogic "telegram_chat_bot/internal/logic/admin"
	"telegram_chat_bot/internal/svc"
	types "telegram_chat_bot/internal/types"
)

type GetConfiguration struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetConfiguration(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetConfiguration {
	return &GetConfiguration{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetConfiguration) GetConfiguration() (resp *types.GetConfigurationResponse, err error) {
	return adminlogic.GetConfiguration(l.ctx, l.svcCtx)
}
