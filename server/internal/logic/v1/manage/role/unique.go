package role

import (
	"context"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

func ensureRoleUnique(ctx context.Context, svcCtx *svc.ServiceContext, roleName, roleCode, excludeUuid string) error {
	if err := ensureRoleFieldUnique(ctx, svcCtx, manage_role.Code, roleCode, excludeUuid, "角色编码已存在"); err != nil {
		return err
	}
	return ensureRoleFieldUnique(ctx, svcCtx, manage_role.Name, roleName, excludeUuid, "角色名称已存在")
}

func ensureRoleFieldUnique(ctx context.Context, svcCtx *svc.ServiceContext, field condition.Field, value, excludeUuid, message string) error {
	conditions := condition.NewChain().Equal(field, value).Build()
	if excludeUuid != "" {
		conditions = append(conditions, condition.NewChain().NotEqual(manage_role.Uuid, excludeUuid).Build()...)
	}

	if _, err := svcCtx.Model.ManageRole.FindOneByCondition(ctx, nil, conditions...); err == nil {
		return errors.New(message)
	} else if !errors.Is(err, manage_role.ErrNotFound) {
		return err
	}

	return nil
}
