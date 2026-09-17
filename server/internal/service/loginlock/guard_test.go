package loginlock

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	miniredisserver "github.com/alicebob/miniredis/v2/server"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func TestGuardLocksOnFifthFailureAndDoesNotExtendLock(t *testing.T) {
	redisServer := miniredis.RunT(t)
	guard := NewGuard(redis.New(redisServer.Addr()))
	ctx := context.Background()
	const userUUID = "user-1"

	for attempt := 1; attempt < MaxFailures; attempt++ {
		locked, err := guard.RecordFailure(ctx, userUUID)
		if err != nil {
			t.Fatalf("record failure %d: %v", attempt, err)
		}
		if locked {
			t.Fatalf("attempt %d locked before threshold", attempt)
		}
	}

	locked, err := guard.RecordFailure(ctx, userUUID)
	if err != nil {
		t.Fatalf("record threshold failure: %v", err)
	}
	if !locked {
		t.Fatal("fifth failure should lock login")
	}
	if got := redisServer.TTL(key(userUUID)); got != LockWindow {
		t.Fatalf("lock ttl = %s, want %s", got, LockWindow)
	}

	redisServer.FastForward(time.Minute)
	locked, err = guard.RecordFailure(ctx, userUUID)
	if err != nil {
		t.Fatalf("record failure while locked: %v", err)
	}
	if !locked {
		t.Fatal("login should remain locked")
	}
	if got := redisServer.TTL(key(userUUID)); got != LockWindow-time.Minute {
		t.Fatalf("lock ttl was extended: got %s, want %s", got, LockWindow-time.Minute)
	}

	locked, err = guard.ResetAfterSuccess(ctx, userUUID)
	if err != nil {
		t.Fatalf("reset while locked: %v", err)
	}
	if !locked {
		t.Fatal("success must not clear an active lock")
	}

	redisServer.FastForward(LockWindow - time.Minute)
	locked, err = guard.IsLocked(ctx, userUUID)
	if err != nil {
		t.Fatalf("check expired lock: %v", err)
	}
	if locked {
		t.Fatal("lock should expire after ten minutes")
	}
}

func TestGuardSuccessResetsConsecutiveFailures(t *testing.T) {
	redisServer := miniredis.RunT(t)
	guard := NewGuard(redis.New(redisServer.Addr()))
	ctx := context.Background()
	const userUUID = "user-1"

	for attempt := 0; attempt < MaxFailures-1; attempt++ {
		locked, err := guard.RecordFailure(ctx, userUUID)
		if err != nil || locked {
			t.Fatalf("record failure %d: locked=%v err=%v", attempt+1, locked, err)
		}
	}

	locked, err := guard.ResetAfterSuccess(ctx, userUUID)
	if err != nil {
		t.Fatalf("reset failures: %v", err)
	}
	if locked {
		t.Fatal("successful login before threshold should not be locked")
	}
	if redisServer.Exists(key(userUUID)) {
		t.Fatal("successful login should delete the failure counter")
	}

	locked, err = guard.RecordFailure(ctx, userUUID)
	if err != nil || locked {
		t.Fatalf("first failure after reset: locked=%v err=%v", locked, err)
	}
	value, err := redisServer.Get(key(userUUID))
	if err != nil {
		t.Fatalf("read failure count: %v", err)
	}
	if value != "1" {
		t.Fatalf("failure count = %s, want 1", value)
	}
}

func TestGuardUsersAreIsolated(t *testing.T) {
	redisServer := miniredis.RunT(t)
	guard := NewGuard(redis.New(redisServer.Addr()))
	ctx := context.Background()

	for attempt := 0; attempt < MaxFailures; attempt++ {
		if _, err := guard.RecordFailure(ctx, "user-1"); err != nil {
			t.Fatalf("record failure: %v", err)
		}
	}
	locked, err := guard.IsLocked(ctx, "user-2")
	if err != nil {
		t.Fatalf("check second user: %v", err)
	}
	if locked {
		t.Fatal("one user's failures must not lock another user")
	}
}

func TestGuardConcurrentFailuresAreCapped(t *testing.T) {
	redisServer := miniredis.RunT(t)
	// go-redis identifies each new pooled connection with CLIENT SETINFO.
	// miniredis does not implement that subcommand, so accept it here to keep
	// the race test focused on the Lua state transition rather than test-server
	// handshake errors in go-zero's circuit breaker.
	redisServer.Server().SetPreHook(func(peer *miniredisserver.Peer, command string, args ...string) bool {
		if strings.EqualFold(command, "CLIENT") && len(args) > 0 && strings.EqualFold(args[0], "SETINFO") {
			peer.WriteOK()
			return true
		}
		return false
	})
	guard := NewGuard(redis.New(redisServer.Addr()))
	ctx := context.Background()
	const userUUID = "user-1"

	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := guard.RecordFailure(ctx, userUUID)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent failure: %v", err)
		}
	}

	value, err := redisServer.Get(key(userUUID))
	if err != nil {
		t.Fatalf("read failure count: %v", err)
	}
	if value != "5" {
		t.Fatalf("failure count = %s, want 5", value)
	}
}

func TestGuardFailsClosedWithoutRedis(t *testing.T) {
	guard := NewGuard(nil)
	if _, err := guard.IsLocked(context.Background(), "user-1"); err == nil {
		t.Fatal("lock check should fail without Redis")
	}
}
