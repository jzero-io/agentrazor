package user

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/model/manage_role"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/model/manage_user_role"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/user"
)

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
	if err := ensureUsernameUnique(l.ctx, l.svcCtx, req.Username, ""); err != nil {
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
