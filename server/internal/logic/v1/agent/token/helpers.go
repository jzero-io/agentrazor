package token

import (
	"context"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

func currentUserUUID(ctx context.Context) (string, error) {
	info, err := auth.Info(ctx)
	if err != nil {
		return "", err
	}
	return info.Uuid, nil
}

func isSuperAdmin(ctx context.Context, svcCtx *svc.ServiceContext) (bool, error) {
	info, err := auth.Info(ctx)
	if err != nil {
		return false, err
	}
	if len(info.RoleUuids) == 0 {
		return false, nil
	}
	roles, err := svcCtx.Model.ManageRole.FindByCondition(ctx, nil, condition.NewChain().
		In(manage_role.Uuid, info.RoleUuids).
		Build()...)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.Code == "R_SUPER" {
			return true, nil
		}
	}
	return false, nil
}
