package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/jzero-io/agentrazor/server/internal/model"
	agentapikeymodel "github.com/jzero-io/agentrazor/server/internal/model/agent_api_key"
)

var ErrInvalidAPIKey = errors.New("invalid api key")

const APIKeyHeader = "X-API-Key"

func APIKeyFromRequest(r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get(APIKeyHeader))
	if !strings.HasPrefix(key, "ar-") {
		return "", false
	}
	return key, true
}

func ContextForAPIKey(ctx context.Context, key string, models model.Model) (context.Context, error) {
	digest := sha256.Sum256([]byte(key))
	row, err := models.AgentApiKey.FindOneByKeyHash(ctx, nil, hex.EncodeToString(digest[:]))
	if err != nil {
		if errors.Is(err, agentapikeymodel.ErrNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}
	user, err := models.ManageUser.FindOneByUuid(ctx, nil, row.UserUuid)
	if err != nil || user.Status != "1" {
		return nil, ErrInvalidAPIKey
	}
	ctx = context.WithValue(ctx, "uuid", user.Uuid)
	ctx = context.WithValue(ctx, "username", user.Username)
	ctx = context.WithValue(ctx, "role_uuids", []any{})
	return ctx, nil
}
