package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/firebase/genkit/go/ai"
)

type MemoryStore struct {
	config *domain.Memory
	root   *files.Root
}

func OpenMemoryStore(config *domain.Memory, agentDir string) (*MemoryStore, error) {
	if config == nil {
		return nil, fmt.Errorf("memory store not configured")
	}

	root, err := files.OpenRoot(agentDir)
	if err != nil {
		return nil, err
	}

	memoryDir := filepath.Dir(config.Path)
	memoryFile := filepath.Base(config.Path)
	if err = root.MkdirAll(memoryDir, 0o755); err != nil {
		return nil, err
	}

	root.WriteFile(
		filepath.Join(memoryDir, ".gitignore"),
		[]byte(memoryFile),
		0o755,
	)

	return &MemoryStore{
		config: config,
		root:   root,
	}, nil
}

func (s *MemoryStore) Load(ctx context.Context) ([]*ai.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b, err := s.root.ReadFile(s.config.Path)
	if err != nil {
		return nil, nil
	}

	var messages []*ai.Message

	err = json.Unmarshal(b, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *MemoryStore) Save(ctx context.Context, messages []*ai.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	err = s.root.WriteFile(s.config.Path, b, 0o755)
	if err != nil {
		return err
	}

	return nil
}

func (s *MemoryStore) Close() error {
	err := s.root.Close()

	return err
}
