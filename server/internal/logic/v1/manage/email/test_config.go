package email

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/mailer"
	manageemailmodel "github.com/jzero-io/agentrazor/server/internal/model/manage_email"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/email"
)

type TestConfig struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewTestConfig(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *TestConfig {
	return &TestConfig{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *TestConfig) TestConfig(req *types.TestConfigRequest) (resp *types.TestConfigResponse, err error) {
	config, err := l.svcCtx.Model.ManageEmail.FindOneByCondition(l.ctx, nil)
	if errors.Is(err, manageemailmodel.ErrNotFound) {
		return nil, errors.New("请先保存邮箱配置")
	}
	if err != nil {
		return nil, err
	}

	if err := mailer.Send(mailer.SMTPConfig{
		From:       config.From,
		Host:       config.Host,
		Port:       cast.ToInt(config.Port),
		Username:   config.Username,
		Password:   config.Password,
		EnableSSL:  cast.ToBool(config.EnableSsl),
		VerifyCert: cast.ToBool(config.IsVerify),
	}, req.Recipient, "AgentRazor 邮箱配置测试", "这是一封来自 AgentRazor 的测试邮件，邮件发送功能工作正常。"); err != nil {
		l.Errorf("test SMTP configuration: %v", err)
		return nil, errors.New("测试邮件发送失败，请检查 SMTP 配置和服务端日志")
	}

	return &types.TestConfigResponse{}, nil
}
