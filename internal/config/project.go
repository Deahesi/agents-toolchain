package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
)

func (s *ConfigService) Init(ctx context.Context, agentsDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	spinner, err := s.ui.LogSpinner("Checking project configuration")
	if err != nil {
		return "", err
	}
	content, err := encode(&domain.ProjectConfig{
		AgentsDir: agentsDir,
	})
	if err != nil {
		spinner.Fail()
		return "", err
	}
	spinner.Success()

	root, err := files.OpenRoot(s.workspaceDir)
	if err != nil {
		return "", fmt.Errorf("open project: %w", err)
	}
	defer root.Close()

	if _, err := root.Stat(ProjectFile); err == nil {
		return "", ErrInitialized
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("check project config: %w", err)
	}

	spinner, err = s.ui.LogSpinner("Creating agents directory: ", agentsDir)
	if err != nil {
		return "", err
	}
	if err := root.MkdirAll(agentsDir, 0o755); err != nil {
		spinner.Fail()
		return "", fmt.Errorf("create agents directory: %w", err)
	}
	spinner.Success()

	spinner, err = s.ui.LogSpinner("Writing project config: ", ProjectFile)
	if err != nil {
		return "", err
	}
	if err := root.WriteNewFile(ProjectFile, content, 0o644); err != nil {
		spinner.Fail()

		if errors.Is(err, fs.ErrExist) {
			return "", ErrInitialized
		}
		return "", fmt.Errorf("create project config: %w", err)
	}
	spinner.Success()

	return ProjectFile, nil
}
