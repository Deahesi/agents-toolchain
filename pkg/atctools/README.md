# atctools

Register the existing agents-toolchain built-in tools directly in your Genkit application. No runner, CLI UI, or internal package imports are required.

```go
import (
    "context"

    "github.com/Deahesi/agents-toolchain/pkg/atctools"
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
)

func generate(ctx context.Context, g *genkit.Genkit) (*ai.ModelResponse, error) {
    registration, err := atctools.Register(g, atctools.Config{
        WorkDir: "./workspace", // must already exist
        Tools: []string{
            atctools.ReadFile,
            atctools.ListFiles,
            atctools.SearchFiles,
        },
    })
    if err != nil {
        return nil, err
    }
    defer registration.Close()

    return genkit.Generate(ctx, g,
        ai.WithModelName("your-provider/your-model"),
        ai.WithPrompt("Describe the files in the workspace"),
        ai.WithTools(registration.Tools...),
    )
}
```

Configure your provider in `g` as usual. Register tools once per Genkit instance; do not register the same names twice. Keep the registration open until all generations and tool calls using it have finished. `Close` releases the filesystem handle but does not unregister tools.

Available tools:

- Filesystem: `ReadFile`, `WriteFile`, `EditFile`, `ListFiles`, `SearchFiles`.
- System: `ExecuteCommand`, `GetEnvironment`, `GetDateTime`.

Only explicitly selected tools are enabled. An empty list registers nothing. `WorkDir` is required only for filesystem tools. Unknown or duplicate names are rejected before registration.

Filesystem paths are confined to `WorkDir` using the existing `os.Root` implementation, including protection against traversal and symlink escapes. This is not a process sandbox. In particular, `ExecuteCommand` runs shell commands in the **process working directory**, with the process permissions, not inside the filesystem boundary. Enable it only for trusted workloads. The package does not add an approval mechanism for tool calls.
