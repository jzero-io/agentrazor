package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	manage_usermodel "github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_user_role"
	"github.com/jzero-io/agentrazor/server/internal/service/loginlock"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

const (
	enabledRoleStatus = "1"
	enabledUserStatus = "1"
)

var (
	errInvalidCredentials = errors.New("用户名或密码错误")
	errRoleDisabled       = errors.New("用户角色已被禁用")
	errUserDisabled       = errors.New("用户已被禁用")
)

func ensureUserEnabled(status string) error {
	if status != enabledUserStatus {
		return errUserDisabled
	}
	return nil
}

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
		return nil, errRoleDisabled
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
		return nil, errRoleDisabled
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
	username := strings.TrimSpace(req.Username)
	user, err := l.svcCtx.Model.ManageUser.FindOneByUsername(l.ctx, nil, username)
	if err != nil {
		if errors.Is(err, manage_usermodel.ErrNotFound) {
			return nil, errInvalidCredentials
		}
		return nil, err
	}

	locked, err := l.svcCtx.LoginLock.IsLocked(l.ctx, user.Uuid)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, loginlock.ErrLocked
	}
	if req.Password != user.Password {
		locked, err = l.svcCtx.LoginLock.RecordFailure(l.ctx, user.Uuid)
		if err != nil {
			return nil, err
		}
		if locked {
			return nil, loginlock.ErrLocked
		}
		return nil, errInvalidCredentials
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

	token, refreshToken, err := issueLoginTokenPair(l.ctx, l.svcCtx, user.Uuid, claims)
	if err != nil {
		return nil, err
	}

	return &types.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}
