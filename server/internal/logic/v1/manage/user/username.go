package user

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jzero-io/jzero/core/stores/condition"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

const maxUsernameLength = 20

var (
	errUsernameTooLong = errors.New("用户名不能超过20个字符")
	errUsernameExists  = errors.New("用户名已存在")
)

func validateUsername(username string) error {
	if utf8.RuneCountInString(strings.TrimSpace(username)) > maxUsernameLength {
		return errUsernameTooLong
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
