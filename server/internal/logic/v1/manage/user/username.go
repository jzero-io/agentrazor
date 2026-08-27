package user

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jzero-io/jzero/core/stores/condition"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

const (
	minUsernameLength = 4
	maxUsernameLength = 20
)

var (
	usernamePattern    = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_-]+$`)
	errUsernameInvalid = errors.New("用户名格式不正确")
	errUsernameExists  = errors.New("用户名已存在")
)

func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	length := utf8.RuneCountInString(username)
	if length < minUsernameLength || length > maxUsernameLength || !usernamePattern.MatchString(username) {
		return errUsernameInvalid
	}
	return nil
}

func ensureUsernameUnique(ctx context.Context, svcCtx *svc.ServiceContext, username, excludeUuid string) error {
	conditions := condition.NewChain().Equal(manage_user.Username, username).Build()
	if excludeUuid != "" {
		conditions = append(conditions, condition.NewChain().NotEqual(manage_user.Uuid, excludeUuid).Build()...)
	}

	if _, err := svcCtx.Model.ManageUser.FindOneByCondition(ctx, nil, conditions...); err == nil {
		return errUsernameExists
	} else if !errors.Is(err, manage_user.ErrNotFound) {
		return err
	}
	return nil
}
