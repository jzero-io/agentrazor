package agent

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	manageusermodel "github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/service/quota"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

func ensureTokenQuotaUser(ctx context.Context, users manageusermodel.ManageUserModel, userUUID string) error {
	_, err := users.FindOneByUuid(ctx, nil, userUUID)
	if errors.Is(err, manageusermodel.ErrNotFound) {
		return errors.New("用户不存在")
	}
	return err
}

type GetUserTokenQuota struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetUserTokenQuota(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetUserTokenQuota {
	return &GetUserTokenQuota{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetUserTokenQuota) GetUserTokenQuota(req *types.GetUserTokenQuotaRequest) (resp *types.GetUserTokenQuotaResponse, err error) {
	if err := ensureTokenQuotaUser(l.ctx, l.svcCtx.Model.ManageUser, req.UserUuid); err != nil {
		return nil, err
	}
	value, err := l.svcCtx.TokenQuota.Status(l.ctx, req.UserUuid)
	if err != nil {
		return nil, err
	}
	return &types.GetUserTokenQuotaResponse{Quota: toTokenQuotaUser(value)}, nil
}

func toTokenQuotaUser(value *quota.Status) types.TokenQuotaUser {
	result := types.TokenQuotaUser{
		UserUuid:               value.UserUUID,
		Configured:             value.Configured,
		Enabled:                value.Enabled,
		FiveHourDisabled:       value.FiveHourDisabled,
		FiveHourLimitTokens:    value.FiveHourLimitTokens,
		SevenDayLimitTokens:    value.SevenDayLimitTokens,
		EffectiveFiveHourLimit: value.EffectiveFiveHourLimit,
		EffectiveSevenDayLimit: value.EffectiveSevenDayLimit,
		FiveHourUsedTokens:     value.FiveHour.UsedTokens,
		SevenDayUsedTokens:     value.SevenDay.UsedTokens,
	}
	if value.QuotaResetAt != nil {
		formatted := value.QuotaResetAt.UTC().Format(time.RFC3339)
		result.QuotaResetAt = &formatted
	}
	if value.FiveHour.ResetAt != nil {
		formatted := value.FiveHour.ResetAt.UTC().Format(time.RFC3339)
		result.FiveHourResetAt = &formatted
	}
	if value.SevenDay.ResetAt != nil {
		formatted := value.SevenDay.ResetAt.UTC().Format(time.RFC3339)
		result.SevenDayResetAt = &formatted
	}
	return result
}
