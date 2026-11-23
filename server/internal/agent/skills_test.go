package agent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestManagedSkillsFromListResponse(t *testing.T) {
	managedRoot := filepath.Join(string(filepath.Separator), "dist", "data", "skills")
	response := map[string]any{
		"data": []any{
			map[string]any{
				"cwd": "/dist/data/workspace",
				"skills": []any{
					map[string]any{
						"name":        "zeta",
						"description": "Zeta skill",
						"enabled":     true,
						"path":        filepath.Join(managedRoot, "zeta", "SKILL.md"),
						"scope":       "user",
					},
					map[string]any{
						"name":        "alpha-manifest-name",
						"description": "Alpha skill",
						"enabled":     false,
						"path":        filepath.Join(managedRoot, "Alpha", "SKILL.md"),
						"scope":       "user",
					},
					map[string]any{
						"name":    "system-skill",
						"enabled": true,
						"path":    filepath.Join(managedRoot, ".system", "system-skill", "SKILL.md"),
						"scope":   "system",
					},
					map[string]any{
						"name":    "repo-skill",
						"enabled": true,
						"path":    filepath.Join(string(filepath.Separator), "repo", ".agents", "skills", "repo-skill", "SKILL.md"),
						"scope":   "repo",
					},
				},
			},
		},
	}

	got, err := managedSkillsFromListResponse(response, managedRoot)
	if err != nil {
		t.Fatalf("managedSkillsFromListResponse() error = %v", err)
	}
	want := []Skill{
		{
			Name:        "alpha-manifest-name",
			Description: "Alpha skill",
			Enabled:     false,
			Path:        filepath.Join(managedRoot, "Alpha", "SKILL.md"),
			Scope:       "user",
		},
		{
			Name:     "system-skill",
			Enabled:  true,
			Path:     filepath.Join(managedRoot, ".system", "system-skill", "SKILL.md"),
			Scope:    "system",
			ReadOnly: true,
		},
		{
			Name:        "zeta",
			Description: "Zeta skill",
			Enabled:     true,
			Path:        filepath.Join(managedRoot, "zeta", "SKILL.md"),
			Scope:       "user",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("managedSkillsFromListResponse() = %#v, want %#v", got, want)
	}
}

func TestManagedSkillsFromListResponseRequiresData(t *testing.T) {
	if _, err := managedSkillsFromListResponse(map[string]any{}, "/dist/data/skills"); err == nil {
		t.Fatal("managedSkillsFromListResponse() error = nil, want non-nil")
	}
}

func TestIsManagedSkillRejectsUnsafeMetadata(t *testing.T) {
	managedRoot := filepath.Join(string(filepath.Separator), "dist", "data", "skills")
	tests := []Skill{
		{Name: "../escape", Path: filepath.Join(managedRoot, "escape", "SKILL.md")},
		{Name: "nested", Path: filepath.Join(managedRoot, "group", "nested", "SKILL.md")},
		{Name: "wrong-file", Path: filepath.Join(managedRoot, "wrong-file", "README.md")},
		{Name: "relative", Path: filepath.Join("relative", "SKILL.md")},
	}
	for _, skill := range tests {
		if isManagedSkill(skill, managedRoot) {
			t.Errorf("isManagedSkill(%#v) = true, want false", skill)
		}
	}
}

func TestManagedSkillReadOnly(t *testing.T) {
	managedRoot := filepath.Join(string(filepath.Separator), "dist", "data", "skills")
	tests := []struct {
		name     string
		path     string
		readOnly bool
		managed  bool
	}{
		{name: "uploaded", path: filepath.Join(managedRoot, "uploaded", "SKILL.md"), managed: true},
		{name: "built-in", path: filepath.Join(managedRoot, ".system", "built-in", "SKILL.md"), readOnly: true, managed: true},
		{name: "nested", path: filepath.Join(managedRoot, ".system", "group", "nested", "SKILL.md")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			readOnly, managed := managedSkillReadOnly(Skill{Name: test.name, Path: test.path}, managedRoot)
			if readOnly != test.readOnly || managed != test.managed {
				t.Fatalf("managedSkillReadOnly() = (%v, %v), want (%v, %v)", readOnly, managed, test.readOnly, test.managed)
			}
		})
	}
}

