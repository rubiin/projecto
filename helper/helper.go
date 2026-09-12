// Package helper provides configuration, filesystem, and terminal helpers
// used by projecto.
package helper

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// configFileName is the name of the configuration file stored in the
// user's configuration directory.
const configFileName = "projecto.json"

// ConfigPath returns the full path of the projecto configuration file,
// located in the user's OS-specific configuration directory.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
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

// IsFlagPassed reports whether the named command-line flag was explicitly
// set on the command line.
func IsFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
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
