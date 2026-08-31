package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/constant"
	manage_usermodel "github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

type CodeLogin struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

var ErrEmailOrVerificationCode = errors.New("邮箱或验证码错误")

func NewCodeLogin(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *CodeLogin {
	return &CodeLogin{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *CodeLogin) CodeLogin(req *types.CodeLoginRequest) (resp *types.LoginResponse, err error) {
	config, err := l.svcCtx.ConfigCenter.GetConfig()
	if err != nil {
		return nil, err
	}

	// check verificationUuid
	var verificationUuidVal string
	if err = l.svcCtx.Cache.Get(fmt.Sprintf("%s:%s", constant.CacheVerificationCodePrefix, req.VerificationUuid), &verificationUuidVal); err != nil {
		l.Errorf("get verification code: %v", err)
		return nil, ErrEmailOrVerificationCode
	}
	if verificationUuidVal != req.VerificationCode {
		return nil, ErrEmailOrVerificationCode
	}

	user, err := l.svcCtx.Model.ManageUser.FindOneByCondition(l.ctx, nil, condition.NewChain().
		Equal(manage_usermodel.Email, req.Email).
		Build()...)
	if err != nil {
		if !errors.Is(err, manage_usermodel.ErrNotFound) {
			l.Errorf("find user by email: %v", err)
		}
		return nil, ErrEmailOrVerificationCode
	}
	if err := ensureUserEnabled(user.Status); err != nil {
		return nil, err
	}

	roleUuids, err := enabledRoleUuidsByUser(l.ctx, l.svcCtx, user.Uuid)
	if err != nil {
		return nil, err
	}

	marshal, err := json.Marshal(auth.Auth{
		Uuid:      user.Uuid,
		Username:  user.Username,
		RoleUuids: roleUuids,
	})
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	err = json.Unmarshal(marshal, &claims)
	if err != nil {
		return nil, err
	}

	// token 过期时间
	expirationTime := time.Now().Add(time.Duration(config.Jwt.AccessExpire) * time.Second).Unix()
	claims["exp"] = expirationTime

	token, err := CreateToken(l.svcCtx.MustGetConfig().Jwt.AccessSecret, claims)
	if err != nil {
		return nil, err
	}

	claims["exp"] = time.Now().Add(time.Duration(config.Jwt.RefreshExpire) * time.Second).Unix()
	refreshToken, err := CreateToken(l.svcCtx.MustGetConfig().Jwt.AccessSecret, claims)
	if err != nil {
		return nil, err
	}

	return &types.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}
