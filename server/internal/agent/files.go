package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxFileSize       = 10 << 20
	maxConfigFileSize = 4 << 20
)

type WorkspaceEntry struct {
	Name     string
	Path     string
	Type     string
	Children []WorkspaceEntry
}

type File struct {
	ContentType string
	Data        []byte
}

type directoryEntry struct {
	Name        string
	IsDirectory bool
	IsFile      bool
}

type fileMetadata struct {
	IsDirectory bool
	IsFile      bool
	IsSymlink   bool
	Size        int64
}

func (s *Service) readFile(ctx context.Context, filePath string, limit int64) ([]byte, error) {
	result, err := s.call(ctx, "fs/readFile", map[string]any{"path": filePath})
	if err != nil {
		return nil, err
	}
	encoded, _ := result["dataBase64"].(string)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode Codex fs/readFile response: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file is larger than %d bytes", limit)
	}
	return data, nil
}

func (s *Service) writeFile(ctx context.Context, filePath string, data []byte) error {
	_, err := s.call(ctx, "fs/writeFile", map[string]any{
		"path":       filePath,
		"dataBase64": base64.StdEncoding.EncodeToString(data),
	})
	return err
}

func (s *Service) createDirectory(ctx context.Context, directory string) error {
	_, err := s.call(ctx, "fs/createDirectory", map[string]any{
		"path":      directory,
		"recursive": true,
	})
	return err
}

func (s *Service) remove(ctx context.Context, target string, recursive, force bool) error {
	_, err := s.call(ctx, "fs/remove", map[string]any{
		"path":      target,
		"recursive": recursive,
		"force":     force,
	})
	return err
}

func (s *Service) readDirectory(ctx context.Context, directory string) ([]directoryEntry, error) {
	result, err := s.call(ctx, "fs/readDirectory", map[string]any{"path": directory})
	if err != nil {
		return nil, err
	}
	values, _ := result["entries"].([]any)
	entries := make([]directoryEntry, 0, len(values))
	for _, value := range values {
		raw, ok := value.(map[string]any)
		if !ok {
			continue
		}
		entries = append(entries, directoryEntry{
			Name:        stringValue(raw["fileName"]),
			IsDirectory: boolValue(raw["isDirectory"]),
			IsFile:      boolValue(raw["isFile"]),
		})
	}
	return entries, nil
}

func (s *Service) metadata(ctx context.Context, filePath string) (fileMetadata, error) {
	result, err := s.call(ctx, "fs/getMetadata", map[string]any{
		"path":           filePath,
		"followSymlinks": false,
	})
	if err != nil {
		return fileMetadata{}, err
	}
	return fileMetadata{
		IsDirectory: boolValue(result["isDirectory"]),
		IsFile:      boolValue(result["isFile"]),
		IsSymlink:   boolValue(result["isSymlink"]),
		Size:        numberInt64(result["size"]),
	}, nil
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func numberInt64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func safeChild(root, relative string) (string, error) {
	relative = filepath.Clean(strings.TrimSpace(relative))
	if relative == "" || relative == "." || filepath.IsAbs(relative) || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid file path")
	}
	target := filepath.Clean(filepath.Join(root, relative))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid file path")
	}
	return target, nil
}

func safePathWithin(root, candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", errors.New("file path is required")
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("file path is outside the allowed root")
	}
	return candidate, nil
}

func isRemoteNotFound(err error) bool {
	var rpcErr *rpcCallError
	return errors.As(err, &rpcErr) && strings.Contains(strings.ToLower(rpcErr.Message), "no such file")
}

func (s *Service) ListWorkspaceFiles(ctx context.Context, conversationID string) ([]WorkspaceEntry, error) {
	if err := validateThreadID(conversationID); err != nil {
		return nil, err
	}
	root, err := safeChild(s.workspace, conversationID)
	if err != nil {
		return nil, err
	}
	files, err := s.workspaceFileTree(ctx, root, "")
	if err != nil {
		if isRemoteNotFound(err) {
			return []WorkspaceEntry{}, nil
		}
		return nil, err
	}
	return files, nil
}

