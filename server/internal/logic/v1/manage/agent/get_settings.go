package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

func toAccountStatus(status agentdomain.AccountStatus) types.AccountStatus {
	return types.AccountStatus{
		AuthMode: status.AuthMode,
		Email:    status.Email,
		PlanType: status.PlanType,
		LoggedIn: status.LoggedIn,
	}
}

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
	settings, err := l.svcCtx.AgentService.Settings(l.ctx)
	if err != nil {
		return nil, err
	}
	account := agentdomain.AccountStatus{}
	if value, accountErr := l.svcCtx.AgentService.AccountStatus(l.ctx); accountErr == nil {
		account = value
	} else {
		l.Errorf("read Codex account: %v", accountErr)
	}
	providers, err := settings.Providers()
	if err != nil {
		return nil, err
	}
	return &types.GetSettingsResponse{
		ActiveProvider: settings.ActiveProvider(), Model: settings.Model(), ReasoningEffort: settings.ReasoningEffort(),
		Providers: providers, Account: toAccountStatus(account), Runtime: types.RuntimeStatus{Running: l.svcCtx.AgentService.Running()},
	}, nil
}
