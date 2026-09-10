package agent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxSkillArchiveSize     int64 = 100 << 20
	maxSkillExtractedSize   int64 = 128 << 20
	maxSkillFileSize        int64 = 2 << 20
	maxSkillArchiveEntries        = 4096
	hiddenPluginSkillMarker       = ".agentrazor-plugin-skill.json"
)

type Skill struct {
	Name string
}

type SkillFile struct {
	Name     string
	Path     string
	Type     string
	Children []SkillFile
}

type SkillDetail struct {
	Skill       Skill
	Files       []SkillFile
	CurrentFile string
	Content     string
}

type SkillError struct {
	Kind    string
	Message string
}

func (e *SkillError) Error() string {
	return e.Message
}

type archiveEntry struct {
	Path      string
	Directory bool
	Data      []byte
}

func (s *Service) ListSkills(ctx context.Context) ([]Skill, error) {
	root := filepath.Join(s.codexHome, "skills")
	entries, err := s.readDirectory(ctx, root)
	if isRemoteNotFound(err) {
		return []Skill{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDirectory && entry.Name != ".system" {
			result = append(result, Skill{Name: entry.Name})
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

func (s *Service) SkillDetail(ctx context.Context, name, file string) (SkillDetail, error) {
	name = safeSkillName(name)
	if name == "" {
		return SkillDetail{}, errors.New("skill name is required")
	}
	root := filepath.Join(s.codexHome, "skills", name)
	metadata, err := s.metadata(ctx, root)
	if err != nil {
		return SkillDetail{}, err
	}
	if metadata.IsSymlink || !metadata.IsDirectory {
		return SkillDetail{}, errors.New("skill is not a directory")
	}
	files, err := s.skillFileTree(ctx, root, "")
	if err != nil {
		return SkillDetail{}, err
	}
	currentFile := strings.TrimSpace(file)
	if currentFile == "" {
		currentFile = defaultSkillFile(files)
	}
	content, err := s.readSkillFile(ctx, root, currentFile)
	if err != nil {
		return SkillDetail{}, err
	}
	return SkillDetail{
		Skill:       Skill{Name: name},
		Files:       files,
		CurrentFile: filepath.ToSlash(currentFile),
		Content:     content,
	}, nil
}

func (s *Service) skillFileTree(ctx context.Context, root, relative string) ([]SkillFile, error) {
	directory := root
	if relative != "" {
		directory = filepath.Join(root, relative)
	}
	entries, err := s.readDirectory(ctx, directory)
	if err != nil {
		return nil, err
	}
	result := make([]SkillFile, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "" || strings.ContainsAny(entry.Name, "/\\") || isHiddenSkillFile(entry.Name) {
			continue
		}
		childRelative := filepath.Join(relative, entry.Name)
		item := SkillFile{
			Name: entry.Name,
			Path: filepath.ToSlash(childRelative),
			Type: "file",
		}
		switch {
		case entry.IsDirectory:
			item.Type = "directory"
			item.Children, err = s.skillFileTree(ctx, root, childRelative)
			if err != nil {
				continue
			}
		case entry.IsFile:
		default:
			continue
		}
		result = append(result, item)
	}
	sortSkillFiles(result)
	return result, nil
}

func sortSkillFiles(files []SkillFile) {
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
}

func defaultSkillFile(files []SkillFile) string {
	for _, item := range files {
		if item.Type == "file" && item.Path == "SKILL.md" {
			return item.Path
		}
	}
	var walk func([]SkillFile) string
	walk = func(items []SkillFile) string {
		for _, item := range items {
			if item.Type == "file" {
				return item.Path
			}
			if next := walk(item.Children); next != "" {
				return next
			}
		}
		return ""
	}
	return walk(files)
}

func (s *Service) readSkillFile(ctx context.Context, root, file string) (string, error) {
	target, err := skillFilePath(root, file)
	if err != nil {
		return "", err
	}
	metadata, err := s.metadata(ctx, target)
	if err != nil {
		return "", err
	}
	if metadata.IsSymlink || !metadata.IsFile {
		return "", errors.New("only regular files can be read")
	}
	if metadata.Size > maxSkillFileSize {
		return "", errors.New("file is larger than 2 MiB")
	}
	data, err := s.readFile(ctx, target, maxSkillFileSize)
	return string(data), err
}

func (s *Service) UpdateSkillFile(ctx context.Context, name, file, content string) error {
	name = safeSkillName(name)
	if name == "" {
		return errors.New("skill name is required")
	}
	if len(content) > int(maxSkillFileSize) {
		return errors.New("file content is larger than 2 MiB")
	}
	root := filepath.Join(s.codexHome, "skills", name)
	rootMetadata, err := s.metadata(ctx, root)
	if err != nil {
		return err
	}
	if rootMetadata.IsSymlink || !rootMetadata.IsDirectory {
		return errors.New("skill is not a directory")
	}
	target, err := skillFilePath(root, file)
	if err != nil {
		return err
	}
	metadata, err := s.metadata(ctx, target)
	if err != nil {
		return err
	}
	if metadata.IsSymlink || !metadata.IsFile {
		return errors.New("only regular files can be edited")
	}
	return s.writeFile(ctx, target, []byte(content))
}

func (s *Service) DeleteSkill(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == ".system" {
		return errors.New("system skills cannot be deleted")
	}
	name = safeSkillName(name)
	if name == "" {
		return errors.New("skill name is required")
	}
	target := filepath.Join(s.codexHome, "skills", name)
	metadata, err := s.metadata(ctx, target)
	if err != nil {
		return err
	}
	if metadata.IsSymlink || !metadata.IsDirectory {
		return errors.New("skill is not a directory")
	}
	return s.remove(ctx, target, true, false)
}

func (s *Service) InstallSkill(ctx context.Context, explicitName, archiveName string, size int64, archive io.Reader) (Skill, error) {
	data, err := readArchiveBytes(archive, size, maxSkillArchiveSize)
	if err != nil {
		return Skill{}, err
	}
	name := strings.TrimSpace(explicitName)
	if name == "" {
		name = archiveName
	}
	name = safeSkillName(name)
	if name == "" {
		return Skill{}, skillError("name_invalid", "invalid skill name")
	}

	var entries []archiveEntry
	switch lower := strings.ToLower(strings.TrimSpace(archiveName)); {
	case strings.HasSuffix(lower, ".zip"):
		entries, err = readSkillZip(data, name)
	case strings.HasSuffix(lower, ".tar.gz"):
		entries, err = readSkillTarGz(data, name)
	default:
		err = skillError("archive_unsupported", "unsupported skill archive format")
	}
	if err != nil {
		return Skill{}, err
	}
	hasManifest := false
	for _, entry := range entries {
		if !entry.Directory && filepath.ToSlash(entry.Path) == "SKILL.md" {
			hasManifest = true
			break
		}
	}
	if !hasManifest {
		return Skill{}, skillError("manifest_missing", "skill archive must contain SKILL.md")
	}

	root := filepath.Join(s.codexHome, "skills")
	target := filepath.Join(root, name)
	if err := s.createDirectory(ctx, root); err != nil {
		return Skill{}, err
	}
	if err := s.remove(ctx, target, true, true); err != nil && !isRemoteNotFound(err) {
		return Skill{}, err
	}
	if err := s.createDirectory(ctx, target); err != nil {
		return Skill{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = s.remove(context.Background(), target, true, true)
		}
	}()
	for _, entry := range entries {
		entryPath, pathErr := safeChild(target, entry.Path)
		if pathErr != nil {
			return Skill{}, skillError("archive_unsafe_entry", pathErr.Error())
		}
		if entry.Directory {
			if err := s.createDirectory(ctx, entryPath); err != nil {
				return Skill{}, err
			}
			continue
		}
		if err := s.createDirectory(ctx, filepath.Dir(entryPath)); err != nil {
			return Skill{}, err
		}
		if err := s.writeFile(ctx, entryPath, entry.Data); err != nil {
			return Skill{}, err
		}
	}
	committed = true
	return Skill{Name: name}, nil
}

func readArchiveBytes(reader io.Reader, size, limit int64) ([]byte, error) {
	if size <= 0 {
		return nil, skillError("archive_empty", "skill archive is empty")
	}
	if size > limit {
		return nil, skillError("archive_too_large", "skill archive is too large")
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, skillError("archive_too_large", "skill archive is too large")
	}
	return data, nil
}

func readSkillZip(data []byte, skillName string) ([]archiveEntry, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, skillError("archive_invalid", fmt.Sprintf("invalid skill archive: %v", err))
	}
	if len(reader.File) > maxSkillArchiveEntries {
		return nil, skillError("archive_too_many_entries", "skill archive contains too many entries")
	}
	result := make([]archiveEntry, 0, len(reader.File))
	var expanded int64
	for _, item := range reader.File {
		entryPath, err := normalizeArchivePath(item.Name, skillName)
		if err != nil {
			return nil, err
		}
		if entryPath == "" {
			continue
		}
		mode := item.Mode()
		if mode&os.ModeSymlink != 0 || (!item.FileInfo().IsDir() && !mode.IsRegular()) {
			return nil, skillError("archive_invalid", fmt.Sprintf("unsupported archive entry %q", item.Name))
		}
		if item.FileInfo().IsDir() {
			result = append(result, archiveEntry{Path: entryPath, Directory: true})
			continue
		}
		if item.UncompressedSize64 > uint64(maxSkillExtractedSize-expanded) {
			return nil, skillError("archive_expanded_too_large", "expanded skill archive is too large")
		}
		source, err := item.Open()
		if err != nil {
			return nil, skillError("archive_invalid", err.Error())
		}
		content, readErr := io.ReadAll(io.LimitReader(source, maxSkillExtractedSize-expanded+1))
		closeErr := source.Close()
		if readErr != nil || closeErr != nil {
			return nil, skillError("archive_invalid", errors.Join(readErr, closeErr).Error())
		}
		expanded += int64(len(content))
		if expanded > maxSkillExtractedSize {
			return nil, skillError("archive_expanded_too_large", "expanded skill archive is too large")
		}
		result = append(result, archiveEntry{Path: entryPath, Data: content})
	}
	return result, nil
}

func readSkillTarGz(data []byte, skillName string) ([]archiveEntry, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, skillError("archive_invalid", fmt.Sprintf("invalid skill archive: %v", err))
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	result := make([]archiveEntry, 0)
	var expanded int64
	for entryCount := 0; ; entryCount++ {
		header, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, skillError("archive_invalid", nextErr.Error())
		}
		if entryCount >= maxSkillArchiveEntries {
			return nil, skillError("archive_too_many_entries", "skill archive contains too many entries")
		}
		entryPath, err := normalizeArchivePath(header.Name, skillName)
		if err != nil {
			return nil, err
		}
		if entryPath == "" {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			result = append(result, archiveEntry{Path: entryPath, Directory: true})
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > maxSkillExtractedSize-expanded {
				return nil, skillError("archive_expanded_too_large", "expanded skill archive is too large")
			}
			content, err := io.ReadAll(io.LimitReader(reader, header.Size+1))
			if err != nil || int64(len(content)) != header.Size {
				return nil, skillError("archive_invalid", fmt.Sprintf("incomplete archive entry %q", header.Name))
			}
			expanded += int64(len(content))
			result = append(result, archiveEntry{Path: entryPath, Data: content})
		default:
			return nil, skillError("archive_invalid", fmt.Sprintf("unsupported archive entry %q", header.Name))
		}
	}
	return result, nil
}

func normalizeArchivePath(rawName, skillName string) (string, error) {
	rawName = strings.ReplaceAll(rawName, "\\", "/")
	entryPath := filepath.Clean(filepath.FromSlash(rawName))
	if entryPath == "." {
		return "", nil
	}
	if filepath.IsAbs(entryPath) || entryPath == ".." || strings.HasPrefix(entryPath, ".."+string(filepath.Separator)) {
		return "", skillError("archive_unsafe_entry", fmt.Sprintf("unsafe archive entry %q", rawName))
	}
	if entryPath == skillName {
		return "", nil
	}
	parts := strings.Split(entryPath, string(filepath.Separator))
	if len(parts) > 1 && safeSkillName(parts[0]) == skillName {
		entryPath = filepath.Join(parts[1:]...)
	}
	if entryPath == "." || entryPath == ".." || strings.HasPrefix(entryPath, ".."+string(filepath.Separator)) {
		return "", skillError("archive_unsafe_entry", fmt.Sprintf("unsafe archive entry %q", rawName))
	}
	return entryPath, nil
}

func safeSkillName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	lowerName := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lowerName, ".tar.gz"):
		name = name[:len(name)-len(".tar.gz")]
	case strings.HasSuffix(lowerName, ".zip"):
		name = name[:len(name)-len(".zip")]
	}
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)
	return strings.Trim(name, "-._")
}

func skillFilePath(root, file string) (string, error) {
	file = filepath.Clean(strings.TrimSpace(file))
	if file == "" || file == "." || isHiddenSkillPath(file) {
		return "", errors.New("file is required")
	}
	return safeChild(root, file)
}

func isHiddenSkillFile(name string) bool {
	return name == hiddenPluginSkillMarker
}

func isHiddenSkillPath(file string) bool {
	for _, part := range strings.Split(filepath.ToSlash(file), "/") {
		if isHiddenSkillFile(part) {
			return true
		}
	}
	return false
}

func skillError(kind, message string) error {
	return &SkillError{Kind: kind, Message: message}
}
