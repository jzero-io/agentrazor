package quota

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/pkg/errors"

	agenttokenquotamodel "github.com/jzero-io/agentrazor/server/internal/model/agent_token_quota"
	conversationtokenusageeventmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_token_usage_event"
)

const (
	SourceGlobal   = "global"
	SourceUser     = "user"
	SourceDisabled = "disabled"
)

var (
	ErrAgentDisabled         = errors.New("Agent 已被管理员禁用")
	ErrFiveHourQuotaExceeded = errors.New("5 小时使用限额已用完")
	ErrSevenDayQuotaExceeded = errors.New("每周使用限额已用完")
)

type GlobalConfig struct {
	FiveHourLimitTokens int64
	SevenDayLimitTokens int64
}

type Window struct {
	Limited          bool
	UsedTokens       int64
	LimitTokens      int64
	RemainingTokens  int64
	RemainingPercent float64
	Source           string
	ResetAt          *time.Time
}

type Status struct {
	UserUUID               string
	Configured             bool
	Enabled                bool
	FiveHourDisabled       bool
	FiveHourLimitTokens    *int64
	SevenDayLimitTokens    *int64
	EffectiveFiveHourLimit *int64
	EffectiveSevenDayLimit int64
	FiveHour               Window
	SevenDay               Window
	QuotaResetAt           *time.Time
}

type Service struct {
	quotas agenttokenquotamodel.AgentTokenQuotaModel
	usage  conversationtokenusageeventmodel.ConversationTokenUsageEventModel
	now    func() time.Time
}

func NewService(
	quotas agenttokenquotamodel.AgentTokenQuotaModel,
	usage conversationtokenusageeventmodel.ConversationTokenUsageEventModel,
) *Service {
	return &Service{
		quotas: quotas,
		usage:  usage,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Global(ctx context.Context) (*GlobalConfig, error) {
	row, err := s.quotas.FindOneByScope(ctx, nil, "global")
	if err != nil {
		return nil, err
	}
	return &GlobalConfig{
		FiveHourLimitTokens: row.FiveHourLimitTokens.Int64,
		SevenDayLimitTokens: row.SevenDayLimitTokens.Int64,
	}, nil
}

func (s *Service) SaveGlobal(ctx context.Context, fiveHourLimitTokens, sevenDayLimitTokens int64) error {
	return s.quotas.SaveGlobal(ctx, fiveHourLimitTokens, sevenDayLimitTokens)
}

func (s *Service) SaveUser(
	ctx context.Context,
	userUUID string,
	enabled, fiveHourDisabled bool,
	fiveHourLimitTokens, sevenDayLimitTokens *int64,
) error {
	if fiveHourDisabled {
		fiveHourLimitTokens = nil
	}
	return s.quotas.SaveUser(
		ctx,
		userUUID,
		enabled,
		fiveHourDisabled,
		nullInt64(fiveHourLimitTokens),
		nullInt64(sevenDayLimitTokens),
	)
}

func (s *Service) DeleteUser(ctx context.Context, userUUID string) error {
	return s.quotas.DeleteUser(ctx, userUUID)
}

func (s *Service) ResetUser(ctx context.Context, userUUID string) error {
	return s.quotas.ResetUser(ctx, userUUID, s.now())
}

func (s *Service) Status(ctx context.Context, userUUID string) (*Status, error) {
	global, err := s.Global(ctx)
	if err != nil {
		return nil, err
	}

	status := &Status{
		UserUUID:               userUUID,
		Enabled:                true,
		EffectiveFiveHourLimit: int64Pointer(global.FiveHourLimitTokens),
		EffectiveSevenDayLimit: global.SevenDayLimitTokens,
	}

	user, err := s.quotas.FindOneByUserUuid(
		ctx,
		nil,
		sql.NullString{String: userUUID, Valid: true},
	)
	if err != nil && !errors.Is(err, agenttokenquotamodel.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		status.Configured = true
		status.Enabled = user.Enabled
		status.FiveHourDisabled = user.FiveHourDisabled
		status.FiveHourLimitTokens = pointerFromNullInt64(user.FiveHourLimitTokens)
		status.SevenDayLimitTokens = pointerFromNullInt64(user.SevenDayLimitTokens)
		if user.QuotaResetAt.Valid {
			resetAt := user.QuotaResetAt.Time
			status.QuotaResetAt = &resetAt
		}
		if user.FiveHourDisabled {
			status.EffectiveFiveHourLimit = nil
		} else if user.FiveHourLimitTokens.Valid {
			status.EffectiveFiveHourLimit = int64Pointer(user.FiveHourLimitTokens.Int64)
		}
		if user.SevenDayLimitTokens.Valid {
			status.EffectiveSevenDayLimit = user.SevenDayLimitTokens.Int64
		}
	}

	now := s.now()
	fiveHourStart := later(now.Add(-5*time.Hour), status.QuotaResetAt)
	sevenDayStart := later(now.Add(-7*24*time.Hour), status.QuotaResetAt)
	usage, err := s.usage.TokenQuotaWindowUsage(ctx, userUUID, fiveHourStart, sevenDayStart)
	if err != nil {
		return nil, err
	}

	fiveHourSource := SourceGlobal
	sevenDaySource := SourceGlobal
	if status.Configured && status.FiveHourLimitTokens != nil {
		fiveHourSource = SourceUser
	}
	if status.Configured && status.SevenDayLimitTokens != nil {
		sevenDaySource = SourceUser
	}
	if status.FiveHourDisabled {
		fiveHourSource = SourceDisabled
	}

	if status.EffectiveFiveHourLimit != nil {
		status.FiveHour = limitedWindow(
			usage.FiveHourTokens,
			*status.EffectiveFiveHourLimit,
			fiveHourSource,
			timePointerFromNullTime(usage.FiveHourResetAt),
		)
	} else {
		status.FiveHour = Window{
			UsedTokens: usage.FiveHourTokens,
			Source:     fiveHourSource,
		}
	}
	status.SevenDay = limitedWindow(
		usage.SevenDayTokens,
		status.EffectiveSevenDayLimit,
		sevenDaySource,
		timePointerFromNullTime(usage.SevenDayResetAt),
	)
	return status, nil
}

func (s *Service) Check(ctx context.Context, userUUID string) error {
	status, err := s.Status(ctx, userUUID)
	if err != nil {
		return err
	}
	if !status.Enabled {
		return ErrAgentDisabled
	}
	if status.FiveHour.Limited && status.FiveHour.UsedTokens >= status.FiveHour.LimitTokens {
		return ErrFiveHourQuotaExceeded
	}
	if status.SevenDay.UsedTokens >= status.SevenDay.LimitTokens {
		return ErrSevenDayQuotaExceeded
	}
	return nil
}

func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func pointerFromNullInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return int64Pointer(value.Int64)
}

func timePointerFromNullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func int64Pointer(value int64) *int64 {
	return &value
}

func later(start time.Time, resetAt *time.Time) time.Time {
	if resetAt != nil && resetAt.After(start) {
		return *resetAt
	}
	return start
}

func limitedWindow(usedTokens, limitTokens int64, source string, resetAt *time.Time) Window {
	remaining := limitTokens - usedTokens
	if remaining < 0 {
		remaining = 0
	}
	percent := math.Round(float64(remaining)*1000/float64(limitTokens)) / 10
	return Window{
		Limited:          true,
		UsedTokens:       usedTokens,
		LimitTokens:      limitTokens,
		RemainingTokens:  remaining,
		RemainingPercent: percent,
		Source:           source,
		ResetAt:          resetAt,
	}
}
