package agent

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	managetypes "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
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

const (
	maxSkillArchiveSize    int64 = 100 << 20
	maxSkillExtractedSize  int64 = 128 << 20
	maxSkillArchiveEntries       = 4096
)

func installSkillArchive(codexHome, explicitName, archiveName string, file multipart.File, size int64) (managetypes.UploadSkillResponse, error) {
	if size <= 0 {
		return managetypes.UploadSkillResponse{}, errSkillArchiveEmpty
	}
	if size > maxSkillArchiveSize {
		return managetypes.UploadSkillResponse{}, errSkillArchiveTooLarge
	}
	name := strings.TrimSpace(explicitName)
	if name == "" {
		name = archiveName
	}
	lowerName := strings.ToLower(strings.TrimSpace(archiveName))
	switch {
	case strings.HasSuffix(lowerName, ".tar.gz"):
		return installSkillTarGz(codexHome, name, file, size)
	case strings.HasSuffix(lowerName, ".zip"):
		return installSkillZip(codexHome, name, file, size)
	default:
		return managetypes.UploadSkillResponse{}, errSkillArchiveUnsupported
	}
}

func installSkillTarGz(codexHome, explicitName string, file multipart.File, size int64) (managetypes.UploadSkillResponse, error) {
	if size <= 0 {
		return managetypes.UploadSkillResponse{}, errSkillArchiveEmpty
	}
	name := safeSkillName(explicitName)
	if name == "" {
		return managetypes.UploadSkillResponse{}, errSkillNameInvalid
	}
	root, err := skillsRoot(codexHome)
	if err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	dest, err := os.MkdirTemp(root, "."+name+"-upload-")
	if err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	defer os.RemoveAll(dest)

	gzipReader, err := gzip.NewReader(io.LimitReader(file, maxSkillArchiveSize))
	if err != nil {
		return managetypes.UploadSkillResponse{}, fmt.Errorf("%w: %v", errSkillArchiveInvalid, err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	var extractedSize int64
	entryCount := 0
	for {
		header, nextErr := tarReader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return managetypes.UploadSkillResponse{}, fmt.Errorf("%w: %v", errSkillArchiveInvalid, nextErr)
		}
		entryCount++
		if entryCount > maxSkillArchiveEntries {
			return managetypes.UploadSkillResponse{}, errSkillArchiveTooManyEntries
		}

		entryName, entryErr := skillTarEntryName(header.Name, name)
		if entryErr != nil {
			return managetypes.UploadSkillResponse{}, entryErr
		}
		if entryName == "" {
			continue
		}
		entryPath := filepath.Join(dest, entryName)
		if rel, relErr := filepath.Rel(dest, entryPath); relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return managetypes.UploadSkillResponse{}, fmt.Errorf("%w: %q", errSkillArchiveUnsafeEntry, header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(entryPath, skillArchiveMode(header.Mode, 0o755)); err != nil {
				return managetypes.UploadSkillResponse{}, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > maxSkillExtractedSize-extractedSize {
				return managetypes.UploadSkillResponse{}, errSkillArchiveExpandedTooLarge
			}
			extractedSize += header.Size
			if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
				return managetypes.UploadSkillResponse{}, err
			}
			dst, err := os.OpenFile(entryPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, skillArchiveMode(header.Mode, 0o644))
			if err != nil {
				return managetypes.UploadSkillResponse{}, err
			}
			written, copyErr := io.Copy(dst, tarReader)
			closeErr := dst.Close()
			if copyErr != nil || closeErr != nil {
				return managetypes.UploadSkillResponse{}, errors.Join(copyErr, closeErr)
			}
			if written != header.Size {
				return managetypes.UploadSkillResponse{}, fmt.Errorf("%w: incomplete entry %q", errSkillArchiveInvalid, header.Name)
			}
		default:
			return managetypes.UploadSkillResponse{}, fmt.Errorf("%w: unsupported entry %q", errSkillArchiveInvalid, header.Name)
		}
	}
	if info, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil || !info.Mode().IsRegular() {
		return managetypes.UploadSkillResponse{}, errSkillManifestMissing
	}
	if err := commitSkillDirectory(root, name, dest); err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	return managetypes.UploadSkillResponse{Name: name}, nil
}

func commitSkillDirectory(root, name, staged string) error {
	dest := filepath.Join(root, name)
	backup, err := os.MkdirTemp(root, "."+name+"-backup-")
	if err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}

	hadExisting := false
	if _, err := os.Lstat(dest); err == nil {
		if err := os.Rename(dest, backup); err != nil {
			return err
		}
		hadExisting = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Rename(staged, dest); err != nil {
		if hadExisting {
			_ = os.Rename(backup, dest)
		}
		return err
	}
	if hadExisting {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func skillTarEntryName(rawName, skillName string) (string, error) {
	rawName = strings.ReplaceAll(rawName, "\\", "/")
	entryName := filepath.Clean(filepath.FromSlash(rawName))
	if entryName == "." {
		return "", nil
	}
	if filepath.IsAbs(entryName) || entryName == ".." || strings.HasPrefix(entryName, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q", errSkillArchiveUnsafeEntry, rawName)
	}
	if entryName == skillName {
		return "", nil
	}
	parts := strings.Split(entryName, string(filepath.Separator))
	if len(parts) > 1 && safeSkillName(parts[0]) == skillName {
		entryName = filepath.Join(parts[1:]...)
	}
	return entryName, nil
}

func skillArchiveMode(mode int64, fallback os.FileMode) os.FileMode {
	result := os.FileMode(mode).Perm()
	if result == 0 {
		return fallback
	}
	return result
}
