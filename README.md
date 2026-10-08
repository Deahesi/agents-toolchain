# Agents Toolchain (atc)

## Overview

A CLI for creating and running AI agents. Define an agent's task, model, and tools in YAML, then run it from the terminal. Agents can read project files, write documentation and tests, or review code.

## Installation

All release packages contain a ready-to-run Go binary. The command is always `atc`.

### npm (Windows, Linux, macOS)

Requires Node.js 22 or later. Install globally:

```sh
npm install -g @deahesi/agents-toolchain
atc --help
atc --version
```

Or run it without a global installation:

```sh
npx @deahesi/agents-toolchain --help
npx @deahesi/agents-toolchain init
```

The npm package includes a launcher and installs the native binary for Windows, Linux, or macOS on x64 or ARM64. Go is not required. Installation also works with `--ignore-scripts`; keep optional dependencies enabled so npm can install the platform package. An unsupported platform or a missing platform package produces an error with installation guidance. The command is `atc` for both local and global installations.

### Homebrew (macOS, Linux)

```sh
brew install --cask Deahesi/tap/agents-toolchain
atc --help
```

The cask selects the x64 or ARM64 archive from GitHub Releases. It requires neither Go nor Node.js. Updates use `brew upgrade --cask agents-toolchain`.

### pip (Windows, Linux, macOS)

Requires Python 3.10 or later. Install in your Python virtual environment:

```sh
python -m pip install agents-toolchain
atc --help
python -m agents_toolchain --version
```

Platform wheels include the native binary; installation does not compile Go or download executables from GitHub. Wheels cover x64 and ARM64 on Windows, macOS 12+, and Linux with glibc or musl. There is no source distribution that would require a Go compiler. Keep your virtual environment activated to use its `atc` command.

### Debian / Ubuntu and RPM distributions

