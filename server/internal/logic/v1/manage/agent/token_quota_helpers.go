package agent

import (
	"context"
	"time"

	"github.com/pkg/errors"

	manageusermodel "github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/quota"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

func ensureTokenQuotaUser(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string) error {
	_, err := svcCtx.Model.ManageUser.FindOneByUuid(ctx, nil, userUUID)
	if errors.Is(err, manageusermodel.ErrNotFound) {
		return errors.New("用户不存在")
	}
	return err
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
