package email

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"

	manageemailmodel "github.com/jzero-io/agentrazor/server/internal/model/manage_email"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/email"
)

type SaveConfig struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveConfig(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveConfig {
	return &SaveConfig{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveConfig) SaveConfig(req *types.SaveConfigRequest) (resp *types.SaveConfigResponse, err error) {
	config, err := l.svcCtx.Model.ManageEmail.FindOneByCondition(l.ctx, nil)
	if errors.Is(err, manageemailmodel.ErrNotFound) {
		if strings.TrimSpace(req.Password) == "" {
			return nil, errors.New("首次配置必须填写 SMTP 密码或授权码")
		}
		config = &manageemailmodel.ManageEmail{Uuid: uuid.NewString()}
		applyRequest(config, req)
		if err := l.svcCtx.Model.ManageEmail.InsertV2(l.ctx, nil, config); err != nil {
			return nil, err
		}
		return &types.SaveConfigResponse{}, nil
	}
	if err != nil {
		return nil, err
	}

	applyRequest(config, req)
	if err := l.svcCtx.Model.ManageEmail.Update(l.ctx, nil, config); err != nil {
		return nil, err
	}
	return &types.SaveConfigResponse{}, nil
}

func applyRequest(config *manageemailmodel.ManageEmail, req *types.SaveConfigRequest) {
	config.From = strings.TrimSpace(req.From)
	config.Host = strings.TrimSpace(req.Host)
	config.Port = req.Port
	config.Username = strings.TrimSpace(req.Username)
	if req.Password != "" {
		config.Password = req.Password
	}
	config.EnableSsl = cast.ToInt64(req.EnableSsl)
	config.IsVerify = cast.ToInt64(req.IsVerify)
}
