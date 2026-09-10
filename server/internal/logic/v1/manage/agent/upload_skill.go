package agent

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jzero-io/jzero/core/status"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/errcodes"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

var (
	errSkillArchiveRequired         = errors.New("skill archive file is required")
	errSkillArchiveEmpty            = errors.New("skill archive is empty")
	errSkillArchiveTooLarge         = errors.New("skill archive is too large")
	errSkillArchiveUnsupported      = errors.New("unsupported skill archive format")
	errSkillArchiveInvalid          = errors.New("invalid skill archive")
	errSkillArchiveTooManyEntries   = errors.New("skill archive contains too many entries")
	errSkillArchiveExpandedTooLarge = errors.New("expanded skill archive is too large")
	errSkillArchiveUnsafeEntry      = errors.New("skill archive contains an unsafe entry")
	errSkillNameInvalid             = errors.New("invalid skill name")
	errSkillManifestMissing         = errors.New("skill archive must contain SKILL.md")
)

type UploadSkill struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUploadSkill(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UploadSkill {
	return &UploadSkill{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *UploadSkill) UploadSkill(req *types.UploadSkillRequest) (resp *types.UploadSkillResponse, err error) {
	if err := l.r.ParseMultipartForm(100 << 20); err != nil {
		return nil, l.skillArchiveError(errSkillArchiveInvalid)
	}
	file, header, err := l.r.FormFile("file")
	if err != nil {
		return nil, l.skillArchiveError(errSkillArchiveRequired)
	}
	defer file.Close()
	if header == nil {
		return nil, l.skillArchiveError(errSkillArchiveRequired)
	}
	name := strings.TrimSpace(l.r.FormValue("name"))
	installed, err := l.svcCtx.AgentService.InstallSkill(l.ctx, name, header.Filename, header.Size, file)
	if err != nil {
		return nil, l.skillArchiveError(normalizeSkillError(err))
	}
	return &types.UploadSkillResponse{Name: installed.Name}, nil
}

func (l *UploadSkill) skillArchiveError(err error) error {
	code := 0
	key := ""
	switch {
	case errors.Is(err, errSkillArchiveRequired):
		code, key = errcodes.SkillArchiveRequiredCode, "manage.agent.skill.archiveRequired"
	case errors.Is(err, errSkillArchiveEmpty):
		code, key = errcodes.SkillArchiveEmptyCode, "manage.agent.skill.archiveEmpty"
	case errors.Is(err, errSkillArchiveTooLarge):
		code, key = errcodes.SkillArchiveTooLargeCode, "manage.agent.skill.archiveTooLarge"
	case errors.Is(err, errSkillArchiveUnsupported):
		code, key = errcodes.SkillArchiveUnsupportedFormatCode, "manage.agent.skill.archiveUnsupportedFormat"
	case errors.Is(err, errSkillArchiveInvalid):
		code, key = errcodes.SkillArchiveInvalidCode, "manage.agent.skill.archiveInvalid"
	case errors.Is(err, errSkillArchiveTooManyEntries):
		code, key = errcodes.SkillArchiveTooManyEntriesCode, "manage.agent.skill.archiveTooManyEntries"
	case errors.Is(err, errSkillArchiveExpandedTooLarge):
		code, key = errcodes.SkillArchiveExpandedTooLargeCode, "manage.agent.skill.archiveExpandedTooLarge"
	case errors.Is(err, errSkillArchiveUnsafeEntry):
		code, key = errcodes.SkillArchiveUnsafeEntryCode, "manage.agent.skill.archiveUnsafeEntry"
	case errors.Is(err, errSkillNameInvalid):
		code, key = errcodes.SkillNameInvalidCode, "manage.agent.skill.nameInvalid"
	case errors.Is(err, errSkillManifestMissing):
		code, key = errcodes.SkillManifestMissingCode, "manage.agent.skill.manifestMissing"
	default:
		return err
	}
	return status.ErrorMessage(status.Code(code), l.svcCtx.Trans.Trans(l.ctx, key))
}

func normalizeSkillError(err error) error {
	var remote *agentdomain.SkillError
	if !errors.As(err, &remote) {
		return err
	}
	var target error
	switch remote.Kind {
	case "archive_required":
		target = errSkillArchiveRequired
	case "archive_empty":
		target = errSkillArchiveEmpty
	case "archive_too_large":
		target = errSkillArchiveTooLarge
	case "archive_unsupported":
		target = errSkillArchiveUnsupported
	case "archive_invalid":
		target = errSkillArchiveInvalid
	case "archive_too_many_entries":
		target = errSkillArchiveTooManyEntries
	case "archive_expanded_too_large":
		target = errSkillArchiveExpandedTooLarge
	case "archive_unsafe_entry":
		target = errSkillArchiveUnsafeEntry
	case "name_invalid":
		target = errSkillNameInvalid
	case "manifest_missing":
		target = errSkillManifestMissing
	default:
		return err
	}
	return errors.Join(target, errors.New(remote.Message))
}
