package agent

import (
	"context"
	"fmt"
	"path/filepath"
)

const maxDefaultSystemPromptSize = 256 << 10

func (s *Service) DefaultSystemPrompt(ctx context.Context) (string, error) {
	content, err := s.readFile(ctx, filepath.Join(s.codexHome, "AGENTS.md"), maxDefaultSystemPromptSize)
	if err != nil {
		return "", fmt.Errorf("read default system prompt: %w", err)
	}
	return string(content), nil
}

func (s *Service) SaveDefaultSystemPrompt(ctx context.Context, content string) error {
	if len([]byte(content)) > maxDefaultSystemPromptSize {
		return fmt.Errorf("default system prompt is larger than %d bytes", maxDefaultSystemPromptSize)
	}
	if err := s.writeFile(ctx, filepath.Join(s.codexHome, "AGENTS.md"), []byte(content)); err != nil {
		return fmt.Errorf("write default system prompt: %w", err)
	}
	return nil
}
