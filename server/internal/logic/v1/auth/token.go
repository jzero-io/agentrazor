package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/jzero-io/agentrazor/server/internal/service/loginlock"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

const (
	claimTokenType   = "token_type"
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

func CreateToken(secret string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func issueTokenPair(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string, claims map[string]any) (string, string, error) {
	config := svcCtx.MustGetConfig()
	now := time.Now()

	accessClaims := cloneClaims(claims)
	accessClaims[claimTokenType] = tokenTypeAccess
	accessClaims["jti"] = uuid.NewString()
	accessClaims["iat"] = now.Unix()
	accessClaims["exp"] = now.Add(time.Duration(config.Jwt.AccessExpire) * time.Second).Unix()
	accessToken, err := CreateToken(config.Jwt.AccessSecret, accessClaims)
	if err != nil {
		return "", "", err
	}

	refreshClaims := cloneClaims(claims)
	refreshClaims[claimTokenType] = tokenTypeRefresh
	refreshClaims["jti"] = uuid.NewString()
	refreshClaims["iat"] = now.Unix()
	refreshClaims["exp"] = now.Add(time.Duration(config.Jwt.RefreshExpire) * time.Second).Unix()
	refreshToken, err := CreateToken(config.Jwt.AccessSecret, refreshClaims)
	if err != nil {
		return "", "", err
	}

	if err := svcCtx.AuthxMiddleware.Save(ctx, userUUID, accessToken, config.Jwt.AccessExpire); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func issueLoginTokenPair(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string, claims map[string]any) (string, string, error) {
	accessToken, refreshToken, err := issueTokenPair(ctx, svcCtx, userUUID, claims)
	if err != nil {
		return "", "", err
	}

	loginGuard := loginlock.NewGuard(svcCtx.Redis)
	locked, resetErr := loginGuard.ResetAfterSuccess(ctx, userUUID)
	if resetErr != nil || locked {
		// Never expose an access token when the final atomic lock check rejects
		// the login. Deletion is best effort because the token has not left the
		// server and therefore cannot be used by the caller.
		_ = svcCtx.AuthxMiddleware.Delete(ctx, userUUID, accessToken)
		if resetErr != nil {
			return "", "", resetErr
		}
		return "", "", loginlock.ErrLocked
	}

	return accessToken, refreshToken, nil
}

func cloneClaims(claims map[string]any) jwt.MapClaims {
	cloned := make(jwt.MapClaims, len(claims))
	for key, value := range claims {
		cloned[key] = value
	}
	return cloned
}
