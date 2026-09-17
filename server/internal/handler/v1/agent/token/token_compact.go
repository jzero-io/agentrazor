package token

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/server/internal/logic/v1/agent/token"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/token"
)

// 获取当前用户 Token 额度状态
func QuotaStatus(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := token.NewQuotaStatus(r.Context(), svcCtx, r)
		resp, err := l.QuotaStatus()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 分页获取账号下的对话和 Turn Token 消耗明细
func UsageConversations(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TokenUsageConversationsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := token.NewUsageConversations(r.Context(), svcCtx, r)
		resp, err := l.UsageConversations(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 分页获取账号 Token 消耗明细
func UsageDetails(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TokenUsageDetailsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := token.NewUsageDetails(r.Context(), svcCtx, r)
		resp, err := l.UsageDetails(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 获取 Token 消耗趋势
func UsageTrend(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TokenUsageTrendRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := token.NewUsageTrend(r.Context(), svcCtx, r)
		resp, err := l.UsageTrend(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
