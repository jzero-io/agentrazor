package agent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
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

func TestSplitPluginSkillsSkipsExistingAndDuplicateNames(t *testing.T) {
	bundled := []pluginSkillDirectory{
		{Name: "already-installed", Path: "/plugins/a/skills/already-installed"},
		{Name: "new-skill", Path: "/plugins/a/skills/new-skill"},
		{Name: "new-skill", Path: "/plugins/b/skills/new-skill"},
	}
	installed := []Skill{{
		Name: "manifest-name",
		Path: "/dist/data/skills/already-installed/SKILL.md",
	}}

	pending, skipped := splitPluginSkills(bundled, installed)
	wantPending := []pluginSkillDirectory{{Name: "new-skill", Path: "/plugins/a/skills/new-skill"}}
	wantSkipped := []string{"already-installed", "new-skill"}
	if !reflect.DeepEqual(pending, wantPending) {
		t.Fatalf("pending = %#v, want %#v", pending, wantPending)
	}
	if !reflect.DeepEqual(skipped, wantSkipped) {
		t.Fatalf("skipped = %#v, want %#v", skipped, wantSkipped)
	}
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
