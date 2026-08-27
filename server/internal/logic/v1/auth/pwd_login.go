package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_user_role"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

func CreateToken(secret string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

const enabledRoleStatus = "1"

var ErrRoleDisabled = errors.New("用户角色已被禁用")

func enabledRoleUuidsByUser(ctx context.Context, svcCtx *svc.ServiceContext, userUuid string) ([]string, error) {
	userRoles, err := svcCtx.Model.ManageUserRole.FindByCondition(ctx, nil, condition.NewChain().
		Equal(manage_user_role.UserUuid, userUuid).
		Build()...)
	if err != nil {
		return nil, err
	}

	roleUuids := make([]string, 0, len(userRoles))
	for _, userRole := range userRoles {
		roleUuids = append(roleUuids, userRole.RoleUuid)
	}
	return enabledRoleUuids(ctx, svcCtx, roleUuids)
}

func enabledRoleUuids(ctx context.Context, svcCtx *svc.ServiceContext, roleUuids []string) ([]string, error) {
	if len(roleUuids) == 0 {
		return nil, ErrRoleDisabled
	}

	roles, err := svcCtx.Model.ManageRole.FindByCondition(ctx, nil, condition.NewChain().
		In(manage_role.Uuid, roleUuids).
		Equal(manage_role.Status, enabledRoleStatus).
		Build()...)
	if err != nil {
		return nil, err
	}

	enabledRoleUuids := make([]string, 0, len(roles))
	for _, role := range roles {
		enabledRoleUuids = append(enabledRoleUuids, role.Uuid)
	}
	if len(enabledRoleUuids) == 0 {
		return nil, ErrRoleDisabled
	}
	return enabledRoleUuids, nil
}

type PwdLogin struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewPwdLogin(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *PwdLogin {
	return &PwdLogin{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *PwdLogin) PwdLogin(req *types.PwdLoginRequest) (resp *types.LoginResponse, err error) {
	config, err := l.svcCtx.ConfigCenter.GetConfig()
	if err != nil {
		return nil, err
	}

	user, err := l.svcCtx.Model.ManageUser.FindOneByUsername(l.ctx, nil, req.Username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}
	if req.Password != user.Password {
		return nil, errors.New("用户名或密码错误")
	}
	if err := ensureUserEnabled(user.Status); err != nil {
		return nil, err
	}
	roleIds, err := enabledRoleUuidsByUser(l.ctx, l.svcCtx, user.Uuid)
	if err != nil {
		return nil, err
	}

	marshal, err := json.Marshal(auth.Auth{
		Uuid:      user.Uuid,
		Username:  user.Username,
		RoleUuids: roleIds,
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
