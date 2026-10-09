# Agents Toolchain (atc)

> **Status: In development — not stable yet.**
> You can create and run agents, but bugs are still possible. Commands and config formats may change between releases.

A CLI for creating and running AI agents. Describe an agent's task, model, and tools in YAML, then run it from your terminal. Use agents to work on documentation, write tests, or review a project.

Supports OpenAI, Anthropic, OpenRouter, and Ollama through Genkit.

## Installation

Choose one of the options below. Release packages include a native binary for Windows, Linux, and macOS on x64 and ARM64. Go is only needed when building from source.

### npm

Requires Node.js 22 or later.

[![npm](https://img.shields.io/npm/v/@deahesi/agents-toolchain)](https://www.npmjs.com/package/@deahesi/agents-toolchain)

```sh
npm install -g @deahesi/agents-toolchain
atc --help
```

Or run without installing globally:

```sh
npx @deahesi/agents-toolchain --help
```

Keep optional dependencies enabled: npm uses them to install the binary for your platform. Installation also works with `--ignore-scripts`.

For installation through GitHub Packages, see [registry setup](build/README.md#npm-registries).

### pip

Requires Python 3.10 or later. Install in a virtual environment and keep it activated when using `atc`.

```sh
python -m pip install agents-toolchain
atc --help
```

You can also run `python -m agents_toolchain`. Wheels include the binary and support Windows, macOS 12+, and Linux with glibc or musl.

### Homebrew

```sh
brew install --cask Deahesi/tap/agents-toolchain
atc --help
```

Update with `brew upgrade --cask agents-toolchain`.

### GitHub Releases and Linux packages

Download an archive, `.deb`, or `.rpm` for your platform from [GitHub Releases](https://github.com/Deahesi/agents-toolchain/releases).

Extract an archive and add the directory containing `atc` to `PATH`. On Debian or Ubuntu, install the downloaded `.deb` with `sudo apt install ./<filename>.deb`; on Fedora, use `sudo dnf install ./<filename>.rpm`. Replace `<filename>` with the downloaded package name. Linux packages install the command at `/usr/bin/atc`.

### From source

Requires Go 1.26.5 or later.

```sh
go install github.com/Deahesi/agents-toolchain/cmd/atc@latest
```

Add `GOBIN` to `PATH`, or `GOPATH/bin` if `GOBIN` is unset. The usual location on Windows is `%USERPROFILE%\go\bin`. Open a new terminal after changing `PATH`.

## Quick start

Run these commands from the project you want the agent to work on.

1. Set your provider's credentials. For OpenAI, create a `.env` file:

   ```dotenv
   OPENAI_API_KEY=your-api-key
   ```

2. Initialize the project and create an agent:

   ```sh
   atc init
   atc add hello
   ```

   Skip `init` if `agents-toolchain.yml` already exists. `add` asks for the agent's settings and saves them to `agents/hello/agent.yml`. Choose a model available through your provider.

3. Check the config and run the agent:

   ```sh
   atc check hello
   atc run hello --prompt "Hello!"
   ```

Use `--prompt` or `-p` to give the agent a request. Without it, the agent follows the task in `system_prompt`.

## Commands

| Command | What it does |
| --- | --- |
| `atc init` | Creates `agents-toolchain.yml` and the agents directory. |
| `atc add <name>` | Creates an agent through interactive prompts. |
| `atc list` | Shows agents with their descriptions, providers, and models. |
| `atc check <name>` | Checks an agent's YAML and config values without calling the model. |
| `atc run <name> [-p "request"]` | Runs an agent and streams its response and tool activity. |
| `atc get-value <name> <field>` | Shows a single config value in a table. |
| `atc set-value <name> <field> <value>` | Updates a config field. |

Use `atc --help` or `atc <command> --help` for available flags. `atc --version` prints the CLI version. Command errors go to stderr and return exit code `1`. `list` reports unreadable configs and skips them.

### Read and update settings

Fields use YAML names separated by dots. List indices start at zero.

```sh
atc get-value hello agent.model
atc get-value hello agent.temperature
atc set-value hello agent.temperature 0.4
atc set-value hello agent.reasoning true
atc set-value hello agent.max_output_tokens null
```

For an agent with tools, use paths such as `agent.tools.0.name` to read or change a list item. `get-value` currently displays scalar values, such as strings, numbers, and booleans; it does not render whole lists or objects.

`set-value` treats string fields as literal text and parses other values as YAML. Use `null` to clear an optional field. The full config must pass validation before it is saved. Unknown fields and invalid list indices are rejected, and `agent.name` must still match its directory.

Updates keep existing comments. A replacement file is written before saving over the original; whether the final rename is atomic depends on the OS. Updates that traverse or replace anchored fields are rejected.

### Choose project and working directories

`atc init --agents-dir configs/agents` stores agents in a different directory inside the project.

All commands accept `--workspace-dir <directory>` to select the project containing `agents-toolchain.yml`:

```sh
atc list --workspace-dir ./my-project
atc run hello --workspace-dir ./my-project
```

`--workspace-dir` selects the config location. Relative `work_dir` values and the `.env` location still use the directory where you started the CLI. Set `work_dir` to the directory the agent should work in.

## Configuration

`agents-toolchain.yml` contains the path to the agents directory, relative to the project:

```yaml
agents_dir: agents
```

Each agent has its own file at `agents/<name>/agent.yml`:

```yaml
version: "1.0"
agent:
  name: hello
  description: My first agent
  provider: openai
  model: gpt-4o
  temperature: 0.2
  work_dir: .
  system_prompt: |
    You are a helpful assistant. Answer clearly and to the point.
  tools: []
```

| Field | Meaning |
| --- | --- |
| `version` | Config version. Currently `"1.0"`. |
| `name` | Agent name; must match its directory. Use 1–64 ASCII letters, digits, hyphens, or underscores, starting with a letter or digit. Windows device names such as `CON` are reserved. |
| `description` | A short description shown by `atc list`. |
| `provider` | `openai`, `anthropic`, `openrouter`, or `ollama`. |
| `model` | The model ID used by the provider. For OpenRouter, keep its model namespace, such as `openai/gpt-4o`. |
| `temperature` | A number from `0` to `2`. The runtime omits it for some reasoning models. |
| `work_dir` | An existing directory used by file tools. Relative paths start from the CLI's current directory. |
| `system_prompt` | The agent's instructions and task. |
| `tools` | The tools the agent can call. Use `[]` for text-only tasks. |
| `max_output_tokens` | Optional positive output token limit. |
| `reasoning` | Optional `true` or `false`. Omit it to use the model's default. |
| `think` | Older Ollama-only setting for reasoning. If both fields are set, they must agree. |

`max_output_tokens` and `reasoning` are converted to provider-specific settings. Model support varies, so a valid config can still be rejected by the provider. For Anthropic's manual thinking mode, an explicit output limit must be greater than 1024. Disabling reasoning for Ollama's `gpt-oss` models is rejected locally. `check` validates the config; provider-specific checks also happen when running the agent.

The config also accepts `memory`, and generated configs include a `local_file` memory entry. The runtime does not yet load or save conversation history.

### Provider credentials

The CLI reads `.env` from the current directory at startup. Existing environment variables take priority. You can skip `.env` if the variables are already set.

| Provider | Environment variables |
| --- | --- |
| `openai` | `OPENAI_API_KEY`; optional `OPENAI_BASE_URL`. |
| `anthropic` | `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN`. |
| `openrouter` | `OPENROUTER_API_KEY`. |
| `ollama` | Optional `OLLAMA_HOST`; defaults to `http://localhost:11434`. |

For Ollama, start the server and make sure the configured model is installed.

## Tools

Add each tool under `agent.tools` with `type: builtin`:

```yaml
tools:
  - type: builtin
    name: list_files
  - type: builtin
    name: read_file
```

| Tool | What it does |
| --- | --- |
| `list_files` | Lists files and directories at one level. |
| `search_files` | Finds names matching a pattern such as `*.go` at one level. |
| `read_file` | Reads a file. |
| `write_file` | Creates or overwrites a file. |
| `edit_file` | Replaces text in an existing file. |
| `execute_command` | Runs a shell command. |
| `get_environment` | Returns the OS, architecture, and selected environment variables. |
| `get_datetime` | Returns the current date and time. |

Only built-in tools are currently registered. MCP and custom tool types are not implemented. Tool calls run without a confirmation prompt; `allow_without_confirm` does not currently change that behavior.

### File access

File tools use paths relative to `work_dir`. They reject absolute paths, `..` components, Windows alternate data streams, and links that lead outside that directory. Listing and searching do not recurse into subdirectories; the agent must explore them explicitly. Search patterns match entry names, not file contents.

Config access is limited to the selected workspace and, for agent files, to that agent's own directory. Shared file operations live in `internal/files` and use `os.Root` for path resolution.

These checks limit file tools. They do not sandbox shell commands or isolate hard links, mount points, or special files.

### Shell commands

`execute_command` takes one string containing the full command:

```json
{"command": "git diff --staged"}
```

It uses `sh` on Linux/macOS and `powershell.exe` on Windows. Quoting, pipes, and redirection follow that shell's syntax; Windows PowerShell does not support `&&`.

Commands run from the directory where the CLI was started, with a 30-second timeout and the current user's permissions. `work_dir` does not change the shell's working directory or restrict its file access.

## Examples

These examples use OpenAI. Set `OPENAI_API_KEY` and choose a model that supports tool calls. Create each folder and `agent.yml` manually, or use `atc add <name>` and replace its config.

### Write documentation

Save as `agents/docs/agent.yml`:

```yaml
version: "1.0"
agent:
  name: docs
  description: Write documentation from project files
  provider: openai
  model: gpt-4o
  temperature: 0.2
  work_dir: .
  system_prompt: |
    Read the project files and existing documentation.
    Explore relevant subdirectories before making changes.
    Update README.md with an overview, setup steps, and usage examples.
    Describe only what the code supports. Keep the documentation language.
    Change only documentation files. Finish with a brief list of changes.
  tools:
    - type: builtin
      name: list_files
    - type: builtin
      name: read_file
    - type: builtin
      name: write_file
    - type: builtin
      name: edit_file
```

```sh
atc run docs
```

### Write tests

Copy the documentation config to `agents/tests/agent.yml`, change `name` to `tests`, and set a suitable description. Replace `system_prompt` with:

```yaml
system_prompt: |
  Read the source, existing tests, and test configuration.
  Add tests for normal behavior, edge cases, and errors.
  Follow the project's test framework and style. Change only test files.
  Run the relevant tests. Report what you added and whether the tests passed.
  If you could not run them, explain why.
```

Add this entry to `agent.tools`, keeping the file tools:

```yaml
- type: builtin
  name: execute_command
```

```sh
atc run tests
```

### Review a project

Save as `agents/review/agent.yml`:

```yaml
version: "1.0"
agent:
  name: review
  description: Review the project without changing files
  provider: openai
  model: gpt-4o
  temperature: 0.2
  work_dir: .
  system_prompt: |
    Read the source, configs, and tests. Explore relevant subdirectories.
    Look for bugs and security issues. Do not modify files.
    For each finding, report the file, line number, problem, and impact.
    Support each finding with evidence from the code.
    If no problems are found, say so. If the review is incomplete, explain why.
  tools:
    - type: builtin
      name: list_files
    - type: builtin
      name: read_file
```

```sh
atc run review
```

You can ask for JSON or another response format in `system_prompt`, but the CLI does not enforce an output schema. A model's finding or a line such as `Status: error` does not determine the command's exit code.