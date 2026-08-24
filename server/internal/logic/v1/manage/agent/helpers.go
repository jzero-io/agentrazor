package agent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	managetypes "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
	"github.com/pelletier/go-toml/v2"
)

func skillsRoot(codexHome string) (string, error) {
	if strings.TrimSpace(codexHome) == "" {
		codexHome = "data/codex-home"
	}
	root, err := filepath.Abs(filepath.Join(codexHome, "skills"))
	if err != nil {
		return "", err
	}
	return root, nil
}

func listSkills(codexHome string) ([]managetypes.Skill, error) {
	root, err := skillsRoot(codexHome)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []managetypes.Skill{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]managetypes.Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".system" {
			continue
		}
		result = append(result, managetypes.Skill{Name: entry.Name()})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func installSkillZip(codexHome, explicitName string, file multipart.File, size int64) (managetypes.UploadSkillResponse, error) {
	if size <= 0 {
		return managetypes.UploadSkillResponse{}, errors.New("skill zip is empty")
	}
	data, err := io.ReadAll(io.LimitReader(file, 64<<20))
	if err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	name := strings.TrimSpace(explicitName)
	if name == "" {
		name = inferSkillName(reader.File)
	}
	name = safeSkillName(name)
	if name == "" {
		return managetypes.UploadSkillResponse{}, errors.New("cannot infer skill name")
	}
	root, err := skillsRoot(codexHome)
	if err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	dest := filepath.Join(root, name)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return managetypes.UploadSkillResponse{}, err
	}
	for _, entry := range reader.File {
		entryName := strings.TrimPrefix(filepath.Clean(entry.Name), string(filepath.Separator))
		parts := strings.Split(entryName, string(filepath.Separator))
		if len(parts) > 1 && safeSkillName(parts[0]) == name {
			entryName = filepath.Join(parts[1:]...)
		}
		if entryName == "." || strings.HasPrefix(entryName, "..") || filepath.IsAbs(entryName) {
			return managetypes.UploadSkillResponse{}, fmt.Errorf("unsafe zip entry %q", entry.Name)
		}
		path := filepath.Join(dest, entryName)
		if rel, err := filepath.Rel(dest, path); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return managetypes.UploadSkillResponse{}, fmt.Errorf("unsafe zip entry %q", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return managetypes.UploadSkillResponse{}, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return managetypes.UploadSkillResponse{}, err
		}
		src, err := entry.Open()
		if err != nil {
			return managetypes.UploadSkillResponse{}, err
		}
		dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode())
		if err != nil {
			_ = src.Close()
			return managetypes.UploadSkillResponse{}, err
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := errors.Join(src.Close(), dst.Close())
		if copyErr != nil || closeErr != nil {
			return managetypes.UploadSkillResponse{}, errors.Join(copyErr, closeErr)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
		return managetypes.UploadSkillResponse{}, errors.New("skill zip must contain SKILL.md")
	}
	return managetypes.UploadSkillResponse{Name: name}, nil
}

func inferSkillName(files []*zip.File) string {
	for _, file := range files {
		name := filepath.Clean(file.Name)
		parts := strings.Split(name, string(filepath.Separator))
		if len(parts) > 1 && strings.EqualFold(parts[len(parts)-1], "SKILL.md") {
			return parts[0]
		}
	}
	return ""
}

// safeSkillName turns a zip filename or top-level zip folder into a single safe directory name.
func safeSkillName(name string) string {
	name = strings.TrimSpace(strings.TrimSuffix(filepath.Base(name), ".zip"))
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)
	return strings.Trim(name, "-._")
}

