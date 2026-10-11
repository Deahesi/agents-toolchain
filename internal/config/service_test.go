package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
)

type serviceUI struct{ uiMock }

func (*serviceUI) LogSpinner(...any) (domain.Spinner, error) { return nopSpinner{}, nil }

type nopSpinner struct{}

func (nopSpinner) Success(...any) {}
func (nopSpinner) Warning(...any) {}
func (nopSpinner) Fail(...any)    {}

func TestConfigServiceRoundTrip(t *testing.T) {
	workspace := t.TempDir()

	service := NewConfigService(&serviceUI{}, workspace)

	ctx := context.Background()

	if path, err := service.Init(ctx, "configs/agents"); err != nil || path != ProjectFile {
		t.Fatalf("Init = %q, %v", path, err)
	}

	if _, err := service.Init(ctx, "other"); !errors.Is(err, ErrInitialized) {
		t.Fatalf("second Init = %v; want ErrInitialized", err)
	}

	wantPath := filepath.Join("configs", "agents", "example", AgentFile)

	path, err := service.CreateAgent(ctx, "example")

	if err != nil || path != wantPath {
		t.Fatalf("CreateAgent = %q, %v; want %q", path, err, wantPath)
	}

	configuration, err := service.LoadAgent(ctx, "example")
	if err != nil {
		t.Fatal(err)
	}

	if configuration.Agent.Name != "example" || configuration.Path != filepath.Join(workspace, wantPath) {
		t.Fatalf("unexpected loaded configuration: %+v", configuration)
	}
	before, err := os.ReadFile(configuration.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateAgent(ctx, "example"); !errors.Is(err, ErrAgentExists) {
		t.Fatalf("second CreateAgent = %v; want ErrAgentExists", err)
	}
	after, err := os.ReadFile(configuration.Path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("existing agent config changed: %v", err)
	}
}

func TestConfigServiceRejectsEscapes(t *testing.T) {
	workspace := t.TempDir()
	service := NewConfigService(&serviceUI{}, workspace)
	ctx := context.Background()
	for _, dir := range []string{"../agents", workspace, "agents/../other"} {
		if _, err := service.Init(ctx, dir); err == nil {
			t.Errorf("Init accepted %q", dir)
		}
	}
	if _, err := service.Init(ctx, "agents"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../other", "..", "a/b", workspace} {
		if _, err := service.CreateAgent(ctx, name); err == nil {
			t.Errorf("CreateAgent accepted %q", name)
		}
		if _, err := service.LoadAgent(ctx, name); err == nil {
			t.Errorf("LoadAgent accepted %q", name)
		}
	}
}

func TestLoadAgentRejectsSiblingConfigSymlink(t *testing.T) {
	workspace := t.TempDir()
	service := NewConfigService(&serviceUI{}, workspace)
	ctx := context.Background()
	if _, err := service.Init(ctx, "agents"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateAgent(ctx, "example"); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(workspace, "agents", "other.yml")
	path := filepath.Join(workspace, "agents", "example", AgentFile)
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../other.yml", path); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if _, err := service.LoadAgent(ctx, "example"); err == nil {
		t.Fatal("loaded a config outside the selected agent directory")
	}
}

func TestConfigServiceRejectsExternalAgentsLink(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(workspace, "agents")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	service := NewConfigService(&serviceUI{}, workspace)
	ctx := context.Background()
	if _, err := service.Init(ctx, "agents"); err == nil {
		t.Fatal("Init followed an external agents link")
	}
	if err := os.WriteFile(filepath.Join(workspace, ProjectFile), []byte("agents_dir: agents\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateAgent(ctx, "example"); err == nil {
		t.Fatal("CreateAgent followed an external agents link")
	}
	if _, err := service.LoadAgent(ctx, "example"); err == nil {
		t.Fatal("LoadAgent followed an external agents link")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("outside directory changed: %v, %v", entries, err)
	}
}
