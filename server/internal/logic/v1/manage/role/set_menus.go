package role

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/jzero-io/agentrazor/server/internal/logic/v1/manage/menu"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_menu"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_role_menu"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	menu_types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/menu"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/role"
)

const (
	agentSaveSettingsMenuUUID = "e110f1d2-8d73-4ff9-9702-4f7395b7a001"
	agentChatGPTLoginMenuUUID = "e110f1d2-8d73-4ff9-9702-4f7395b7a002"
	agentAPIKeyLoginMenuUUID  = "e110f1d2-8d73-4ff9-9702-4f7395b7a003"
)

type SetMenus struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSetMenus(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SetMenus {
	return &SetMenus{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *SetMenus) SetMenus(req *types.SetMenusRequest) (resp *types.SetMenusResponse, err error) {
	req.MenuUuids = normalizeMenuDependencies(req.MenuUuids)

	if err = l.svcCtx.SqlxConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		// 找到该角色的首页
		roleHomeMenu, err := l.svcCtx.Model.ManageRoleMenu.FindOneByCondition(l.ctx, nil, condition.NewChain().
			Equal(manage_role_menu.RoleUuid, req.RoleUuid).
			Equal(manage_role_menu.IsHome, cast.ToInt(true)).
			Build()...)
		if err != nil {
			return errors.New("该角色无首页路由")
		}
		var datas []*manage_role_menu.ManageRoleMenu

		for _, v := range req.MenuUuids {
			data := &manage_role_menu.ManageRoleMenu{
				Uuid:     uuid.New().String(),
				RoleUuid: req.RoleUuid,
				MenuUuid: v,
			}
			if data.MenuUuid == roleHomeMenu.MenuUuid {
				data.IsHome = cast.ToInt64(true)
			}
			datas = append(datas, data)
		}

		if err = l.svcCtx.Model.ManageRoleMenu.DeleteByCondition(l.ctx, session, condition.Condition{
			Field:    manage_role_menu.RoleUuid,
			Operator: condition.Equal,
			Value:    req.RoleUuid,
		}); err != nil {
			return err
		}
		if len(datas) > 0 {
			if err = l.svcCtx.Model.ManageRoleMenu.BulkInsert(l.ctx, session, datas); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// update casbin_rule
	_, err = l.svcCtx.CasbinEnforcer.RemoveFilteredPolicy(0, req.RoleUuid)
	if err != nil {
		return nil, errors.New("fail to remove filtered policy: " + err.Error())
	}
	// load policies
	err = l.svcCtx.CasbinEnforcer.LoadPolicy()
	if err != nil {
		return nil, errors.New("fail to load policy: " + err.Error())
	}

	// add casbin_rule
	var newPolicies [][]string
	policyCodes := make(map[string]struct{})
	// get menu perms
	menus, err := l.svcCtx.Model.ManageMenu.FindByCondition(l.ctx, nil, condition.NewChain().
		In(manage_menu.Uuid, req.MenuUuids).
		Build()...)
	if err != nil {
		return nil, err
	}
	for _, v := range menus {
		var permissions []menu_types.Permission
		menu.Unmarshal(v.Permissions, &permissions)
		for _, perm := range permissions {
			if _, ok := policyCodes[perm.Code]; ok {
				continue
			}
			policyCodes[perm.Code] = struct{}{}
			newPolicies = append(newPolicies, []string{req.RoleUuid, perm.Code})
		}
	}

	var b bool
	if len(newPolicies) > 0 {
		b, err = l.svcCtx.CasbinEnforcer.AddPolicies(newPolicies)
		if err != nil {
			return nil, errors.New("fail to add policies: " + err.Error())
		}
		if !b {
			return nil, errors.New("fail to add policies")
		}
		// load policies
		err = l.svcCtx.CasbinEnforcer.LoadPolicy()
	}
	return
}

func normalizeMenuDependencies(menuUUIDs []string) []string {
	seen := make(map[string]struct{}, len(menuUUIDs)+1)
	normalized := make([]string, 0, len(menuUUIDs)+1)
	requiresSaveSettings := false

	for _, menuUUID := range menuUUIDs {
		if _, ok := seen[menuUUID]; ok {
			continue
		}
		seen[menuUUID] = struct{}{}
		normalized = append(normalized, menuUUID)

		if menuUUID == agentChatGPTLoginMenuUUID || menuUUID == agentAPIKeyLoginMenuUUID {
			requiresSaveSettings = true
		}
	}

	if requiresSaveSettings {
		if _, ok := seen[agentSaveSettingsMenuUUID]; !ok {
			normalized = append(normalized, agentSaveSettingsMenuUUID)
		}
	}

	return normalized
}
