package character

import (
	"context"
	"net/http"
	"strings"

	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	agentcharactermodel "github.com/jzero-io/agentrazor/server/internal/model/agent_character"
	manageusermodel "github.com/jzero-io/agentrazor/server/internal/model/manage_user"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/character"
)

type List struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewList(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *List {
	return &List{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *List) List(req *types.ListRequest) (*types.ListResponse, error) {
	isBuiltin := req.Kind == "builtin"
	keyword := strings.TrimSpace(req.Keyword)
	rows, total, err := l.svcCtx.Model.AgentCharacter.PageByCondition(l.ctx, nil, condition.NewChain().
		Equal(agentcharactermodel.IsBuiltin, isBuiltin).
		Like(agentcharactermodel.Name, "%"+keyword+"%", condition.WithSkip(keyword == "")).
		Page(req.Current, req.Size).
		OrderByAsc(agentcharactermodel.Sort).
		OrderByDesc(agentcharactermodel.CreateTime).
		Build()...)
	if err != nil {
		return nil, err
	}
	records := make([]types.Character, 0, len(rows))
	for _, row := range rows {
		var owner *types.Owner
		if row.UserUuid.Valid {
			user, err := l.svcCtx.Model.ManageUser.FindOneByCondition(l.ctx, nil, condition.NewChain().
				Equal(manageusermodel.Uuid, row.UserUuid.String).
				Build()...)
			if err != nil {
				return nil, err
			}
			owner = &types.Owner{UserUuid: user.Uuid, Username: user.Username, NickName: user.Nickname, UserEmail: user.Email}
		}
		records = append(records, toCharacter(row, owner))
	}
	return &types.ListResponse{
		PageResponse: types.PageResponse{Current: req.Current, Size: req.Size, Total: total},
		Records:      records,
	}, nil
}
