package conversation

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/server/internal/logic/v1/conversation"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

// 获取会话列表
func List(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := conversation.NewList(r.Context(), svcCtx, r)
		resp, err := l.List()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 创建会话
func Create(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewCreate(r.Context(), svcCtx, r)
		resp, err := l.Create(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 获取会话详情
func Get(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PathRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewGet(r.Context(), svcCtx, r)
		resp, err := l.Get(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 修改会话
func Update(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewUpdate(r.Context(), svcCtx, r)
		resp, err := l.Update(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 删除会话
func Delete(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PathRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewDelete(r.Context(), svcCtx, r)
		resp, err := l.Delete(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 发送消息
func SendMessage(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendMessageRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewSendMessage(r.Context(), svcCtx, r)
		resp, err := l.SendMessage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 停止当前 Turn
func CancelTurn(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PathRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewCancelTurn(r.Context(), svcCtx, r)
		resp, err := l.CancelTurn(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

// 订阅会话与 Agent 流式事件
func StreamEvents(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EventsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// Buffer size of 16 is chosen as a reasonable default to balance throughput and memory usage.
		// You can change this based on your application's needs.
		// if your go-zero version less than 1.8.1, you need to add 3 lines below.
		// w.Header().Set("Content-Type", "text/event-stream")
		// w.Header().Set("Cache-Control", "no-cache")
		// w.Header().Set("Connection", "keep-alive")
		client := make(chan *types.EventsResponse, 16)

		l := conversation.NewStreamEvents(r.Context(), svcCtx)
		threading.GoSafeCtx(r.Context(), func() {
			defer close(client)
			err := l.StreamEvents(&req, client)
			if err != nil {
				logc.Errorw(r.Context(), "StreamEventsHandler", logc.Field("error", err))
				return
			}
		})

		for {
			select {
			case data, ok := <-client:
				if !ok {
					return
				}
				output, err := json.Marshal(data)
				if err != nil {
					logc.Errorw(r.Context(), "StreamEventsHandler", logc.Field("error", err))
					continue
				}

				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(output)); err != nil {
					logc.Errorw(r.Context(), "StreamEventsHandler", logc.Field("error", err))
					return
				}
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	}
}
