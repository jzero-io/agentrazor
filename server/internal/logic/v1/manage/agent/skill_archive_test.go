package agent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type memoryMultipartFile struct {
	*bytes.Reader
}

func (memoryMultipartFile) Close() error {
	return nil
}

type archiveTestEntry struct {
	name    string
	content string
}

func TestInstallSkillArchiveSupportsZipAndTarGz(t *testing.T) {
	tests := []struct {
		name        string
		archiveName string
		data        []byte
		skillName   string
		files       map[string]string
	}{
		{
			name:        "zip",
			archiveName: "zip-skill.zip",
			data: buildSkillZip(t, []archiveTestEntry{
				{name: "zip-skill/SKILL.md", content: "# ZIP skill"},
				{name: "zip-skill/references/guide.md", content: "zip guide"},
			}),
			skillName: "zip-skill",
			files: map[string]string{
				"SKILL.md":            "# ZIP skill",
				"references/guide.md": "zip guide",
			},
		},
		{
			name:        "tar gz",
			archiveName: "tar-skill.tar.gz",
			data: buildSkillTarGz(t, []archiveTestEntry{
				{name: "tar-skill/SKILL.md", content: "# TAR.GZ skill"},
				{name: "tar-skill/scripts/run.sh", content: "#!/bin/sh\n"},
			}),
			skillName: "tar-skill",
			files: map[string]string{
				"SKILL.md":       "# TAR.GZ skill",
				"scripts/run.sh": "#!/bin/sh\n",
			},
		},
		{
			name:        "root tar gz",
			archiveName: "root-skill.TAR.GZ",
			data: buildSkillTarGz(t, []archiveTestEntry{
				{name: "SKILL.md", content: "# Root skill"},
			}),
			skillName: "root-skill",
			files: map[string]string{
				"SKILL.md": "# Root skill",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codexHome := t.TempDir()
			file := memoryMultipartFile{Reader: bytes.NewReader(tt.data)}
			result, err := installSkillArchive(codexHome, "", tt.archiveName, file, int64(len(tt.data)))
			if err != nil {
				t.Fatalf("installSkillArchive() error = %v", err)
			}
			if result.Name != tt.skillName {
				t.Fatalf("installSkillArchive() name = %q, want %q", result.Name, tt.skillName)
			}
			for name, want := range tt.files {
				data, readErr := os.ReadFile(filepath.Join(codexHome, "skills", tt.skillName, filepath.FromSlash(name)))
				if readErr != nil {
					t.Fatalf("read installed file %q: %v", name, readErr)
				}
				if got := string(data); got != want {
					t.Fatalf("installed file %q = %q, want %q", name, got, want)
				}
			}
		})
	}
}

func TestInstallSkillArchiveRejectsUnsafeTarPath(t *testing.T) {
	data := buildSkillTarGz(t, []archiveTestEntry{
		{name: "../outside.txt", content: "unsafe"},
		{name: "unsafe-skill/SKILL.md", content: "# Unsafe"},
	})
	codexHome := t.TempDir()
	file := memoryMultipartFile{Reader: bytes.NewReader(data)}
	if _, err := installSkillArchive(codexHome, "", "unsafe-skill.tar.gz", file, int64(len(data))); err == nil {
		t.Fatal("installSkillArchive() error = nil, want unsafe path error")
	}
	if _, err := os.Stat(filepath.Join(codexHome, "skills", "outside.txt")); !os.IsNotExist(err) {
		t.Fatalf("unsafe archive wrote outside target: %v", err)
	}
}

func TestInstallSkillArchiveRejectsUnsupportedFormat(t *testing.T) {
	data := []byte("not an archive")
	file := memoryMultipartFile{Reader: bytes.NewReader(data)}
	if _, err := installSkillArchive(t.TempDir(), "", "skill.rar", file, int64(len(data))); err == nil {
		t.Fatal("installSkillArchive() error = nil, want unsupported format error")
	}
}

func TestInstallSkillArchiveRejectsMissingManifestWithoutInstalling(t *testing.T) {
	tests := []struct {
		name        string
		archiveName string
		data        []byte
	}{
		{
			name:        "zip",
			archiveName: "invalid-skill.zip",
			data: buildSkillZip(t, []archiveTestEntry{
				{name: "invalid-skill/readme.md", content: "not a skill"},
			}),
		},
		{
			name:        "tar gz",
			archiveName: "invalid-skill.tar.gz",
			data: buildSkillTarGz(t, []archiveTestEntry{
				{name: "invalid-skill/readme.md", content: "not a skill"},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codexHome := t.TempDir()
			file := memoryMultipartFile{Reader: bytes.NewReader(tt.data)}
			if _, err := installSkillArchive(codexHome, "", tt.archiveName, file, int64(len(tt.data))); !errors.Is(err, errSkillManifestMissing) {
				t.Fatalf("installSkillArchive() error = %v, want %v", err, errSkillManifestMissing)
			}
			if _, err := os.Stat(filepath.Join(codexHome, "skills", "invalid-skill")); !os.IsNotExist(err) {
				t.Fatalf("invalid skill directory exists after rejected upload: %v", err)
			}
			entries, err := os.ReadDir(filepath.Join(codexHome, "skills"))
			if err != nil {
				t.Fatalf("read skills directory: %v", err)
			}
			if len(entries) != 0 {
				t.Fatalf("temporary upload directories were not cleaned: %v", entries)
			}
		})
	}
}

func TestInstallSkillArchivePreservesExistingSkillWhenReplacementIsInvalid(t *testing.T) {
	codexHome := t.TempDir()
	existingDir := filepath.Join(codexHome, "skills", "existing-skill")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(existingDir, "SKILL.md")
	if err := os.WriteFile(manifest, []byte("# Existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	data := buildSkillZip(t, []archiveTestEntry{
		{name: "existing-skill/readme.md", content: "invalid replacement"},
	})
	file := memoryMultipartFile{Reader: bytes.NewReader(data)}
	if _, err := installSkillArchive(codexHome, "", "existing-skill.zip", file, int64(len(data))); !errors.Is(err, errSkillManifestMissing) {
		t.Fatalf("installSkillArchive() error = %v, want %v", err, errSkillManifestMissing)
	}
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("read existing manifest: %v", err)
	}
	if string(content) != "# Existing" {
		t.Fatalf("existing manifest = %q, want preserved content", content)
	}
}

func buildSkillZip(t *testing.T, entries []archiveTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		dst, err := writer.Create(entry.name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := dst.Write([]byte(entry.content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buffer.Bytes()
}

func buildSkillTarGz(t *testing.T, entries []archiveTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.name,
			Mode: 0o644,
			Size: int64(len(entry.content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
			t.Fatalf("write tar entry: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buffer.Bytes()
}
