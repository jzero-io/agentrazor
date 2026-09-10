package agent

import (
	"errors"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	managetypes "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

func normalizeRuntimeSkillError(err error) error {
	var remote *agentdomain.RuntimeRemoteError
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

func manageSkills(values []agentdomain.RuntimeSkill) []managetypes.Skill {
	result := make([]managetypes.Skill, 0, len(values))
	for _, value := range values {
		result = append(result, managetypes.Skill{Name: value.Name})
	}
	return result
}

func manageSkillDetail(value agentdomain.RuntimeSkillDetail) *managetypes.SkillDetailResponse {
	return &managetypes.SkillDetailResponse{
		Skill:       managetypes.Skill{Name: value.Skill.Name},
		Files:       manageSkillFiles(value.Files),
		CurrentFile: value.CurrentFile,
		Content:     value.Content,
	}
}

func manageSkillFiles(values []agentdomain.RuntimeSkillFile) []managetypes.SkillFile {
	result := make([]managetypes.SkillFile, 0, len(values))
	for _, value := range values {
		result = append(result, managetypes.SkillFile{
			Name: value.Name, Path: value.Path, Type: value.Type, Children: manageSkillFiles(value.Children),
		})
	}
	return result
}
