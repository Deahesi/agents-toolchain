// Package atctools registers agents-toolchain's built-in tools with Genkit.
package atctools

import (
	"fmt"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/Deahesi/agents-toolchain/internal/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Built-in tool names. Tools are registered only when explicitly selected.
const (
	ReadFile       = "read_file"
	WriteFile      = "write_file"
	EditFile       = "edit_file"
	ListFiles      = "list_files"
	SearchFiles    = "search_files"
	ExecuteCommand = "execute_command"
	GetEnvironment = "get_environment"
	GetDateTime    = "get_datetime"
)

// Config selects the tools to register.
type Config struct {
	// WorkDir must be an existing directory when filesystem tools are selected.
	// Their paths are relative to this directory and cannot escape it via '..'
	// or symlinks. This is not a sandbox for shell commands: ExecuteCommand
	// runs in the process working directory with the process permissions.
	WorkDir string
	Tools   []string
}

// Registration owns the directory handle used by its tools.
// Keep it open until all tool calls have finished.
type Registration struct {
	Tools []ai.ToolRef
	root  *files.Root
}

// Register defines the selected tools in g. Pass the returned Tools to
// ai.WithTools and close the registration after all generations finish.
// Names must be unique within this call and must not already be registered in g.
func Register(g *genkit.Genkit, config Config) (*Registration, error) {
	if g == nil {
		return nil, fmt.Errorf("register tools: Genkit instance is nil")
	}

	configs := make([]domain.ToolConfig, 0, len(config.Tools))
	seen := make(map[string]bool, len(config.Tools))
	needsRoot := false
	for _, name := range config.Tools {
		if seen[name] {
			return nil, fmt.Errorf("register tools: duplicate tool %q", name)
		}
		seen[name] = true
		switch name {
		case ReadFile, WriteFile, EditFile, ListFiles, SearchFiles:
			needsRoot = true
		case ExecuteCommand, GetEnvironment, GetDateTime:
		default:
			return nil, fmt.Errorf("register tools: unknown built-in tool %q", name)
		}
		configs = append(configs, domain.ToolConfig{
			Type: domain.ToolTypeBuiltin,
			Name: name,
		})
	}

	r := &Registration{}
	if needsRoot {
		root, err := files.OpenRoot(config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("register tools: open work directory: %w", err)
		}
		r.root = root
	}

	registration := tools.NewToolRegistration(g, configs, r.root)
	if err := registration.Register(); err != nil {
		r.Close()
		return nil, fmt.Errorf("register tools: %w", err)
	}
	r.Tools = registration.Tools
	return r, nil
}

// Close releases the directory handle. It does not unregister tools from Genkit;
// do not invoke filesystem tools after closing their registration.
func (r *Registration) Close() error {
	if r.root == nil {
		return nil
	}
	return r.root.Close()
}
