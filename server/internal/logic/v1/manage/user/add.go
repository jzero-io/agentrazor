package user

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_user_role"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/user"
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

func ensureUsernameUnique(ctx context.Context, model manage_user.ManageUserModel, username, excludeUuid string) error {
	conditions := condition.NewChain().Equal(manage_user.Username, username).Build()
	if excludeUuid != "" {
		conditions = append(conditions, condition.NewChain().NotEqual(manage_user.Uuid, excludeUuid).Build()...)
	}

	if _, err := model.FindOneByCondition(ctx, nil, conditions...); err == nil {
		return errUsernameExists
	} else if !errors.Is(err, manage_user.ErrNotFound) {
		return err
	}
	return nil
}

type Add struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewAdd(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Add {
	return &Add{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *Add) Add(req *types.AddRequest) (resp *types.AddResponse, err error) {
	if err := validateUsername(req.Username); err != nil {
		return nil, err
	}
	if len(req.UserRoles) == 0 {
		return nil, errors.New("用户角色不能为空")
	}
	if err := ensureUsernameUnique(l.ctx, l.svcCtx.Model.ManageUser, req.Username, ""); err != nil {
		return nil, err
	}

	userUuid := uuid.New().String()
	if err = l.svcCtx.Model.ManageUser.InsertV2(l.ctx, nil, &manage_user.ManageUser{
		Uuid:     userUuid,
		Username: req.Username,
		Password: req.Password,
		Nickname: req.NickName,
		Phone:    req.UserPhone,
		Status:   req.Status,
		Email:    req.UserEmail,
	}); err != nil {
		return nil, err
	}

	var bulk []*manage_user_role.ManageUserRole
	roles, err := l.svcCtx.Model.ManageRole.FindByCondition(l.ctx, nil, condition.Condition{
		Field:    manage_role.Code,
		Operator: condition.In,
		Value:    req.UserRoles,
	})
	if err != nil {
		return nil, err
	}

	for _, v := range roles {
		bulk = append(bulk, &manage_user_role.ManageUserRole{
			Uuid:     uuid.New().String(),
			UserUuid: userUuid,
			RoleUuid: v.Uuid,
		})
	}

	err = l.svcCtx.Model.ManageUserRole.BulkInsert(l.ctx, nil, bulk)

	return
}
