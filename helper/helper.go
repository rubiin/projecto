// Package helper provides configuration, filesystem, and terminal helpers
// used by projecto.
package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// configFileName is the name of the configuration file stored in the
// projecto configuration directory.
const configFileName = "projecto.json"

// appName is the subdirectory of the user configuration directory where
// projecto stores its data.
const appName = "projecto"

// DefaultEditor returns the editor command to use as the global default:
// the basename of $EDITOR when set, falling back to "code". The value is
// trimmed; arguments (e.g. "code --wait") are stripped because the editor
// is launched with the project path as its sole argument.
func DefaultEditor() string {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return "code"
	}
	if base := filepath.Base(strings.Fields(editor)[0]); base != "." {
		return base
	}
	return editor
}

// ConfigDir returns the directory where projecto stores its data, following
// the XDG Base Directory Specification: $XDG_CONFIG_HOME/projecto when
// XDG_CONFIG_HOME is set to an absolute path, otherwise the platform default
// with a "projecto" subdirectory appended (~/.config/projecto on Linux,
// ~/Library/Application Support/projecto on macOS, %AppData%\projecto on
// Windows).
func ConfigDir() (string, error) {
	base, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// ConfigPath returns the full path of the projecto configuration file inside
// ConfigDir.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// MigrateLegacyConfig moves a configuration file from the pre-projecto
// subdirectory location (the user configuration directory root) to the
// current one, if the legacy file exists and the new one does not. It
// reports whether a migration was performed.
func MigrateLegacyConfig() (bool, error) {
	base, err := userConfigDir()
	if err != nil {
		return false, err
	}

	legacy := filepath.Join(base, configFileName)
	if !ConfigFileExists(legacy) {
		return false, nil
	}

	current, err := ConfigPath()
	if err != nil {
		return false, err
	}
	if ConfigFileExists(current) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(current), 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(legacy, current); err != nil {
		return false, err
	}
	return true, nil
}

// userConfigDir resolves the user configuration directory. It prefers a
// valid XDG_CONFIG_HOME (non-empty and absolute) over os.UserConfigDir,
// which neither honors XDG_CONFIG_HOME on macOS nor validates that the
// value is absolute.
func userConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		if !filepath.IsAbs(xdg) {
			return "", fmt.Errorf("XDG_CONFIG_HOME is set but not an absolute path: %q", xdg)
		}
		return filepath.Clean(xdg), nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return dir, nil
}

// ConfigFileExists reports whether the configuration file exists and is a
// regular file. Any other error is treated as "not exists".
func ConfigFileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirName returns the last element of a path, for example "projecto" for
// both "/home/user/projecto" and "C:\\Users\\user\\projecto".
func DirName(path string) string {
	return filepath.Base(filepath.Clean(path))
}

// ShortenHome abbreviates the user's home directory prefix to "~" so paths
// read as "~/work/api-server". Paths outside the home directory are
// returned unchanged.
func ShortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if prefix := home + string(filepath.Separator); strings.HasPrefix(path, prefix) {
		return "~" + path[len(home):]
	}
	return path
}

// CurrentDir returns the current working directory and its name. It calls
// log.Fatalln on failure, terminating the program.
func CurrentDir() (string, string) {
	path, err := os.Getwd()
	CheckError(err)
	return path, DirName(path)
}

// CheckError terminates the program with the error message if e is not nil.
func CheckError(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

// ReadConfigFile reads and parses the projecto configuration file from the
// given configuration directory.
func ReadConfigFile(configDir string) Projecto {
	path := filepath.Join(configDir, configFileName)

	data, err := os.ReadFile(path)
	CheckError(err)

	var config Projecto
	CheckError(json.Unmarshal(data, &config))
	return config
}

// WriteConfigFile serializes the given configuration and writes it to the
// configuration file in the given directory in a single atomic write.
func WriteConfigFile(config Projecto, configDir string) {
	data, err := json.MarshalIndent(config, "", "\t")
	CheckError(err)

	path := filepath.Join(configDir, configFileName)
	CheckError(os.WriteFile(path, data, 0o644))
}

// OpenConfigFile opens the projecto configuration file with the OS default
// application. On Windows it uses cmd's start builtin; on Linux xdg-open;
// and on macOS open.
func OpenConfigFile() {
	path, err := ConfigPath()
	CheckError(err)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		return
	}

	CheckError(cmd.Start())
}
