package auth

import (
	"context"
	"net/http"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

type ResetPassword struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewResetPassword(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ResetPassword {
	return &ResetPassword{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *ResetPassword) ResetPassword(req *types.ResetPasswordRequest) (resp *types.ResetPasswordResponse, err error) {
	matched, err := verifyEmailCode(l.svcCtx, req.VerificationUuid, req.Email, req.VerificationCode)
	if err != nil {
		return nil, RegisterError
	}
	if !matched {
		return nil, errors.New("邮箱或验证码错误")
	}

	user, err := l.svcCtx.Model.ManageUser.FindOneByCondition(l.ctx, nil, condition.NewChain().Equal(manage_user.Email, req.Email).Build()...)
	if err != nil {
		return nil, errors.New("用户名/密码错误")
	}

	user.Password = req.Password
	err = l.svcCtx.Model.ManageUser.Update(l.ctx, nil, user)

	return
}
