
# Projecto

[![CI](https://github.com/rubiin/projecto/actions/workflows/ci.yml/badge.svg)](https://github.com/rubiin/projecto/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rubiin/projecto.svg)](https://pkg.go.dev/github.com/rubiin/projecto)
[![Release](https://img.shields.io/github/v/release/rubiin/projecto)](https://github.com/rubiin/projecto/releases/latest)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)


<img width="130" alt="projecto" src="https://github.com/user-attachments/assets/b11863e8-d24a-4d14-930c-bdc7e4348b96" />

Projecto is a small CLI tool that launches your project folders directly in your editor of choice. Instead of opening a terminal, navigating to the project, and typing an editor command, Projecto keeps an interactive registry of your projects and opens the selected one in a single command.

## Features

- **Interactive picker** — clean keyboard-driven project list built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), with fuzzy filtering and name/path columns
- **Recently-used ordering** — the projects you open most often float to the top of the picker
- **Per-project editors** — assign a dedicated editor to any project, with a global fallback for the rest
- **Cross-platform** — single static binary for Linux, macOS, and Windows
- **Simple config** — human-readable JSON you can edit directly with `projecto edit`
- **Shell completions** — bash, zsh, fish, and PowerShell

## Installation

### Go

Requires [Go](https://go.dev/dl/) 1.26 or newer:

```sh
go install github.com/rubiin/projecto@latest
```

### AUR (Arch Linux)

```sh
yay -S projecto-bin
```

The AUR package ships shell completions for bash, zsh, and fish — they are installed system-wide and picked up automatically on new shells.

### From source with just

Requires [Go](https://go.dev/dl/) 1.26+ and [just](https://github.com/casey/just):

```sh
just build        # produces ./projecto with version info from git describe
just --list       # see all available recipes (test, lint, fmt, completions…)
```

### Prebuilt binaries

Download the archive for your platform from the [releases page](https://github.com/rubiin/projecto/releases), extract it, and place the `projecto` binary somewhere on your `PATH`.

## Usage

| Command | Description |
| --- | --- |
| `projecto add [dir]` | Register a directory as a project (defaults to the current directory) |
| `projecto add [dir] --editor` | Register a directory and choose a dedicated editor for it |
| `projecto` | Pick a project from an interactive list and open it (default) |
| `projecto open` | Same as above — the picker is the default command |
| `projecto open --verbose` | Open a project, showing the editor command being invoked |
| `projecto rm` | Remove a project from the registry |
| `projecto seteditor <cmd>` | Set the global editor used for projects without their own editor (defaults to `$EDITOR`) |
| `projecto rmeditor` | Remove the editor override from a project |
| `projecto edit` | Open the configuration file in the default application |
| `projecto --help` | Show all available commands |

A typical workflow looks like this:

```sh
# Inside a project directory, register it and pick its editor
cd ~/projects/api
projecto add --editor

# From anywhere, jump straight into it
projecto open
```

### Shell completions

Projecto can generate completion scripts for bash, zsh, fish, and PowerShell via [urfave/cli](https://github.com/urfave/cli):

```sh
# bash
projecto completion bash | sudo tee /etc/bash_completion.d/projecto >/dev/null

# zsh
projecto completion zsh > "${fpath[1]}/_projecto"

# fish
projecto completion fish > ~/.config/fish/completions/projecto.fish

# powershell
projecto completion powershell > projecto.ps1
```

### Editors

Editors are stored as shell commands, so anything your terminal can launch works — for example `code`, `vim`, `idea`, or `subl`. When adding a project with `projecto add --editor`, you can pick a preset (VS Code, Atom, Sublime Text) or enter a custom command.

## Development

This repository ships a [justfile](https://github.com/casey/just) with common tasks:

| Recipe | Description |
| --- | --- |
| `just build` | Build `./projecto` with version info from `git describe` |
| `just test` | Run all tests |
| `just test-race` | Run tests with the race detector |
| `just vet` | Run `go vet` |
| `just fmt` | Format code with `gofmt` |
| `just lint` | Run `golangci-lint` |
| `just completions <shell>` | Print the completion script for a shell |
| `just clean` | Remove build artifacts |

## Configuration

Projecto stores its data in a `projecto` subdirectory of your user configuration directory (`~/.config/projecto/` on Linux, `~/Library/Application Support/projecto/` on macOS, `%AppData%\projecto` on Windows), with the configuration file at `projecto.json` inside it.

```json
{
	"commandToOpen": "code",
	"projects": [
		{
			"name": "projecto",
			"path": "/home/user/projecto",
			"editor": "vim"
		}
	]
}
```

When opening a project, Projecto uses the project's `editor` if set, and falls back to `commandToOpen` otherwise. The file is created with these defaults on first run and can be edited by hand with `projecto edit`.

### Default editor

On first run, the global editor (`commandToOpen`) is taken from the `EDITOR` environment variable when set — only the command name is kept, so `EDITOR="code --wait"` stores `code`. If `EDITOR` is unset, it defaults to `code`. The same applies to `projecto seteditor` without an argument: it resets the global editor to `$EDITOR` (or `code`).

On Linux and macOS, Projecto follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/): `$XDG_CONFIG_HOME` is honored when set (it must be an absolute path), making the config `$XDG_CONFIG_HOME/projecto/projecto.json`, and falling back to `~/.config/projecto/`. On macOS this means setting `XDG_CONFIG_HOME` moves the config to `~/.config/projecto/` instead of `~/Library/Application Support/projecto/`. 

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[GPL-3.0](./LICENSE)

Made with ❤️ for opensource.
