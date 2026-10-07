# agent-toolchain

## About

A Go CLI for creating and running AI agents through Genkit. Agent settings are stored in YAML, and responses stream to the terminal as they are generated.

## Configuration

The root `agents-toolchain.yml` file specifies the agents directory:

```yaml
agents_dir: agents
```

Each agent has its own directory. For example, `agents/hello/agent.yml`:

```yaml
version: "1.0"
agent:
  name: hello
  description: My first agent
  model: openai/gpt-4o
  temperature: 0.2
  system_prompt: |
    You are a helpful assistant. Answer clearly and to the point.
  memory:
    type: local_file
    path: ./.agent_history.json
  tools: []
```

`name` must match the agent directory name. `model` uses the `provider/model` format; supported providers are `openai`, `anthropic`, and `ollama`. The `memory` section is optional, and its path is relative to the agent directory. Tool execution is not implemented yet, so keep `tools` empty.

## Commands

Requires Go 1.26.5 or later. Run the commands from the repository root.

PowerShell example:

```powershell
go run . init
go run . add hello
$env:OPENAI_API_KEY = "your-api-key"
go run . run hello --prompt "Hello!"
```

- `init` creates the `agents` directory and `agents-toolchain.yml`.
- `add <agent-name>` creates an agent directory and its `agent.yml`.
- `run <agent-name>` runs an agent. Use `--prompt` or `-p` to provide a request; otherwise, the agent follows the task in its system prompt.

Use `init --agents-dir <directory>` to set the agents directory. The `--workspace-dir <directory>` flag is available for all commands and sets the project directory instead of using the current one.
