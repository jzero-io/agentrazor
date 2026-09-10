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

	"github.com/jzero-io/agentrazor/server/internal/model/manage_menu"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_role_menu"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/role"
)

func ensureRoleUnique(ctx context.Context, model manage_role.ManageRoleModel, roleName, roleCode, excludeUuid string) error {
	if err := ensureRoleFieldUnique(ctx, model, manage_role.Code, roleCode, excludeUuid, "角色编码已存在"); err != nil {
		return err
	}
	return ensureRoleFieldUnique(ctx, model, manage_role.Name, roleName, excludeUuid, "角色名称已存在")
}

func ensureRoleFieldUnique(ctx context.Context, model manage_role.ManageRoleModel, field condition.Field, value, excludeUuid, message string) error {
	conditions := condition.NewChain().Equal(field, value).Build()
	if excludeUuid != "" {
		conditions = append(conditions, condition.NewChain().NotEqual(manage_role.Uuid, excludeUuid).Build()...)
	}

	if _, err := model.FindOneByCondition(ctx, nil, conditions...); err == nil {
		return errors.New(message)
	} else if !errors.Is(err, manage_role.ErrNotFound) {
		return err
	}
	return nil
}

type Add struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewAdd(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Add {
	return &Add{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *Add) Add(req *types.AddRequest) (resp *types.AddResponse, err error) {
	if err := ensureRoleUnique(l.ctx, l.svcCtx.Model.ManageRole, req.RoleName, req.RoleCode, ""); err != nil {
		return nil, err
	}

	// find home menu
	var homeMenuUuid string
	if home, err := l.svcCtx.Model.ManageMenu.FindOneByCondition(l.ctx, nil, condition.NewChain().Equal(manage_menu.RoutePath, "/home").Build()...); err != nil {
		return nil, err
	} else {
		homeMenuUuid = home.Uuid
	}

	err = l.svcCtx.SqlxConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		roleUuid := uuid.New().String()
		if err = l.svcCtx.Model.ManageRole.InsertV2(l.ctx, session, &manage_role.ManageRole{
			Uuid:   roleUuid,
			Code:   req.RoleCode,
			Name:   req.RoleName,
			Desc:   req.RoleDesc,
			Status: req.Status,
		}); err != nil {
			return err
		}

		// 添加首页路由
		if err = l.svcCtx.Model.ManageRoleMenu.InsertV2(l.ctx, session, &manage_role_menu.ManageRoleMenu{
			Uuid:     uuid.New().String(),
			RoleUuid: roleUuid,
			MenuUuid: homeMenuUuid,
			IsHome:   cast.ToInt64(true),
		}); err != nil {
			return err
		}

		return nil
	})

	return
}
