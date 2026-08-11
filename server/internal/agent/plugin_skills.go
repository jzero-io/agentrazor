package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const pluginSkillMarker = ".agentrazor-plugin-skill.json"

type pluginSkillManifest struct {
	Source string `json:"source"`
	Digest string `json:"digest"`
}

// syncPluginSkills installs skills shipped by server plugins into the
// persistent Codex home. User-managed skills are never overwritten.
func syncPluginSkills(pluginsRoot, codexHome string) error {
	if codexHome == "" {
		return nil
	}
	if pluginsRoot == "" {
		pluginsRoot = "plugins"
	}
	pluginsRoot, err := filepath.Abs(pluginsRoot)
	if err != nil {
		return fmt.Errorf("resolve plugins root: %w", err)
	}
	skillFiles, err := filepath.Glob(filepath.Join(pluginsRoot, "*", "skills", "*", "SKILL.md"))
	if err != nil {
		return fmt.Errorf("discover plugin skills: %w", err)
	}
	sort.Strings(skillFiles)
	for _, skillFile := range skillFiles {
		source := filepath.Dir(skillFile)
		destination := filepath.Join(codexHome, "skills", filepath.Base(source))
		if err := syncPluginSkill(pluginsRoot, source, destination); err != nil {
			return fmt.Errorf("sync plugin skill %q: %w", filepath.Base(source), err)
		}
	}
	return nil
}

func syncPluginSkill(pluginsRoot, source, destination string) error {
	digest, err := skillDirectoryDigest(source, false)
	if err != nil {
		return err
	}
	relativeSource, err := filepath.Rel(pluginsRoot, source)
	if err != nil {
		return fmt.Errorf("resolve skill source: %w", err)
	}
	manifest := pluginSkillManifest{Source: filepath.ToSlash(relativeSource), Digest: digest}

	if info, statErr := os.Stat(destination); statErr == nil {
		if !info.IsDir() {
			return errors.New("destination exists and is not a directory")
		}
		existing, err := readPluginSkillManifest(destination)
		if err != nil {
			return err
		}
		currentDigest, err := skillDirectoryDigest(destination, true)
		if err != nil {
			return err
		}
		if currentDigest != existing.Digest {
			return errors.New("managed skill was modified locally; refusing to overwrite it")
		}
		if existing == manifest {
			return nil
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect destination: %w", statErr)
	}

	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create skills directory: %w", err)
	}
	temporary, err := os.MkdirTemp(parent, ".plugin-skill-")
	if err != nil {
		return fmt.Errorf("create temporary skill: %w", err)
	}
	defer os.RemoveAll(temporary)
	if err := copySkillDirectory(source, temporary); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode skill manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(temporary, pluginSkillMarker), append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write skill manifest: %w", err)
	}

	backup := destination + ".previous"
	if _, err := os.Stat(destination); err == nil {
		_ = os.RemoveAll(backup)
		if err := os.Rename(destination, backup); err != nil {
			return fmt.Errorf("backup previous skill: %w", err)
		}
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Rename(backup, destination)
		return fmt.Errorf("publish plugin skill: %w", err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("remove previous skill backup: %w", err)
	}
	return nil
}

func readPluginSkillManifest(dir string) (pluginSkillManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, pluginSkillMarker))
	if errors.Is(err, os.ErrNotExist) {
		return pluginSkillManifest{}, errors.New("destination is not managed by AgentRazor; refusing to overwrite it")
	}
	if err != nil {
		return pluginSkillManifest{}, fmt.Errorf("read skill manifest: %w", err)
	}
	var manifest pluginSkillManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return pluginSkillManifest{}, fmt.Errorf("decode skill manifest: %w", err)
	}
	if manifest.Source == "" || manifest.Digest == "" {
		return pluginSkillManifest{}, errors.New("skill manifest is incomplete")
	}
	return manifest, nil
}

func skillDirectoryDigest(root string, ignoreMarker bool) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if ignoreMarker && relative == pluginSkillMarker {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed: %s", relative)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported entry: %s", relative)
		}
		_, _ = io.WriteString(hash, filepath.ToSlash(relative))
		_, _ = hash.Write([]byte{0})
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		return "", fmt.Errorf("hash skill directory: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copySkillDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if relative == pluginSkillMarker {
			return errors.New("source uses reserved manifest name")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed: %s", relative)
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported entry: %s", relative)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return outputCloseErr
	})
}
