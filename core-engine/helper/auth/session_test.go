package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func TestSessionStoreLifecycle(t *testing.T) {
	redisServer := miniredis.RunT(t)
	store := NewSessionStore(redis.New(redisServer.Addr()))
	ctx := context.Background()

	const (
		userUUID = "user-1"
		token    = "header.payload.signature"
		ttl      = 30
	)
	if err := store.Save(ctx, userUUID, token, ttl); err != nil {
		t.Fatalf("save session: %v", err)
	}

	stored, err := redisServer.Get(SessionKey(userUUID, token))
	if err != nil {
		t.Fatalf("read stored session: %v", err)
	}
	if stored != token {
		t.Fatalf("stored token = %q, want %q", stored, token)
	}
	if got := redisServer.TTL(SessionKey(userUUID, token)); got != ttl*time.Second {
		t.Fatalf("ttl = %s, want %s", got, ttl*time.Second)
	}

	exists, err := store.Exists(ctx, userUUID, token)
	if err != nil {
		t.Fatalf("check session: %v", err)
	}
	if !exists {
		t.Fatal("saved session should exist")
	}

	exists, err = store.Exists(ctx, userUUID, "another-token")
	if err != nil {
		t.Fatalf("check unknown session: %v", err)
	}
	if exists {
		t.Fatal("unknown session should not exist")
	}

	if err := store.Delete(ctx, userUUID, token); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	exists, err = store.Exists(ctx, userUUID, token)
	if err != nil {
		t.Fatalf("check deleted session: %v", err)
	}
	if exists {
		t.Fatal("deleted session should not exist")
	}
}

func TestSessionStoreRejectsInvalidInputAndUnavailableRedis(t *testing.T) {
	ctx := context.Background()
	store := NewSessionStore(nil)

	if err := store.Save(ctx, "user-1", "token", 30); err == nil {
		t.Fatal("save should fail without Redis")
	}
	if _, err := store.Exists(ctx, "user-1", "token"); err == nil {
		t.Fatal("exists should fail without Redis")
	}
	if err := store.Delete(ctx, "user-1", "token"); err == nil {
		t.Fatal("delete should fail without Redis")
	}

	redisServer := miniredis.RunT(t)
	store = NewSessionStore(redis.New(redisServer.Addr()))
	if err := store.Save(ctx, "", "token", 30); err == nil {
		t.Fatal("save should reject empty user UUID")
	}
	if err := store.Save(ctx, "user-1", "", 30); err == nil {
		t.Fatal("save should reject empty token")
	}
	if err := store.Save(ctx, "user-1", "token", 0); err == nil {
		t.Fatal("save should reject non-positive TTL")
	}
}
