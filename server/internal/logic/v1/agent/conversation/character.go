package conversation

import (
	"context"
	"errors"
	"strings"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

var errCharacterNotAvailable = errors.New("agent character not found")

func normalizeOwnedCharacter(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string, characterID *string) (string, string, error) {
	if characterID == nil || strings.TrimSpace(*characterID) == "" {
		return "", "", nil
	}
	row, err := svcCtx.Model.AgentCharacter.FindOne(ctx, nil, strings.TrimSpace(*characterID))
	if errors.Is(err, agentcharactermodel.ErrNotFound) {
		return "", "", errCharacterNotAvailable
	}
	if err != nil {
		return "", "", err
	}
	if !row.IsBuiltin && (!row.UserUuid.Valid || row.UserUuid.String != userUUID) {
		return "", "", errCharacterNotAvailable
	}
	prompt := strings.TrimSpace(row.Prompt)
	if prompt == "" {
		return "", "", errCharacterNotAvailable
	}
	return row.Uuid, prompt, nil
}
