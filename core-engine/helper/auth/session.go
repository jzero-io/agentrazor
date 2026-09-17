package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const SessionKeyPrefix = "JWT_AUTH"

var errInvalidSession = errors.New("invalid JWT session")

type SessionStore struct {
	redis *redis.Redis
}

func NewSessionStore(redisStore *redis.Redis) *SessionStore {
	return &SessionStore{redis: redisStore}
}

func SessionKey(userUUID, token string) string {
	return fmt.Sprintf("%s:%s:%s", SessionKeyPrefix, userUUID, token)
}

func (s *SessionStore) Save(ctx context.Context, userUUID, token string, ttlSeconds int) error {
	if err := validateSession(userUUID, token, ttlSeconds); err != nil {
		return err
	}
	if s == nil || s.redis == nil {
		return errors.New("JWT session store is unavailable")
	}

	return s.redis.SetexCtx(ctx, SessionKey(userUUID, token), token, ttlSeconds)
}

func (s *SessionStore) Exists(ctx context.Context, userUUID, token string) (bool, error) {
	if strings.TrimSpace(userUUID) == "" || strings.TrimSpace(token) == "" {
		return false, errInvalidSession
	}
	if s == nil || s.redis == nil {
		return false, errors.New("JWT session store is unavailable")
	}

	storedToken, err := s.redis.GetCtx(ctx, SessionKey(userUUID, token))
	if err != nil {
		return false, err
	}
	if storedToken == "" {
		return false, nil
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) == 1, nil
}

func (s *SessionStore) Delete(ctx context.Context, userUUID, token string) error {
	if strings.TrimSpace(userUUID) == "" || strings.TrimSpace(token) == "" {
		return errInvalidSession
	}
	if s == nil || s.redis == nil {
		return errors.New("JWT session store is unavailable")
	}

	_, err := s.redis.DelCtx(ctx, SessionKey(userUUID, token))
	return err
}

func validateSession(userUUID, token string, ttlSeconds int) error {
	if strings.TrimSpace(userUUID) == "" || strings.TrimSpace(token) == "" || ttlSeconds <= 0 {
		return errInvalidSession
	}
	return nil
}
