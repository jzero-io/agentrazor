package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxRuntimeFileSize = 10 << 20
	maxConfigFileSize  = 4 << 20
)

type RuntimeSkill struct {
	Name string
}

type RuntimeSkillFile struct {
	Name     string
	Path     string
	Type     string
	Children []RuntimeSkillFile
}

type RuntimeSkillDetail struct {
	Skill       RuntimeSkill
	Files       []RuntimeSkillFile
	CurrentFile string
	Content     string
}

type RuntimeWorkspaceEntry struct {
	Name string
	Path string
	Type string
	Size int64
}

type RuntimeFile struct {
	ContentType string
	Data        []byte
}

type RuntimeRemoteError struct {
	Kind    string
	Message string
}

func (e *RuntimeRemoteError) Error() string {
	return e.Message
}

type runtimeDirectoryEntry struct {
	Name        string
	IsDirectory bool
	IsFile      bool
}

type runtimeFileMetadata struct {
	IsDirectory bool
	IsFile      bool
	IsSymlink   bool
	Size        int64
}

type CodexAppServerClient struct {
	threads        *ThreadService
	codexHome      string
	agentrazorHome string
}

func NewCodexAppServerClient(threads *ThreadService, codexHome, agentrazorHome string) *CodexAppServerClient {
	return &CodexAppServerClient{
		threads:        threads,
		codexHome:      cleanRuntimeRoot(codexHome, "data"),
		agentrazorHome: cleanRuntimeRoot(agentrazorHome, "data/agentrazor-home"),
	}
}

func cleanRuntimeRoot(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	if resolved, err := filepath.Abs(value); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(value)
}

func (c *CodexAppServerClient) request(ctx context.Context, method string, params any) (map[string]any, error) {
	if c == nil || c.threads == nil {
		return nil, errors.New("Codex app-server is not configured")
	}
	return c.threads.RuntimeRequest(ctx, method, params)
}

func (c *CodexAppServerClient) readFile(ctx context.Context, filePath string, limit int64) ([]byte, error) {
	result, err := c.request(ctx, "fs/readFile", map[string]any{"path": filePath})
	if err != nil {
		return nil, err
	}
	encoded, _ := result["dataBase64"].(string)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode Codex fs/readFile response: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("runtime file is larger than %d bytes", limit)
	}
	return data, nil
}

func (c *CodexAppServerClient) writeFile(ctx context.Context, filePath string, data []byte) error {
	_, err := c.request(ctx, "fs/writeFile", map[string]any{
		"path":       filePath,
		"dataBase64": base64.StdEncoding.EncodeToString(data),
	})
	return err
}

func (c *CodexAppServerClient) createDirectory(ctx context.Context, directory string) error {
	_, err := c.request(ctx, "fs/createDirectory", map[string]any{
		"path":      directory,
		"recursive": true,
	})
	return err
}

func (c *CodexAppServerClient) remove(ctx context.Context, target string, recursive, force bool) error {
	_, err := c.request(ctx, "fs/remove", map[string]any{
		"path":      target,
		"recursive": recursive,
		"force":     force,
	})
	return err
}

func (c *CodexAppServerClient) readDirectory(ctx context.Context, directory string) ([]runtimeDirectoryEntry, error) {
	result, err := c.request(ctx, "fs/readDirectory", map[string]any{"path": directory})
	if err != nil {
		return nil, err
	}
	values, _ := result["entries"].([]any)
	entries := make([]runtimeDirectoryEntry, 0, len(values))
	for _, value := range values {
		raw, ok := value.(map[string]any)
		if !ok {
			continue
		}
		entries = append(entries, runtimeDirectoryEntry{
			Name:        stringValue(raw["fileName"]),
			IsDirectory: boolValue(raw["isDirectory"]),
			IsFile:      boolValue(raw["isFile"]),
		})
	}
	return entries, nil
}

func (c *CodexAppServerClient) metadata(ctx context.Context, filePath string) (runtimeFileMetadata, error) {
	result, err := c.request(ctx, "fs/getMetadata", map[string]any{
		"path":           filePath,
		"followSymlinks": false,
	})
	if err != nil {
		return runtimeFileMetadata{}, err
	}
	return runtimeFileMetadata{
		IsDirectory: boolValue(result["isDirectory"]),
		IsFile:      boolValue(result["isFile"]),
		IsSymlink:   boolValue(result["isSymlink"]),
		Size:        runtimeInt64Value(result["size"]),
	}, nil
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func runtimeInt64Value(value any) int64 {
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

func runtimeChild(root, relative string) (string, error) {
	relative = filepath.Clean(strings.TrimSpace(relative))
	if relative == "" || relative == "." || filepath.IsAbs(relative) || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid runtime file path")
	}
	target := filepath.Clean(filepath.Join(root, relative))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid runtime file path")
	}
	return target, nil
}

func runtimePathWithin(root, candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", errors.New("runtime file path is required")
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("runtime file path is outside the allowed root")
	}
	return candidate, nil
}

func runtimeNotFound(err error) bool {
	var rpcErr *RPCError
	return errors.As(err, &rpcErr) && strings.Contains(strings.ToLower(rpcErr.Message), "no such file")
}

