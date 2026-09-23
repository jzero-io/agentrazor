package character

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/character"
)

const maxCharacterPromptBytes = 16000
const builtinCharacterNameConstraint = "uk_agent_character_builtin_name"

var (
	errBuiltinCharacterNotFound = errors.New("内置角色不存在")
	errBuiltinCharacterExists   = errors.New("内置角色名称已存在")
)

func normalizeCharacterInput(name, description, prompt string) (string, string, string, error) {
	name, description, prompt = strings.TrimSpace(name), strings.TrimSpace(description), strings.TrimSpace(prompt)
	if name == "" {
		return "", "", "", errors.New("角色名称不能为空")
	}
	if prompt == "" {
		return "", "", "", errors.New("角色提示词不能为空")
	}
	if len([]byte(prompt)) > maxCharacterPromptBytes {
		return "", "", "", errors.New("角色提示词不能超过 16000 字节")
	}
	return name, description, prompt, nil
}

func findBuiltinCharacter(ctx context.Context, svcCtx *svc.ServiceContext, characterID string) (*agentcharactermodel.AgentCharacter, error) {
	row, err := svcCtx.Model.AgentCharacter.FindOneByCondition(ctx, nil, condition.NewChain().
		Equal(agentcharactermodel.Uuid, strings.TrimSpace(characterID)).
		Equal(agentcharactermodel.IsBuiltin, true).
		Build()...)
	if errors.Is(err, agentcharactermodel.ErrNotFound) {
		return nil, errBuiltinCharacterNotFound
	}
	return row, err
}

func ensureBuiltinNameUnique(ctx context.Context, svcCtx *svc.ServiceContext, name, excludeID string) error {
	chain := condition.NewChain().Equal(agentcharactermodel.IsBuiltin, true).Equal(agentcharactermodel.Name, name)
	if excludeID != "" {
		chain = chain.NotEqual(agentcharactermodel.Uuid, excludeID)
	}
	_, err := svcCtx.Model.AgentCharacter.FindOneByCondition(ctx, nil, chain.Build()...)
	if err == nil {
		return errBuiltinCharacterExists
	}
	if errors.Is(err, agentcharactermodel.ErrNotFound) {
		return nil
	}
	return err
}

func normalizeWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == builtinCharacterNameConstraint {
		return errBuiltinCharacterExists
	}
	return err
}

func toCharacter(row *agentcharactermodel.AgentCharacter, owner *types.Owner) types.Character {
	return types.Character{
		Id: row.Uuid, Name: row.Name, Description: row.Description, Prompt: row.Prompt,
		Builtin: row.IsBuiltin, Sort: row.Sort, Owner: owner,
		CreatedAt: row.CreateTime.UTC().Format(time.RFC3339Nano), UpdatedAt: row.UpdateTime.UTC().Format(time.RFC3339Nano),
	}
}

func builtinOwner() sql.NullString { return sql.NullString{} }
