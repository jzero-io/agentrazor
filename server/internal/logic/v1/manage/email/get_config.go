package email

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"

	manageemailmodel "github.com/jzero-io/agentrazor/server/internal/model/manage_email"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/email"
)

type GetConfig struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetConfig(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetConfig {
	return &GetConfig{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetConfig) GetConfig(req *types.GetConfigRequest) (resp *types.GetConfigResponse, err error) {
	config, err := l.svcCtx.Model.ManageEmail.FindOneByCondition(l.ctx, nil)
	if errors.Is(err, manageemailmodel.ErrNotFound) {
		return &types.GetConfigResponse{Config: types.EmailConfig{Port: 465, EnableSsl: true, IsVerify: true}}, nil
	}
	if err != nil {
		return nil, err
	}

	return &types.GetConfigResponse{
		Configured: true,
		Config: types.EmailConfig{
			From:        config.From,
			Host:        config.Host,
			Port:        config.Port,
			Username:    config.Username,
			EnableSsl:   cast.ToBool(config.EnableSsl),
			IsVerify:    cast.ToBool(config.IsVerify),
			HasPassword: config.Password != "",
		},
	}, nil
}
