package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/google/uuid"
	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	agentapikeymodel "github.com/jzero-io/agentrazor/server/internal/model/agent_api_key"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth/apikey"
)

type Create struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 生成 Agent API 密钥，每个账户最多三个
func NewCreate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Create {
	return &Create{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Create) Create() (resp *types.CreateResponse, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	if user.Uuid == "" {
		return nil, errors.New("未登录")
	}

	plainKey := "ar-" + uuid.NewString()
	digest := sha256.Sum256([]byte(plainKey))
	row := &agentapikeymodel.AgentApiKey{
		Uuid:     uuid.NewString(),
		UserUuid: user.Uuid,
		KeyHash:  hex.EncodeToString(digest[:]),
		KeyHint:  keyHint(plainKey),
	}

	err = l.svcCtx.SqlxConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		if _, err := session.ExecCtx(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", user.Uuid); err != nil {
			return err
		}
		count, err := l.svcCtx.Model.AgentApiKey.CountByCondition(ctx, session, condition.NewChain().
			Equal(agentapikeymodel.UserUuid, user.Uuid).
			Build()...)
		if err != nil {
			return err
		}
		if count >= 3 {
			return errors.New("每个账户最多只能创建三个密钥")
		}
		return l.svcCtx.Model.AgentApiKey.InsertV2(ctx, session, row)
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateResponse{ApiKey: toAPIKey(row), Key: plainKey}, nil
}
