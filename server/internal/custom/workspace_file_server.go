package custom

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jzero-io/agentrazor/server/internal/middleware"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/token"
)

const workspaceURLPrefix = "/dist/data/agentrazor-home/"

// RegisterWorkspaceFileServer exposes conversation workspace files through a
// controlled endpoint. nginx keeps Codex-style /dist/data/... links outside,
// then rewrites them to this endpoint so ownership can be checked here.
func RegisterWorkspaceFileServer(server *rest.Server, svcCtx *svc.ServiceContext) {
	parser := token.NewTokenParser()
	handler := func(w http.ResponseWriter, r *http.Request) {
		serveWorkspaceFile(w, r, svcCtx, parser)
	}
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/workspace-files", Handler: handler},
		{Method: http.MethodHead, Path: "/api/v1/workspace-files", Handler: handler},
	})
}

func serveWorkspaceFile(w http.ResponseWriter, r *http.Request, svcCtx *svc.ServiceContext, parser *token.TokenParser) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	userUUID, ok := authenticatedUserUUID(r, svcCtx, parser)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	conversationID, filePath, ok := splitWorkspacePath(r.URL.Query().Get("path"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	row, err := svcCtx.Model.Conversation.FindOne(r.Context(), nil, conversationID)
	if errors.Is(err, sqlx.ErrNotFound) || (err == nil && row.UserUuid != userUUID) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	runtimeFile, err := svcCtx.Codex.ReadWorkspaceFile(r.Context(), conversationID, filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contentType := strings.TrimSpace(runtimeFile.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(runtimeFile.Data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodGet {
		_, _ = w.Write(runtimeFile.Data)
	}
}

func authenticatedUserUUID(r *http.Request, svcCtx *svc.ServiceContext, parser *token.TokenParser) (string, bool) {
	if key, ok := middleware.APIKeyFromRequest(r); ok {
		ctx, err := middleware.ContextForAPIKey(r.Context(), key, svcCtx.Model)
		if err != nil {
			return "", false
		}
		userUUID, _ := ctx.Value("uuid").(string)
		userUUID = strings.TrimSpace(userUUID)
		return userUUID, userUUID != ""
	}

	tok, err := parser.ParseToken(r, svcCtx.MustGetConfig().Jwt.AccessSecret, "")
	if err != nil || !tok.Valid {
		return "", false
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	userUUID, _ := claims["uuid"].(string)
	return strings.TrimSpace(userUUID), strings.TrimSpace(userUUID) != ""
}

func splitWorkspacePath(urlPath string) (conversationID string, filePath string, ok bool) {
	cleanURLPath := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	if !strings.HasPrefix(cleanURLPath, workspaceURLPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(cleanURLPath, workspaceURLPrefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	if strings.ContainsAny(parts[0], `/\`) {
		return "", "", false
	}
	return parts[0], parts[1], true
}
