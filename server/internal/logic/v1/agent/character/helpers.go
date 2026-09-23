package character

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/character"
)

const characterUserNameConstraint = "uk_agent_character_user_name"

var (
	errCharacterNotFound = errors.New("角色不存在")
	errCharacterExists   = errors.New("角色名称已存在")
)

func currentUserUUID(ctx context.Context) (string, error) {
	user, err := auth.Info(ctx)
	if err != nil {
		return "", err
	}
	return user.Uuid, nil
}

func findOwnedCustomCharacter(ctx context.Context, svcCtx *svc.ServiceContext, userUUID, characterUUID string) (*agentcharactermodel.AgentCharacter, error) {
	row, err := svcCtx.Model.AgentCharacter.FindOneByCondition(ctx, nil, condition.NewChain().
		Equal(agentcharactermodel.Uuid, strings.TrimSpace(characterUUID)).
		Equal(agentcharactermodel.UserUuid, userUUID).
		Equal(agentcharactermodel.IsBuiltin, false).
		Build()...)
	if errors.Is(err, agentcharactermodel.ErrNotFound) {
		return nil, errCharacterNotFound
	}
	return row, err
}

func ensureCharacterNameUnique(ctx context.Context, svcCtx *svc.ServiceContext, userUUID, name, excludeUUID string) error {
	chain := condition.NewChain().
		Equal(agentcharactermodel.UserUuid, userUUID).
		Equal(agentcharactermodel.Name, name)
	if excludeUUID != "" {
		chain = chain.NotEqual(agentcharactermodel.Uuid, excludeUUID)
	}
	_, err := svcCtx.Model.AgentCharacter.FindOneByCondition(ctx, nil, chain.Build()...)
	if err == nil {
		return errCharacterExists
	}
	if errors.Is(err, agentcharactermodel.ErrNotFound) {
		return nil
	}
	return err
}

func normalizeCharacterWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == characterUserNameConstraint {
		return errCharacterExists
	}
	return err
}

func toCharacter(row *agentcharactermodel.AgentCharacter) types.Character {
	return types.Character{
		Id:          row.Uuid,
		Name:        row.Name,
		Description: row.Description,
		Prompt:      row.Prompt,
		Builtin:     row.IsBuiltin,
		CreatedAt:   row.CreateTime.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   row.UpdateTime.UTC().Format(time.RFC3339Nano),
	}
}

func customOwner(userUUID string) sql.NullString {
	return sql.NullString{String: userUUID, Valid: true}
}
