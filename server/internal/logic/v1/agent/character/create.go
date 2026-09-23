package character

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/character"
)

type Create struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewCreate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Create {
	return &Create{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *Create) Create(req *types.CreateRequest) (*types.Character, error) {
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	name, description, prompt := strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), strings.TrimSpace(req.Prompt)
	if name == "" {
		return nil, errors.New("角色名称不能为空")
	}
	if prompt == "" {
		return nil, errors.New("角色提示词不能为空")
	}
	if err := ensureCharacterNameUnique(l.ctx, l.svcCtx, userUUID, name, ""); err != nil {
		return nil, err
	}
	row := &agentcharactermodel.AgentCharacter{
		Uuid: uuid.NewString(), UserUuid: customOwner(userUUID), Name: name,
		Description: description, Prompt: prompt, IsBuiltin: false,
	}
	if err := l.svcCtx.Model.AgentCharacter.InsertV2(l.ctx, nil, row); err != nil {
		return nil, normalizeCharacterWriteError(err)
	}
	row, err = l.svcCtx.Model.AgentCharacter.FindOne(l.ctx, nil, row.Uuid)
	if err != nil {
		return nil, err
	}
	result := toCharacter(row)
	return &result, nil
}
