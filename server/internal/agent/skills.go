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
	Name        string
	Description string
	Enabled     bool
	Path        string
	Scope       string
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

type PluginSkillSyncResult struct {
	Installed []string
	Skipped   []string
}

type SkillError struct {
	Kind    string
	Message string
}

func (e *SkillError) Error() string {
	return e.Message
}

type archiveEntry struct {
	Path       string
	Directory  bool
	Executable bool
	Data       []byte
}

func (s *Service) ListSkills(ctx context.Context) ([]Skill, error) {
	return s.listSkills(ctx, false)
}

func (s *Service) listSkills(ctx context.Context, forceReload bool) ([]Skill, error) {
	result, err := s.call(ctx, "skills/list", map[string]any{
		"cwds":        []string{s.workspace},
		"forceReload": forceReload,
	})
	if err != nil {
		return nil, err
	}
	return managedSkillsFromListResponse(result, filepath.Join(s.codexHome, "skills"))
}

func managedSkillsFromListResponse(response map[string]any, managedRoot string) ([]Skill, error) {
	values, ok := response["data"].([]any)
	if !ok {
		return nil, errors.New("Codex skills/list response did not contain data")
	}

	byName := make(map[string]Skill)
	for _, value := range values {
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		rawSkills, _ := entry["skills"].([]any)
		for _, rawSkill := range rawSkills {
			metadata, ok := rawSkill.(map[string]any)
			if !ok {
				continue
			}
			skill := Skill{
				Name:        stringValue(metadata["name"]),
				Description: stringValue(metadata["description"]),
				Enabled:     boolValue(metadata["enabled"]),
				Path:        filepath.Clean(stringValue(metadata["path"])),
				Scope:       stringValue(metadata["scope"]),
			}
			if !isManagedSkill(skill, managedRoot) {
				continue
			}
			byName[skill.Name] = skill
		}
	}

	result := make([]Skill, 0, len(byName))
	for _, skill := range byName {
		result = append(result, skill)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

func isManagedSkill(skill Skill, managedRoot string) bool {
	if skill.Name == "" || safeSkillName(skill.Name) != skill.Name || !filepath.IsAbs(skill.Path) ||
		filepath.Base(skill.Path) != "SKILL.md" {
		return false
	}
	relative, err := filepath.Rel(filepath.Clean(managedRoot), filepath.Dir(skill.Path))
	return err == nil && relative != "." && filepath.Dir(relative) == "." && safeSkillName(relative) == relative
}

func (s *Service) managedSkill(ctx context.Context, name string) (Skill, error) {
	name = safeSkillName(name)
	if name == "" {
		return Skill{}, errors.New("skill name is required")
	}
	skills, err := s.listSkills(ctx, false)
	if err != nil {
		return Skill{}, err
	}
	for _, skill := range skills {
		if skill.Name == name {
			return skill, nil
		}
	}
	return Skill{}, errors.New("skill not found")
}

func (s *Service) SkillDetail(ctx context.Context, name, file string) (SkillDetail, error) {
	skill, err := s.managedSkill(ctx, name)
	if err != nil {
		return SkillDetail{}, err
	}
	root := filepath.Dir(skill.Path)
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
		Skill:       skill,
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
	if len(content) > int(maxSkillFileSize) {
		return errors.New("file content is larger than 2 MiB")
	}
	skill, err := s.managedSkill(ctx, name)
	if err != nil {
		return err
	}
	root := filepath.Dir(skill.Path)
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
	if err := s.writeFile(ctx, target, []byte(content)); err != nil {
		return err
	}
	_, err = s.listSkills(ctx, true)
	return err
}

func (s *Service) DeleteSkill(ctx context.Context, name string) error {
	skill, err := s.managedSkill(ctx, name)
	if err != nil {
		return err
	}
	target := filepath.Dir(skill.Path)
	metadata, err := s.metadata(ctx, target)
	if err != nil {
		return err
	}
	if metadata.IsSymlink || !metadata.IsDirectory {
		return errors.New("skill is not a directory")
	}
	if err := s.remove(ctx, target, true, false); err != nil {
		return err
	}
	_, err = s.listSkills(ctx, true)
	return err
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
	return s.installSkillEntries(ctx, name, entries)
}

func (s *Service) installSkillEntries(ctx context.Context, name string, entries []archiveEntry) (Skill, error) {
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
		if entry.Executable {
			if err := s.makeSkillScriptExecutable(ctx, target, entryPath); err != nil {
				return Skill{}, err
			}
		}
	}
	installedSkills, err := s.listSkills(ctx, true)
	if err != nil {
		return Skill{}, err
	}
	for _, installed := range installedSkills {
		if filepath.Dir(installed.Path) == target {
			committed = true
			return installed, nil
		}
	}
	return Skill{}, skillError("archive_invalid", "Codex did not discover the installed skill")
}

func (s *Service) SyncPluginSkills(ctx context.Context, pluginsRoot string) (PluginSkillSyncResult, error) {
	var result PluginSkillSyncResult
	bundled, err := pluginSkillDirectories(pluginsRoot)
	if err != nil {
		return result, err
	}
	if len(bundled) == 0 {
		return result, nil
	}

	installed, err := s.listSkills(ctx, true)
	if err != nil {
		return result, err
	}
	pending, skipped := splitPluginSkills(bundled, installed)
	result.Skipped = skipped

	for _, item := range pending {
		entries, err := readSkillDirectory(item.Path)
		if err != nil {
			return result, fmt.Errorf("package plugin skill %q: %w", item.Name, err)
		}
		skill, err := s.installSkillEntries(ctx, item.Name, entries)
		if err != nil {
			return result, fmt.Errorf("install plugin skill %q: %w", item.Name, err)
		}
		result.Installed = append(result.Installed, skill.Name)
	}
	return result, nil
}

func splitPluginSkills(bundled []pluginSkillDirectory, installed []Skill) ([]pluginSkillDirectory, []string) {
	existing := make(map[string]struct{}, len(installed)*2)
	for _, skill := range installed {
		existing[skill.Name] = struct{}{}
		existing[filepath.Base(filepath.Dir(skill.Path))] = struct{}{}
	}

	pending := make([]pluginSkillDirectory, 0, len(bundled))
	skipped := make([]string, 0, len(bundled))
	for _, item := range bundled {
		if _, ok := existing[item.Name]; ok {
			skipped = append(skipped, item.Name)
			continue
		}
		pending = append(pending, item)
		existing[item.Name] = struct{}{}
	}
	return pending, skipped
}

type pluginSkillDirectory struct {
	Name string
	Path string
}

func pluginSkillDirectories(pluginsRoot string) ([]pluginSkillDirectory, error) {
	pluginsRoot = filepath.Clean(strings.TrimSpace(pluginsRoot))
	if pluginsRoot == "" || pluginsRoot == "." {
		return nil, nil
	}
	plugins, err := os.ReadDir(pluginsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result []pluginSkillDirectory
	for _, plugin := range plugins {
		if !plugin.IsDir() || plugin.Type()&os.ModeSymlink != 0 {
			continue
		}
		skillsRoot := filepath.Join(pluginsRoot, plugin.Name(), "skills")
		skills, err := os.ReadDir(skillsRoot)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, skill := range skills {
			if !skill.IsDir() || skill.Type()&os.ModeSymlink != 0 || safeSkillName(skill.Name()) != skill.Name() {
				continue
			}
			result = append(result, pluginSkillDirectory{
				Name: skill.Name(),
				Path: filepath.Join(skillsRoot, skill.Name()),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Path < result[j].Path
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func readSkillDirectory(root string) ([]archiveEntry, error) {
	root = filepath.Clean(root)
	var (
		entries  []archiveEntry
		expanded int64
	)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if len(entries) >= maxSkillArchiveEntries {
			return skillError("archive_too_many_entries", "skill contains too many entries")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return skillError("archive_invalid", fmt.Sprintf("unsupported skill entry %q", path))
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			entries = append(entries, archiveEntry{Path: relative, Directory: true})
			return nil
		}
		if info.Size() < 0 || info.Size() > maxSkillExtractedSize-expanded {
			return skillError("archive_expanded_too_large", "plugin skill is too large")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		expanded += int64(len(content))
		entries = append(entries, archiveEntry{
			Path:       relative,
			Executable: isExecutableSkillScript(relative, info.Mode()),
			Data:       content,
		})
		return nil
	})
	return entries, err
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
		result = append(result, archiveEntry{
			Path:       entryPath,
			Executable: isExecutableSkillScript(entryPath, mode),
			Data:       content,
		})
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
			result = append(result, archiveEntry{
				Path:       entryPath,
				Executable: isExecutableSkillScript(entryPath, os.FileMode(header.Mode)),
				Data:       content,
			})
		default:
			return nil, skillError("archive_invalid", fmt.Sprintf("unsupported archive entry %q", header.Name))
		}
	}
	return result, nil
}

func isExecutableSkillScript(path string, mode os.FileMode) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	return strings.HasPrefix(path, "scripts/") && mode.Perm()&0o111 != 0
}

func (s *Service) makeSkillScriptExecutable(ctx context.Context, skillRoot, path string) error {
	result, err := s.call(ctx, "command/exec", map[string]any{
		"command":        []string{"chmod", "0755", path},
		"cwd":            skillRoot,
		"timeoutMs":      5000,
		"outputBytesCap": 4096,
		"sandboxPolicy": map[string]any{
			"type":          "externalSandbox",
			"networkAccess": "restricted",
		},
	})
	if err != nil {
		return fmt.Errorf("set executable permission on %q: %w", filepath.Base(path), err)
	}
	exitCode, ok := integerValue(result["exitCode"])
	if !ok || exitCode != 0 {
		stderr := strings.TrimSpace(stringValue(result["stderr"]))
		if stderr == "" {
			stderr = "chmod failed"
		}
		return fmt.Errorf("set executable permission on %q: %s", filepath.Base(path), stderr)
	}
	return nil
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), typed == float64(int64(typed))
	default:
		return 0, false
	}
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
