package group

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation/group"
)

const conversationGroupUserNameConstraint = "uk_conversation_group_user_name"

var errGroupNameExists = errors.New("分组名称已存在")

func ensureGroupNameUnique(ctx context.Context, model conversationgroupmodel.ConversationGroupModel, userUUID, name, excludeUUID string) error {
	chain := condition.NewChain().
		Equal(conversationgroupmodel.UserUuid, userUUID).
		Equal(conversationgroupmodel.Name, name)
	if excludeUUID != "" {
		chain = chain.NotEqual(conversationgroupmodel.Uuid, excludeUUID)
	}

	_, err := model.FindOneByCondition(ctx, nil, chain.Build()...)
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
	if err := ensureGroupNameUnique(l.ctx, l.svcCtx.Model.ConversationGroup, user.Uuid, name, ""); err != nil {
		return nil, err
	}
	row := &conversationgroupmodel.ConversationGroup{Uuid: uuid.NewString(), UserUuid: user.Uuid, Name: name}
	if err := l.svcCtx.Model.ConversationGroup.InsertV2(l.ctx, nil, row); err != nil {
		return nil, normalizeGroupWriteError(err)
	}
	row, err = l.svcCtx.Model.ConversationGroup.FindOne(l.ctx, nil, row.Uuid)
	if err != nil {
		return nil, err
	}
	return &types.ConversationGroup{Id: row.Uuid, Name: row.Name, CreatedAt: row.CreateTime.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), UpdatedAt: row.UpdateTime.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}, nil
}
