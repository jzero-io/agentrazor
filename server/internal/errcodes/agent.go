package errcodes

import "github.com/jzero-io/jzero/core/status"

const (
	AgentDisabledCode                 = 10401
	FiveHourQuotaExceededCode         = 10402
	SevenDayQuotaExceededCode         = 10403
	SkillArchiveRequiredCode          = 10404
	SkillArchiveEmptyCode             = 10405
	SkillArchiveTooLargeCode          = 10406
	SkillArchiveUnsupportedFormatCode = 10407
	SkillArchiveInvalidCode           = 10408
	SkillArchiveTooManyEntriesCode    = 10409
	SkillArchiveExpandedTooLargeCode  = 10410
	SkillArchiveUnsafeEntryCode       = 10411
	SkillNameInvalidCode              = 10412
	SkillManifestMissingCode          = 10413
)

func init() {
	status.Register(AgentDisabledCode)
	status.Register(FiveHourQuotaExceededCode)
	status.Register(SevenDayQuotaExceededCode)
	status.Register(SkillArchiveRequiredCode)
	status.Register(SkillArchiveEmptyCode)
	status.Register(SkillArchiveTooLargeCode)
	status.Register(SkillArchiveUnsupportedFormatCode)
	status.Register(SkillArchiveInvalidCode)
	status.Register(SkillArchiveTooManyEntriesCode)
	status.Register(SkillArchiveExpandedTooLargeCode)
	status.Register(SkillArchiveUnsafeEntryCode)
	status.Register(SkillNameInvalidCode)
	status.Register(SkillManifestMissingCode)
}
