package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type GetSettings struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetSettings(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetSettings {
	return &GetSettings{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetSettings) GetSettings(req *types.GetSettingsRequest) (resp *types.GetSettingsResponse, err error) {
	store, err := readAgentSettingsStore(l.ctx, l.svcCtx.Codex)
	if err != nil {
		return nil, err
	}
	account := agentdomain.AccountStatus{}
	if l.svcCtx.AgentThreads != nil {
		if value, accountErr := l.svcCtx.AgentThreads.AccountStatus(l.ctx); accountErr == nil {
			account = value
		} else {
			l.Errorf("read Codex account: %v", accountErr)
		}
	}
	providers, err := store.providers()
	if err != nil {
		return nil, err
	}
	runtime := types.RuntimeStatus{}
	if l.svcCtx.AgentThreads != nil {
		runtime = runtimeStatus(l.svcCtx.AgentThreads.RuntimeStatus())
	}
	return &types.GetSettingsResponse{
		ActiveProvider: store.activeProvider(), Model: store.model(), ReasoningEffort: store.reasoningEffort(),
		Providers: providers, Account: toAccountStatus(account), Runtime: runtime,
	}, nil
}