func skillDetail(codexHome, name, file string) (managetypes.SkillDetailResponse, error) {
	name = safeSkillName(name)
	if name == "" {
		return managetypes.SkillDetailResponse{}, errors.New("skill name is required")
	}
	root, err := skillsRoot(codexHome)
	if err != nil {
		return managetypes.SkillDetailResponse{}, err
	}
	path := filepath.Join(root, name)
	if rel, err := filepath.Rel(root, path); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return managetypes.SkillDetailResponse{}, errors.New("invalid skill name")
	}
	files, err := skillFileTree(path)
	if err != nil {
		return managetypes.SkillDetailResponse{}, err
	}
	currentFile := strings.TrimSpace(file)
	if currentFile == "" {
		currentFile = defaultSkillFile(path, files)
	}
	content, err := readSkillFile(path, currentFile)
	if err != nil {
		return managetypes.SkillDetailResponse{}, err
	}
	return managetypes.SkillDetailResponse{
		Skill:       managetypes.Skill{Name: name},
		Files:       files,
		CurrentFile: filepath.ToSlash(currentFile),
		Content:     content,
	}, nil
}

func skillFileTree(root string) ([]managetypes.SkillFile, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	result := make([]managetypes.SkillFile, 0, len(entries))
	for _, entry := range entries {
		item, err := skillFileNode(root, entry.Name(), entry)
		if err != nil {
			continue
		}
		result = append(result, item)
	}
	sortSkillFiles(result)
	return result, nil
}

func skillFileNode(root, rel string, entry os.DirEntry) (managetypes.SkillFile, error) {
	if entry.Type()&os.ModeSymlink != 0 {
		return managetypes.SkillFile{}, errors.New("skip symlink")
	}
	item := managetypes.SkillFile{Name: entry.Name(), Path: filepath.ToSlash(rel), Type: "file"}
	if !entry.IsDir() {
		return item, nil
	}
	item.Type = "directory"
	children, err := os.ReadDir(filepath.Join(root, rel))
	if err != nil {
		return item, err
	}
	item.Children = make([]managetypes.SkillFile, 0, len(children))
	for _, child := range children {
		childItem, err := skillFileNode(root, filepath.Join(rel, child.Name()), child)
		if err != nil {
			continue
		}
		item.Children = append(item.Children, childItem)
	}
	sortSkillFiles(item.Children)
	return item, nil
}

func sortSkillFiles(files []managetypes.SkillFile) {
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
}

func defaultSkillFile(root string, files []managetypes.SkillFile) string {
	if _, err := os.Stat(filepath.Join(root, "SKILL.md")); err == nil {
		return "SKILL.md"
	}
	var walk func([]managetypes.SkillFile) string
	walk = func(items []managetypes.SkillFile) string {
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

func resolveSkillFile(root, file string) (string, error) {
	file = strings.TrimSpace(filepath.Clean(file))
	if file == "" || file == "." || filepath.IsAbs(file) {
		return "", errors.New("file is required")
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, file)
	pathReal, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootReal, pathReal)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid file path")
	}
	return pathReal, nil
}

func readSkillFile(root, file string) (string, error) {
	path, err := resolveSkillFile(root, file)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("cannot read a directory")
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("only regular files can be read")
	}
	if info.Size() > 2<<20 {
		return "", errors.New("file is larger than 2 MiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func updateSkillFile(codexHome, name, file, content string) error {
	name = safeSkillName(name)
	if name == "" {
		return errors.New("skill name is required")
	}
	if len(content) > 2<<20 {
		return errors.New("file content is larger than 2 MiB")
	}
	root, err := skillsRoot(codexHome)
	if err != nil {
		return err
	}
	skillRoot := filepath.Join(root, name)
	if rel, err := filepath.Rel(root, skillRoot); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("invalid skill name")
	}
	info, err := os.Stat(skillRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("skill is not a directory")
	}
	path, err := resolveSkillFile(skillRoot, file)
	if err != nil {
		return err
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return errors.New("cannot edit a directory")
	}
	if !fileInfo.Mode().IsRegular() {
		return errors.New("only regular files can be edited")
	}
	return os.WriteFile(path, []byte(content), fileInfo.Mode().Perm())
}

