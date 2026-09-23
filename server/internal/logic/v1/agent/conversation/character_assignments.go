package conversation

import (
	"context"

	"github.com/jzero-io/jzero/core/stores/condition"

	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

func characterAssignments(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string) (map[string]string, error) {
	rows, err := svcCtx.Model.Conversation.FindFieldsByCondition(ctx, nil,
		[]condition.Field{conversationmodel.Id, conversationmodel.CharacterUuid},
		condition.NewChain().Equal(conversationmodel.UserUuid, userUUID).Build()...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, row := range rows {
		if row.CharacterUuid.Valid {
			result[row.Id] = row.CharacterUuid.String
		}
	}
	return result, nil
}
