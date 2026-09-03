package group

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jzero-io/jzero/core/stores/condition"

	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

const conversationGroupUserNameConstraint = "uk_conversation_group_user_name"

var errGroupNameExists = errors.New("分组名称已存在")

func ensureGroupNameUnique(ctx context.Context, svcCtx *svc.ServiceContext, userUUID, name, excludeUUID string) error {
	chain := condition.NewChain().
		Equal(conversationgroupmodel.UserUuid, userUUID).
		Equal(conversationgroupmodel.Name, name)
	if excludeUUID != "" {
		chain = chain.NotEqual(conversationgroupmodel.Uuid, excludeUUID)
	}

	_, err := svcCtx.Model.ConversationGroup.FindOneByCondition(ctx, nil, chain.Build()...)
	if err == nil {
		return errGroupNameExists
	}
	if errors.Is(err, conversationgroupmodel.ErrNotFound) {
		return nil
	}
	return err
}

func normalizeGroupWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == conversationGroupUserNameConstraint {
		return errGroupNameExists
	}
	return err
}