func (s *Service) workspaceFileTree(ctx context.Context, root, relative string) ([]WorkspaceEntry, error) {
	directory := root
	if relative != "" {
		directory = filepath.Join(root, relative)
	}
	entries, err := s.readDirectory(ctx, directory)
	if err != nil {
		return nil, err
	}
	result := make([]WorkspaceEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "" || strings.ContainsAny(entry.Name, "/\\") || isInternalWorkspaceEntry(entry.Name) {
			continue
		}
		childRelative := filepath.Join(relative, entry.Name)
		childPath := filepath.Join(root, childRelative)
		item := WorkspaceEntry{
			Name: entry.Name, Path: filepath.ToSlash(childRelative), Type: "file", Children: []WorkspaceEntry{},
		}
		switch {
		case entry.IsDirectory:
			item.Type = "directory"
			item.Children, err = s.workspaceFileTree(ctx, root, childRelative)
			if err != nil {
				return nil, err
			}
		case entry.IsFile:
			metadata, err := s.metadata(ctx, childPath)
			if err != nil || metadata.IsSymlink || !metadata.IsFile {
				continue
			}
		default:
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type == "directory"
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

func isInternalWorkspaceEntry(name string) bool {
	return name == ".git" || name == ".agents" || name == ".codex"
}

func (s *Service) ReadWorkspaceFile(ctx context.Context, conversationID, filePath string) (File, error) {
	if err := validateThreadID(conversationID); err != nil {
		return File{}, err
	}
	root, err := safeChild(s.workspace, conversationID)
	if err != nil {
		return File{}, err
	}
	target, err := safeChild(root, filePath)
	if err != nil {
		return File{}, err
	}
	metadata, err := s.metadata(ctx, target)
	if err != nil {
		return File{}, err
	}
	if metadata.IsSymlink || !metadata.IsFile {
		return File{}, errors.New("workspace path is not a regular file")
	}
	data, err := s.readFile(ctx, target, maxFileSize)
	if err != nil {
		return File{}, err
	}
	return File{ContentType: http.DetectContentType(data), Data: data}, nil
}

func (s *Service) ReadGeneratedImage(ctx context.Context, conversationID, savedPath string) (File, error) {
	if err := validateThreadID(conversationID); err != nil {
		return File{}, err
	}
	root := filepath.Join(s.codexHome, "generated_images", conversationID)
	target, err := safePathWithin(root, savedPath)
	if err != nil {
		return File{}, err
	}
	metadata, err := s.metadata(ctx, target)
	if err != nil {
		return File{}, err
	}
	if metadata.IsSymlink || !metadata.IsFile || metadata.Size > maxFileSize {
		return File{}, errors.New("generated image is not a supported regular file")
	}
	data, err := s.readFile(ctx, target, maxFileSize)
	if err != nil {
		return File{}, err
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return File{}, errors.New("generated image has an unsupported content type")
	}
	return File{ContentType: contentType, Data: data}, nil
}

func (s *Service) readConfigFile(ctx context.Context, name string) (string, error) {
	target, err := s.configPath(name)
	if err != nil {
		return "", err
	}
	data, err := s.readFile(ctx, target, maxConfigFileSize)
	if isRemoteNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Service) readEffectiveConfig(ctx context.Context) (map[string]any, error) {
	result, err := s.call(ctx, "config/read", map[string]any{"includeLayers": false})
	if err != nil {
		return nil, err
	}
	config, ok := result["config"].(map[string]any)
	if !ok {
		return nil, errors.New("Codex config/read response did not contain config")
	}
	return config, nil
}

func (s *Service) writeConfigValues(ctx context.Context, values map[string]any) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	edits := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		edits = append(edits, map[string]any{
			"keyPath":       key,
			"value":         values[key],
			"mergeStrategy": "upsert",
		})
	}
	_, err := s.call(ctx, "config/batchWrite", map[string]any{
		"edits":            edits,
		"reloadUserConfig": true,
	})
	return err
}

func (s *Service) configPath(name string) (string, error) {
	switch strings.TrimSpace(name) {
	case "config.toml", "models.json":
		return filepath.Join(s.codexHome, strings.TrimSpace(name)), nil
	default:
		return "", fmt.Errorf("unsupported config file %q", name)
	}
}