func TestManagedSkillsFromListResponsePrefersSystemSkill(t *testing.T) {
	managedRoot := filepath.Join(string(filepath.Separator), "dist", "data", "skills")
	response := map[string]any{
		"data": []any{
			map[string]any{
				"skills": []any{
					map[string]any{
						"name": "shared",
						"path": filepath.Join(managedRoot, ".system", "shared", "SKILL.md"),
					},
					map[string]any{
						"name": "shared",
						"path": filepath.Join(managedRoot, "shared", "SKILL.md"),
					},
				},
			},
		},
	}

	got, err := managedSkillsFromListResponse(response, managedRoot)
	if err != nil {
		t.Fatalf("managedSkillsFromListResponse() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("managedSkillsFromListResponse() returned %d skills, want 1", len(got))
	}
	if !got[0].ReadOnly {
		t.Fatal("managedSkillsFromListResponse() selected editable skill, want system skill")
	}
	wantPath := filepath.Join(managedRoot, ".system", "shared", "SKILL.md")
	if got[0].Path != wantPath {
		t.Fatalf("managedSkillsFromListResponse() path = %q, want %q", got[0].Path, wantPath)
	}
}

func TestReadSkillZipPreservesOnlyScriptExecuteBits(t *testing.T) {
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	addZipFile(t, writer, "demo/SKILL.md", 0o755, "---\nname: demo\ndescription: Demo\n---\n")
	addZipFile(t, writer, "demo/scripts/demo-cli", 0o755, "binary")
	addZipFile(t, writer, "demo/references/example.md", 0o755, "reference")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := readSkillZip(data.Bytes(), "demo")
	if err != nil {
		t.Fatalf("readSkillZip() error = %v", err)
	}
	assertExecutableEntries(t, entries, map[string]bool{
		"SKILL.md":              false,
		"scripts/demo-cli":      true,
		"references/example.md": false,
	})
}

func TestReadSkillTarGzPreservesOnlyScriptExecuteBits(t *testing.T) {
	var data bytes.Buffer
	gzipWriter := gzip.NewWriter(&data)
	writer := tar.NewWriter(gzipWriter)
	addTarFile(t, writer, "demo/SKILL.md", 0o755, "---\nname: demo\ndescription: Demo\n---\n")
	addTarFile(t, writer, "demo/scripts/demo-cli", 0o755, "binary")
	addTarFile(t, writer, "demo/scripts/not-executable", 0o644, "data")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := readSkillTarGz(data.Bytes(), "demo")
	if err != nil {
		t.Fatalf("readSkillTarGz() error = %v", err)
	}
	assertExecutableEntries(t, entries, map[string]bool{
		"SKILL.md":               false,
		"scripts/demo-cli":       true,
		"scripts/not-executable": false,
	})
}

func TestPluginSkillDirectoriesAndReadSkillDirectory(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "example-plugin", "skills", "example-skill")
	if err := os.MkdirAll(filepath.Join(skillRoot, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("---\nname: example-skill\ndescription: Demo\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "scripts", "example_cli"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	directories, err := pluginSkillDirectories(root)
	if err != nil {
		t.Fatalf("pluginSkillDirectories() error = %v", err)
	}
	wantDirectories := []pluginSkillDirectory{{Name: "example-skill", Path: skillRoot}}
	if !reflect.DeepEqual(directories, wantDirectories) {
		t.Fatalf("pluginSkillDirectories() = %#v, want %#v", directories, wantDirectories)
	}

	entries, err := readSkillDirectory(skillRoot)
	if err != nil {
		t.Fatalf("readSkillDirectory() error = %v", err)
	}
	assertExecutableEntries(t, entries, map[string]bool{
		"SKILL.md":            false,
		"scripts/example_cli": true,
	})
}

func TestSplitPluginSkillsSkipsOnlyDuplicateNames(t *testing.T) {
	bundled := []pluginSkillDirectory{
		{Name: "already-installed", Path: "/plugins/a/skills/already-installed"},
		{Name: "new-skill", Path: "/plugins/a/skills/new-skill"},
		{Name: "new-skill", Path: "/plugins/b/skills/new-skill"},
	}
	pending, skipped := splitPluginSkills(bundled)
	wantPending := []pluginSkillDirectory{
		{Name: "already-installed", Path: "/plugins/a/skills/already-installed"},
		{Name: "new-skill", Path: "/plugins/a/skills/new-skill"},
	}
	wantSkipped := []string{"new-skill"}
	if !reflect.DeepEqual(pending, wantPending) {
		t.Fatalf("pending = %#v, want %#v", pending, wantPending)
	}
	if !reflect.DeepEqual(skipped, wantSkipped) {
		t.Fatalf("skipped = %#v, want %#v", skipped, wantSkipped)
	}
}

func TestSyncPluginSkillsPreservesUserEdits(t *testing.T) {
	home := t.TempDir()
	plugins := t.TempDir()
	bundle := filepath.Join(plugins, "plugin", "skills", "example")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "SKILL.md"), []byte("---\nname: example\ndescription: Default\n---\nDefault content"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := skillSyncTestService(t, home, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := s.SyncPluginSkills(ctx, plugins)
	if err != nil || !reflect.DeepEqual(result.Installed, []string{"example"}) {
		t.Fatalf("first sync = %#v, %v", result, err)
	}
	manifest := filepath.Join(home, "skills", "example", "SKILL.md")
	custom := []byte("User-edited manifest, deliberately without valid frontmatter")
	if err := os.WriteFile(manifest, custom, 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(manifest)
	// A newer bundle must not reset the user's existing directory.
	if err := os.WriteFile(filepath.Join(bundle, "new-reference.md"), []byte("new default"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err = s.SyncPluginSkills(ctx, plugins)
	if err != nil || len(result.Installed) != 0 || !reflect.DeepEqual(result.Skipped, []string{"example"}) {
		t.Fatalf("repeat sync = %#v, %v", result, err)
	}
	got, _ := os.ReadFile(manifest)
	after, _ := os.Stat(manifest)
	if !bytes.Equal(got, custom) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("startup synchronization rewrote the user-edited skill")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(manifest), "new-reference.md")); !os.IsNotExist(err) {
		t.Fatal("startup synchronization injected files into an existing skill")
	}
}

func TestSyncPluginSkillsStopsOnInspectionFailure(t *testing.T) {
	plugins := t.TempDir()
	if err := os.MkdirAll(filepath.Join(plugins, "plugin", "skills", "example"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := skillSyncTestService(t, t.TempDir(), true)
	if _, err := s.SyncPluginSkills(context.Background(), plugins); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("sync error = %v, want inspection failure", err)
	}
}

// Exercise startup synchronization through the real JSON-RPC client with a
// filesystem-backed peer, so preservation is checked on actual file contents.
func skillSyncTestService(t *testing.T, home string, denyMetadata bool) *Service {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var req struct {
				ID     int64          `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			result := map[string]any{}
			path, _ := req.Params["path"].(string)
			var opErr error
			switch req.Method {
			case "fs/getMetadata":
				if denyMetadata {
					opErr = fmt.Errorf("permission denied")
				} else if st, err := os.Lstat(path); err != nil {
					opErr = err
				} else {
					result["isDirectory"] = st.IsDir()
				}
			case "fs/createDirectory":
				opErr = os.MkdirAll(path, 0o755)
			case "fs/remove":
				opErr = os.RemoveAll(path)
			case "fs/writeFile":
				data, err := base64.StdEncoding.DecodeString(req.Params["dataBase64"].(string))
				if err == nil {
					err = os.WriteFile(path, data, 0o644)
				}
				opErr = err
			case "skills/list":
				result["data"] = []any{map[string]any{"skills": []any{map[string]any{
					"name": "example", "path": filepath.Join(home, "skills", "example", "SKILL.md"), "enabled": true, "scope": "user",
				}}}}
			default:
				opErr = fmt.Errorf("unexpected RPC %s", req.Method)
			}
			response := map[string]any{"id": req.ID, "result": result}
			if opErr != nil {
				delete(response, "result")
				response["error"] = map[string]any{"code": -32000, "message": opErr.Error()}
			}
			if err := conn.WriteJSON(response); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	rpc := &appServer{connection: conn, waitDone: make(chan struct{}), pending: make(map[int64]chan rpcEnvelope)}
	go rpc.readLoop()
	t.Cleanup(func() { _ = rpc.close() })
	return &Service{server: rpc, codexHome: home}
}

func addZipFile(t *testing.T, writer *zip.Writer, name string, mode int64, content string) {
	t.Helper()
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(os.FileMode(mode))
	file, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
}

func addTarFile(t *testing.T, writer *tar.Writer, name string, mode int64, content string) {
	t.Helper()
	header := &tar.Header{Name: name, Mode: mode, Size: int64(len(content)), Typeflag: tar.TypeReg}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
}

func assertExecutableEntries(t *testing.T, entries []archiveEntry, want map[string]bool) {
	t.Helper()
	got := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !entry.Directory {
			got[filepath.ToSlash(entry.Path)] = entry.Executable
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("executable entries = %#v, want %#v", got, want)
	}
}
