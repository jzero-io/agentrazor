package auth

import "github.com/pkg/errors"

const enabledUserStatus = "1"

var ErrUserDisabled = errors.New("用户已被禁用")

func ensureUserEnabled(status string) error {
	if status != enabledUserStatus {
		return ErrUserDisabled
	}
	return nil
}