func deleteSkill(codexHome, name string) error {
	requestedName := strings.TrimSpace(name)
	if requestedName == ".system" {
		return errors.New("system skills cannot be deleted")
	}
	name = safeSkillName(name)
	if name == "" {
		return errors.New("skill name is required")
	}
	root, err := skillsRoot(codexHome)
	if err != nil {
		return err
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("symlink skills cannot be deleted")
	}
	if !info.IsDir() {
		return errors.New("skill is not a directory")
	}
	pathReal, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootReal, pathReal)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("invalid skill path")
	}
	if rel == ".system" || strings.HasPrefix(rel, ".system"+string(filepath.Separator)) {
		return errors.New("system skills cannot be deleted")
	}
	return os.RemoveAll(pathReal)
}

var allowedConfigFiles = []string{"config.toml", "models.json", "auth.json"}

func listAgentConfigFiles() []managetypes.AgentConfigFile {
	result := make([]managetypes.AgentConfigFile, 0, len(allowedConfigFiles))
	for _, name := range allowedConfigFiles {
		result = append(result, managetypes.AgentConfigFile{Name: name})
	}
	return result
}

func codexHomeRoot(codexHome string) (string, error) {
	if strings.TrimSpace(codexHome) == "" {
		codexHome = "data/codex-home"
	}
	return filepath.Abs(codexHome)
}

func resolveAgentConfigFile(codexHome, name string) (string, string, error) {
	name = strings.TrimSpace(name)
	allowed := false
	for _, item := range allowedConfigFiles {
		if name == item {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", "", fmt.Errorf("unsupported config file %q", name)
	}
	root, err := codexHomeRoot(codexHome)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(root, name)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", errors.New("invalid config file path")
	}
	return name, path, nil
}

func readAgentConfigFile(codexHome, name string) (managetypes.ConfigFileResponse, error) {
	resolvedName, path, err := resolveAgentConfigFile(codexHome, name)
	if err != nil {
		return managetypes.ConfigFileResponse{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return managetypes.ConfigFileResponse{Name: resolvedName, Content: ""}, nil
	}
	if err != nil {
		return managetypes.ConfigFileResponse{}, err
	}
	if len(data) > 4<<20 {
		return managetypes.ConfigFileResponse{}, errors.New("config file is larger than 4 MiB")
	}
	return managetypes.ConfigFileResponse{Name: resolvedName, Content: string(data)}, nil
}

func writeAgentConfigFile(codexHome, name, content string) error {
	_, path, err := resolveAgentConfigFile(codexHome, name)
	if err != nil {
		return err
	}
	if len(content) > 4<<20 {
		return errors.New("config content is larger than 4 MiB")
	}
	if err := validateAgentConfigContent(name, content); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	perm := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		perm = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func validateAgentConfigContent(name, content string) error {
	switch name {
	case "config.toml":
		var value map[string]any
		if err := toml.Unmarshal([]byte(content), &value); err != nil {
			return fmt.Errorf("invalid TOML: %w", err)
		}
	case "models.json":
		if strings.TrimSpace(content) == "" {
			return nil
		}
		var value any
		if err := json.Unmarshal([]byte(content), &value); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	case "auth.json":
		if strings.TrimSpace(content) == "" {
			return nil
		}
		var value any
		if err := json.Unmarshal([]byte(content), &value); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file %q", name)
	}
	return nil
}

func runtimeStatus(status agentdomain.RuntimeStatus) managetypes.RuntimeStatus {
	lastRestartTime := ""
	if !status.LastRestartTime.IsZero() {
		lastRestartTime = status.LastRestartTime.Format(time.RFC3339)
	}
	return managetypes.RuntimeStatus{
		Running:         status.Running,
		Restarting:      status.Restarting,
		ActiveRunCount:  int64(status.ActiveRunCount),
		LastRestartTime: lastRestartTime,
	}
}
