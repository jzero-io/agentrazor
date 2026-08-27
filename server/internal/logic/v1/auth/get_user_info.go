package auth

import (
	"context"
	"net/http"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_menu"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_role_menu"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

type GetUserInfo struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetUserInfo(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetUserInfo {
	return &GetUserInfo{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *GetUserInfo) GetUserInfo(req *types.GetUserInfoRequest) (resp *types.GetUserInfoResponse, err error) {
	info, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}

	user, err := l.svcCtx.Model.ManageUser.FindOneByUuid(l.ctx, nil, info.Uuid)
	if err != nil {
		return nil, err
	}

	if err := ensureUserEnabled(user.Status); err != nil {
		return nil, err
	}
	roleUuids, err := enabledRoleUuidsByUser(l.ctx, l.svcCtx, user.Uuid)
	if err != nil {
		return nil, err
	}

	roles, err := l.svcCtx.Model.ManageRole.FindByCondition(l.ctx, nil, condition.NewChain().
		In(manage_role.Uuid, roleUuids).
		Equal(manage_role.Status, enabledRoleStatus).
		Build()...)
	if err != nil {
		return nil, err
	}
	var roleCodes []string
	for _, role := range roles {
		roleCodes = append(roleCodes, role.Code)
	}

	// get role buttons
	roleMenus, err := l.svcCtx.Model.ManageRoleMenu.FindByCondition(l.ctx, nil, condition.NewChain().
		In(manage_role_menu.RoleUuid, roleUuids).
		Build()...)
	if err != nil {
		return nil, err
	}
	var menuUuids []string
	for _, roleMenu := range roleMenus {
		menuUuids = append(menuUuids, roleMenu.MenuUuid)
	}
	menus, err := l.svcCtx.Model.ManageMenu.FindByCondition(l.ctx, nil, condition.NewChain().
		In(manage_menu.Uuid, menuUuids).
		Equal(manage_menu.Status, "1").
		Equal(manage_menu.MenuType, "3").
		Build()...)
	if err != nil {
		return nil, err
	}
	buttons := make([]string, 0)
	for _, menu := range menus {
		buttons = append(buttons, menu.ButtonCode)
	}

	return &types.GetUserInfoResponse{
		UserUuid: user.Uuid,
		Username: user.Username,
		Roles:    roleCodes,
		Buttons:  buttons,
	}, nil
}
