package locale

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/server/internal/logic/v1/plugin/locale"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

func Get(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := locale.NewGet(r.Context(), svcCtx, r)
		resp, err := l.Get()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
