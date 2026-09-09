package conversation

import (
	"time"

	"github.com/jzero-io/agentrazor/server/internal/quota"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

func toTokenQuotaWindow(value quota.Window) types.TokenQuotaWindow {
	result := types.TokenQuotaWindow{
		Limited:          value.Limited,
		UsedTokens:       value.UsedTokens,
		LimitTokens:      value.LimitTokens,
		RemainingTokens:  value.RemainingTokens,
		RemainingPercent: value.RemainingPercent,
		Source:           value.Source,
	}
	if value.ResetAt != nil {
		formatted := value.ResetAt.UTC().Format(time.RFC3339)
		result.ResetAt = &formatted
	}
	return result
}
