package loginlock

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	MaxFailures = 5
	LockWindow  = 10 * time.Minute
	keyPrefix   = "PWD_LOGIN_FAILURE:"

	actionCheck   = "check"
	actionFailure = "failure"
	actionSuccess = "success"

	stateAllowed = 0
	stateLocked  = 1
)

var (
	ErrLocked = errors.New("密码错误次数过多，请 10 分钟后重试")

	guardScript = redis.NewScript(`
local action = ARGV[1]
local maxFailures = tonumber(ARGV[2])
local lockSeconds = tonumber(ARGV[3])
local failures = tonumber(redis.call("GET", KEYS[1]) or "0")

if action == "check" then
    if failures >= maxFailures then
        return 1
    end
    return 0
end

if action == "failure" then
    if failures >= maxFailures then
        return 1
    end

    failures = failures + 1
    if failures >= maxFailures then
        redis.call("SET", KEYS[1], maxFailures, "EX", lockSeconds)
        return 1
    end

    redis.call("SET", KEYS[1], failures)
    return 0
end

if action == "success" then
    if failures >= maxFailures then
        return 1
    end

    redis.call("DEL", KEYS[1])
    return 0
end

return -1
`)
)

type Guard struct {
	redis *redis.Redis
}

func NewGuard(redisStore *redis.Redis) *Guard {
	return &Guard{redis: redisStore}
}

func (g *Guard) IsLocked(ctx context.Context, userUUID string) (bool, error) {
	return g.run(ctx, userUUID, actionCheck)
}

func (g *Guard) RecordFailure(ctx context.Context, userUUID string) (bool, error) {
	return g.run(ctx, userUUID, actionFailure)
}

func (g *Guard) ResetAfterSuccess(ctx context.Context, userUUID string) (bool, error) {
	return g.run(ctx, userUUID, actionSuccess)
}

func (g *Guard) run(ctx context.Context, userUUID, action string) (bool, error) {
	if g == nil || g.redis == nil {
		return false, errors.New("password login limiter is unavailable")
	}
	if strings.TrimSpace(userUUID) == "" {
		return false, errors.New("password login limiter requires user UUID")
	}

	result, err := g.redis.ScriptRunCtx(ctx, guardScript, []string{key(userUUID)}, []string{
		action,
		strconv.Itoa(MaxFailures),
		strconv.Itoa(int(LockWindow / time.Second)),
	})
	if err != nil {
		return false, err
	}

	state, ok := result.(int64)
	if !ok {
		return false, errors.New("unexpected password login limiter response")
	}
	switch state {
	case stateAllowed:
		return false, nil
	case stateLocked:
		return true, nil
	default:
		return false, errors.New("unknown password login limiter state")
	}
}

func key(userUUID string) string {
	return keyPrefix + strings.TrimSpace(userUUID)
}
