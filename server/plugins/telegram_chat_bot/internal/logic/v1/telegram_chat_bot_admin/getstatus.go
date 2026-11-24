package telegram_chat_bot_admin

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	adminlogic "telegram_chat_bot/internal/logic/admin"
	"telegram_chat_bot/internal/svc"
	types "telegram_chat_bot/internal/types"
)

type GetStatus struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetStatus(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetStatus {
	return &GetStatus{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetStatus) GetStatus() (resp *types.GetStatusResponse, err error) {
	return adminlogic.GetStatus(l.ctx, l.svcCtx)
}
