package conversation

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type WorkspaceFiles struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取会话工作区文件
func NewWorkspaceFiles(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *WorkspaceFiles {
	return &WorkspaceFiles{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *WorkspaceFiles) WorkspaceFiles(req *types.PathRequest) (resp *types.WorkspaceFilesResponse, err error) {
	if _, err = requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return nil, err
	}

	root := filepath.Join(l.svcCtx.MustGetConfig().Agent.AgentrazorHome, req.ConversationId)
	entries := make([]types.WorkspaceEntry, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		relativePath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		itemType := "file"
		var size int64
		if entry.IsDir() {
			itemType = "directory"
		} else {
			info, infoErr := entry.Info()
			if infoErr != nil {
				return infoErr
			}
			size = info.Size()
		}
		entries = append(entries, types.WorkspaceEntry{
			Name: entry.Name(),
			Path: filepath.ToSlash(relativePath),
			Type: itemType,
			Size: size,
		})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return &types.WorkspaceFilesResponse{Entries: entries}, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })

	return &types.WorkspaceFilesResponse{Entries: entries}, nil
}
