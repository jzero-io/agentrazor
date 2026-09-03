package auth

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/constant"
	"github.com/jzero-io/agentrazor/server/internal/mailer"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/auth"
)

var SendVerificationError = errors.New("发送失败, 请联系管理员")

func verificationCacheKey(verificationUUID string) string {
	return fmt.Sprintf("%s:%s", constant.CacheVerificationCodePrefix, verificationUUID)
}

func encodedEmailVerification(email, code string) string {
	return strings.TrimSpace(email) + "\n" + code
}

func verifyEmailCode(svcCtx *svc.ServiceContext, verificationUUID, email, code string) (bool, error) {
	var cached string
	if err := svcCtx.Cache.Get(verificationCacheKey(verificationUUID), &cached); err != nil {
		return false, err
	}
	parts := strings.SplitN(cached, "\n", 2)
	return len(parts) == 2 && strings.EqualFold(parts[0], strings.TrimSpace(email)) && parts[1] == code, nil
}

type SendVerificationCode struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSendVerificationCode(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SendVerificationCode {
	return &SendVerificationCode{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx, r: r,
	}
}

func (l *SendVerificationCode) SendVerificationCode(req *types.SendVerificationCodeRequest) (resp *types.SendVerificationCodeResponse, err error) {
	if req.VerificationType == "email" {
		email, err := l.svcCtx.Model.ManageEmail.FindOneByCondition(l.ctx, nil)
		if err != nil {
			return nil, SendVerificationError
		}

		verificationUuid := uuid.New().String()
		verificationCode, err := genValidateCode()
		if err != nil {
			l.Errorf("generate verification code: %v", err)
			return nil, SendVerificationError
		}

		if err = mailer.Send(mailer.SMTPConfig{
			From:       email.From,
			Host:       email.Host,
			Port:       cast.ToInt(email.Port),
			Username:   email.Username,
			Password:   email.Password,
			EnableSSL:  cast.ToBool(email.EnableSsl),
			VerifyCert: cast.ToBool(email.IsVerify),
		}, req.Email, "AgentRazor 验证码", fmt.Sprintf("AgentRazor 邮箱验证码：%s（5 分钟内有效）", verificationCode)); err != nil {
			l.Errorf("send verification email: %v", err)
			return nil, SendVerificationError
		}

		if err = l.svcCtx.Cache.SetWithExpireCtx(context.Background(), verificationCacheKey(verificationUuid), encodedEmailVerification(req.Email, verificationCode), time.Minute*5); err != nil {
			return nil, SendVerificationError
		}

		return &types.SendVerificationCodeResponse{
			VerificationUuid: verificationUuid,
		}, nil
	}
	return nil, errors.New("暂不支持手机号验证码")
}

func genValidateCode() (string, error) {
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}