Download the package for your architecture from [GitHub Releases](https://github.com/Deahesi/agents-toolchain/releases). For example, for version `0.1.2` on x64:

```sh
# Debian / Ubuntu
sudo apt install ./agents-toolchain_0.1.2_amd64.deb

# Fedora / RHEL family
sudo dnf install ./agents-toolchain-0.1.2-1.x86_64.rpm

atc --help
```

ARM64 packages end in `_arm64.deb` and `.aarch64.rpm`. Packages install `/usr/bin/atc`, the README, and the MIT license. These are downloadable packages, not an APT or YUM repository.

### From source

Alternatively, build from source with Go 1.26.5 or later:

```powershell
go install github.com/Deahesi/agents-toolchain/cmd/atc@latest
```

Make sure the installation directory is in `PATH`. Go uses `GOBIN` if set, otherwise `GOPATH/bin` (usually `%USERPROFILE%\go\bin` on Windows). `go install` does not update `PATH`; after adding the directory, open a new terminal.

## Base usage

Run the following steps from the project directory you want the agent to work on.

1. Create a `.env` file with your provider's API key:

   ```dotenv
   OPENAI_API_KEY=your-api-key
   ```

2. Initialize the project and create an agent:

   ```powershell
   atc init
   atc add hello
   ```

   Skip `init` if `agents-toolchain.yml` already exists. The `add` command asks for the agent's settings and saves them to `agents/hello/agent.yml`.

3. Run the agent:

   ```powershell
   atc run hello --prompt "Hello!"
   ```

Use `--prompt` or `-p` to provide a request. Without it, the agent follows the task in `system_prompt`.

Use `atc init --agents-dir <directory>` to choose where agent configs are stored. All commands accept `--workspace-dir <directory>` to choose the project containing `agents-toolchain.yml`. File tools and `.env` still use the current working directory, so launch the CLI from the project you want the agent to work on.

## Config

`agents-toolchain.yml` specifies the directory containing agents:

```yaml
agents_dir: agents
```

Each agent has a config at `agents/<name>/agent.yml`:

```yaml
version: "1.0"
agent:
  name: hello
  description: My first agent
  provider: openai
  model: gpt-4o
  temperature: 0.2
  system_prompt: |
    You are a helpful assistant. Answer clearly and to the point.
  tools: []
```

- `name` must match the agent directory name.
- `description` briefly describes the agent.
- `provider` is `openai`, `anthropic`, `ollama`, or `openrouter`.
- `model` is the model ID for that provider, without adding the provider prefix.
- `temperature` ranges from `0` to `2`; lower values give more predictable responses.
- `system_prompt` describes the agent's task and expected output.
- `tools` lists the tools the agent can use. Leave it empty for text-only tasks.

Built-in tools use `type: builtin`. Available names are `list_files`, `search_files`, `read_file`, `write_file`, `edit_file`, `execute_command`, `get_environment`, and `get_datetime`.

The CLI automatically reads `.env` from the current working directory at startup. Existing environment variables take priority. The file is optional if the variables are already set; `--workspace-dir` does not change where `.env` is read.

| Provider | Environment variables |
| --- | --- |
| `openai` | `OPENAI_API_KEY`; optional `OPENAI_BASE_URL` |
| `anthropic` | `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN` |
| `openrouter` | `OPENROUTER_API_KEY` |
| `ollama` | Optional `OLLAMA_HOST`, defaults to `http://localhost:11434` |

## Examples

Save each config at the path shown. You can create the folder and file manually or use `atc add <name>` and replace the generated config. These examples use OpenAI; set `OPENAI_API_KEY` in `.env` and choose a model that supports tools.

### Documentation from project files

`agents/docs/agent.yml`:

```yaml
version: "1.0"
agent:
  name: docs
  description: Write documentation from local project files
  provider: openai
  model: gpt-4o
  temperature: 0.2
  system_prompt: |
    Read the local project, starting from the current directory.
    Explore relevant subdirectories and inspect source files, configs,
    and existing documentation. Write or update README.md with a simple
    overview, setup steps, configuration, and usage examples.
    Describe only behavior supported by the files you read.
    Keep the project's documentation language and style.
    Change only documentation files. End with a brief list of changes.
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

```powershell
atc run docs
```

### Automated tests from a local project

`agents/tests/agent.yml`:

```yaml
version: "1.0"
agent:
  name: tests
  description: Write and run tests for the local project
  provider: openai
  model: gpt-4o
  temperature: 0.2
  system_prompt: |
    Read the local project, starting from the current directory.
    Explore relevant subdirectories, source files, existing tests,
    and test configuration. Use the project's test framework and style.
    Add meaningful tests for normal behavior, edge cases, and errors.
    Change only test files. Run the relevant tests with execute_command.
    End with a brief list of added tests and their results.
    If tests could not be run, say so and explain why.
  tools:
    - type: builtin
      name: list_files
    - type: builtin
      name: read_file
    - type: builtin
      name: write_file
    - type: builtin
      name: edit_file
    - type: builtin
      name: execute_command
```

```powershell
atc run tests
```

### Project review with structured findings

`agents/review/agent.yml`:

```yaml
version: "1.0"
agent:
  name: review
  description: Review the local project and report problems
  provider: openai
  model: gpt-4o
  temperature: 0.2
  system_prompt: |
    Review the local project, starting from the current directory.
    Explore relevant subdirectories and read source files, configs,
    and tests. Look for bugs, security issues, and incorrect behavior.
    Report only problems supported by the files you read.
    Do not modify files.
    In your final response, report each problem as a separate block:
    file: <relative file path>
    row: <line number or range, starting at 1>
    problem: <what is wrong and its impact>
    Separate blocks with a blank line. Do not wrap them in Markdown fences.
    If problems were found, the final line must be exactly:
    Status: error
    If no problems were found after completing the review, output only:
    Status: success
    If the review could not be completed, report the blocker using the
    same block format, use row: N/A when needed, and end with Status: error.
    Do not output anything after the status line.
  tools:
    - type: builtin
      name: list_files
    - type: builtin
      name: read_file
```

```powershell
atc run review
```

Example final response when a problem is found:

```text
file: src/config.go
row: 42
problem: An empty API key is accepted, causing requests to fail later.

Status: error
```

With no problems:

```text
Status: success
```