package auth

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jzero-io/jzero/core/status"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/token"

	"github.com/jzero-io/agentrazor/server/internal/errcodes"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

var (
	RefreshTokenExpiredErr = errors.New("refresh token expired")
	InvalidRefreshTokenErr = errors.New("invalid refresh token")
)

type RefreshToken struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewRefreshToken(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *RefreshToken {
	return &RefreshToken{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *RefreshToken) RefreshToken(req *types.RefreshTokenRequest) (resp *types.RefreshTokenResponse, err error) {
	// 解析 refreshToken
	parser := token.NewTokenParser()
	tok, err := parser.ParseToken(&http.Request{
		Header: http.Header{
			"Authorization": []string{req.RefreshToken},
		},
	}, l.svcCtx.MustGetConfig().Jwt.AccessSecret, "")
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, status.Wrap(errcodes.RefreshTokenExpiredCode, RefreshTokenExpiredErr)
		}
		return nil, status.Wrap(errcodes.RefreshTokenExpiredCode, InvalidRefreshTokenErr)
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Wrap(errcodes.RefreshTokenExpiredCode, InvalidRefreshTokenErr)
	}
	if tokenType, ok := claims[claimTokenType].(string); !ok || tokenType != tokenTypeRefresh {
		return nil, status.Wrap(errcodes.RefreshTokenExpiredCode, InvalidRefreshTokenErr)
	}
	userUUID, ok := claims["uuid"].(string)
	if !ok || userUUID == "" {
		return nil, status.Wrap(errcodes.RefreshTokenExpiredCode, InvalidRefreshTokenErr)
	}
	user, err := l.svcCtx.Model.ManageUser.FindOneByUuid(l.ctx, nil, userUUID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserEnabled(user.Status); err != nil {
		return nil, err
	}
	claims["username"] = user.Username
	roleUuids, err := enabledRoleUuidsByUser(l.ctx, l.svcCtx, user.Uuid)
	if err != nil {
		return nil, err
	}
	claims["role_uuids"] = roleUuids

	newAccessToken, newRefreshToken, err := issueTokenPair(l.ctx, l.svcCtx, user.Uuid, claims)
	if err != nil {
		return nil, err
	}

	return &types.RefreshTokenResponse{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
