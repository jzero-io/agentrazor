package group

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/zeromicro/go-zero/core/logx"

	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation/group"
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

func (l *Create) Create(req *types.CreateRequest) (resp *types.ConversationGroup, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("group name is required")
	}
	row := &conversationgroupmodel.ConversationGroup{Uuid: uuid.NewString(), UserUuid: user.Uuid, Name: name}
	if err := l.svcCtx.Model.ConversationGroup.InsertV2(l.ctx, nil, row); err != nil {
		return nil, err
	}
	row, err = l.svcCtx.Model.ConversationGroup.FindOne(l.ctx, nil, row.Uuid)
	if err != nil {
		return nil, err
	}
	return &types.ConversationGroup{Id: row.Uuid, Name: row.Name, CreatedAt: row.CreateTime.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), UpdatedAt: row.UpdateTime.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}, nil
}
