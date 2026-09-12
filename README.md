# Projecto

[![CI](https://github.com/rubiin/projecto/actions/workflows/go.yml/badge.svg)](https://github.com/rubiin/projecto/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rubiin/projecto.svg)](https://pkg.go.dev/github.com/rubiin/projecto)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Projecto is a small CLI tool that launches your project folders directly in your editor of choice. Instead of opening a terminal, navigating to the project, and typing an editor command, Projecto keeps an interactive registry of your projects and opens the selected one in a single command.

## Features

- Interactive project picker powered by [promptui](https://github.com/manifoldco/promptui)
- Per-project editor overrides with a global fallback
- Cross-platform (Linux, macOS, Windows)
- Human-readable JSON configuration, editable directly with `--edit`

## Installation

Requires [Go](https://go.dev/dl/) 1.26 or newer:

```sh
go install github.com/rubiin/projecto@latest
```

## Usage

| Command | Description |
| --- | --- |
| `projecto --add` | Register the current directory as a project |
| `projecto --add --editor` | Register the current directory and choose a dedicated editor for it |
| `projecto --open` | Pick a project from an interactive list and open it |
| `projecto --rm` | Remove a project from the registry |
| `projecto --seteditor <cmd>` | Set the global editor used for projects without their own editor |
| `projecto --rmeditor` | Remove the editor override from a project |
| `projecto --edit` | Open the configuration file in the default application |
| `projecto --help` | Show all available options |

Editors are stored as shell commands, so anything your terminal can launch works — for example `code`, `vim`, `idea`, or `subl`.

## Configuration

Projecto stores its configuration in `projecto.json` inside your user configuration directory (`$XDG_CONFIG_HOME` or `~/.config` on Linux, `~/Library/Application Support` on macOS, `%AppData%` on Windows).

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

When opening a project, Projecto uses the project's `editor` if set, and falls back to `commandToOpen` otherwise. The file is created with these defaults on first run and can be edited by hand with `projecto --edit`.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

Distributed under the [MIT License](LICENSE).
