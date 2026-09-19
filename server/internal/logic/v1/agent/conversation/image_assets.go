package conversation

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/agent/conversation"
)

type ImageAssets struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取会话生成的图片资产
func NewImageAssets(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ImageAssets {
	return &ImageAssets{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *ImageAssets) ImageAssets(req *types.PathRequest) (resp *types.ImageAssetsResponse, err error) {
	if _, err = requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return nil, err
	}
	assets, err := l.svcCtx.AgentService.ListGeneratedImageAssets(l.ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return &types.ImageAssetsResponse{Images: imageAssets(assets)}, nil
}

func imageAssets(values []agentdomain.GeneratedImageAsset) []types.ImageAsset {
	result := make([]types.ImageAsset, 0, len(values))
	for _, value := range values {
		result = append(result, types.ImageAsset{Name: value.Name, Size: value.Size})
	}
	return result
}
