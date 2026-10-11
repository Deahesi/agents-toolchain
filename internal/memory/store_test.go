package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
)

func TestMemoryStore(t *testing.T) {
	open := func(t *testing.T) *MemoryStore {
		t.Helper()
		store, err := OpenMemoryStore(&domain.Memory{
			Type: "local_file",
			Path: "state/history.json",
		}, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := store.Close(); err != nil {
				t.Error(err)
			}
		})
		return store
	}
	ctx := context.Background()

	t.Run("missing history is empty", func(t *testing.T) {
		store := open(t)
		messages, err := store.Load(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 0 {
			t.Fatalf("got %d messages, want empty history", len(messages))
		}
	})

	t.Run("save load and overwrite", func(t *testing.T) {
		store := open(t)
		for _, want := range [][]*ai.Message{
			{ai.NewUserTextMessage("Привет"), ai.NewModelTextMessage("Привет!")},
			{ai.NewUserTextMessage("Новая история")},
		} {
			if err := store.Save(ctx, want); err != nil {
				t.Fatal(err)
			}
			got, err := store.Load(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(want) {
				t.Fatalf("got %d messages, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i].Role != want[i].Role || got[i].Text() != want[i].Text() {
					t.Fatalf("message %d differs: got %+v, want %+v", i, got[i], want[i])
				}
			}
		}
	})

	t.Run("invalid JSON returns an error", func(t *testing.T) {
		store := open(t)
		if err := store.root.WriteFile(store.config.Path, []byte("{broken"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Load(ctx); err == nil {
			t.Fatal("expected an error for invalid JSON")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		store := open(t)
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := store.Load(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Load error = %v, want context.Canceled", err)
		}
		if err := store.Save(ctx, []*ai.Message{ai.NewUserTextMessage("hello")}); !errors.Is(err, context.Canceled) {
			t.Fatalf("Save error = %v, want context.Canceled", err)
		}
	})
}