func (c *CodexAppServerClient) ListWorkspaceFiles(ctx context.Context, conversationID string) ([]RuntimeWorkspaceEntry, error) {
	if err := validateThreadID(conversationID); err != nil {
		return nil, err
	}
	root, err := runtimeChild(c.agentrazorHome, conversationID)
	if err != nil {
		return nil, err
	}
	entries := make([]RuntimeWorkspaceEntry, 0)
	if err := c.walkWorkspace(ctx, root, "", &entries); err != nil {
		if runtimeNotFound(err) {
			return entries, nil
		}
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func (c *CodexAppServerClient) walkWorkspace(ctx context.Context, root, relative string, result *[]RuntimeWorkspaceEntry) error {
	directory := root
	if relative != "" {
		directory = filepath.Join(root, relative)
	}
	entries, err := c.readDirectory(ctx, directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name == "" || strings.ContainsAny(entry.Name, "/\\") {
			continue
		}
		childRelative := filepath.Join(relative, entry.Name)
		childPath := filepath.Join(root, childRelative)
		switch {
		case entry.IsDirectory:
			*result = append(*result, RuntimeWorkspaceEntry{
				Name: entry.Name, Path: filepath.ToSlash(childRelative), Type: "directory",
			})
			if err := c.walkWorkspace(ctx, root, childRelative, result); err != nil {
				return err
			}
		case entry.IsFile:
			metadata, err := c.metadata(ctx, childPath)
			if err != nil || metadata.IsSymlink || !metadata.IsFile {
				continue
			}
			*result = append(*result, RuntimeWorkspaceEntry{
				Name: entry.Name, Path: filepath.ToSlash(childRelative), Type: "file", Size: metadata.Size,
			})
		}
	}
	return nil
}

func (c *CodexAppServerClient) ReadWorkspaceFile(ctx context.Context, conversationID, filePath string) (RuntimeFile, error) {
	if err := validateThreadID(conversationID); err != nil {
		return RuntimeFile{}, err
	}
	root, err := runtimeChild(c.agentrazorHome, conversationID)
	if err != nil {
		return RuntimeFile{}, err
	}
	target, err := runtimeChild(root, filePath)
	if err != nil {
		return RuntimeFile{}, err
	}
	metadata, err := c.metadata(ctx, target)
	if err != nil {
		return RuntimeFile{}, err
	}
	if metadata.IsSymlink || !metadata.IsFile {
		return RuntimeFile{}, errors.New("workspace path is not a regular file")
	}
	data, err := c.readFile(ctx, target, maxRuntimeFileSize)
	if err != nil {
		return RuntimeFile{}, err
	}
	return RuntimeFile{ContentType: http.DetectContentType(data), Data: data}, nil
}

func (c *CodexAppServerClient) ReadGeneratedImage(ctx context.Context, conversationID, savedPath string) (RuntimeFile, error) {
	if err := validateThreadID(conversationID); err != nil {
		return RuntimeFile{}, err
	}
	root := filepath.Join(c.codexHome, "generated_images", conversationID)
	target, err := runtimePathWithin(root, savedPath)
	if err != nil {
		return RuntimeFile{}, err
	}
	metadata, err := c.metadata(ctx, target)
	if err != nil {
		return RuntimeFile{}, err
	}
	if metadata.IsSymlink || !metadata.IsFile || metadata.Size > maxRuntimeFileSize {
		return RuntimeFile{}, errors.New("generated image is not a supported regular file")
	}
	data, err := c.readFile(ctx, target, maxRuntimeFileSize)
	if err != nil {
		return RuntimeFile{}, err
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return RuntimeFile{}, errors.New("generated image has an unsupported content type")
	}
	return RuntimeFile{ContentType: contentType, Data: data}, nil
}

func (c *CodexAppServerClient) ReadConfigFile(ctx context.Context, name string) (string, error) {
	target, err := c.configPath(name)
	if err != nil {
		return "", err
	}
	data, err := c.readFile(ctx, target, maxConfigFileSize)
	if runtimeNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *CodexAppServerClient) WriteConfigFile(ctx context.Context, name, content string) error {
	if len(content) > maxConfigFileSize {
		return errors.New("config content is larger than 4 MiB")
	}
	target, err := c.configPath(name)
	if err != nil {
		return err
	}
	return c.writeFile(ctx, target, []byte(content))
}

func (c *CodexAppServerClient) configPath(name string) (string, error) {
	switch strings.TrimSpace(name) {
	case "config.toml", "models.json":
		return filepath.Join(c.codexHome, strings.TrimSpace(name)), nil
	default:
		return "", fmt.Errorf("unsupported config file %q", name)
	}
}

func (c *CodexAppServerClient) Restart(context.Context) error {
	if c == nil || c.threads == nil {
		return errors.New("Codex app-server is not configured")
	}
	return c.threads.RestartRuntime()
}

func readArchiveBytes(reader io.Reader, size, limit int64) ([]byte, error) {
	if size <= 0 {
		return nil, &RuntimeRemoteError{Kind: "archive_empty", Message: "skill archive is empty"}
	}
	if size > limit {
		return nil, &RuntimeRemoteError{Kind: "archive_too_large", Message: "skill archive is too large"}
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, &RuntimeRemoteError{Kind: "archive_too_large", Message: "skill archive is too large"}
	}
	return data, nil
}
