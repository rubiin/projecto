package helper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestConfigFileExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "config.json")
	if err := os.WriteFile(existing, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", existing, true},
		{"missing file", filepath.Join(dir, "missing.json"), false},
		{"directory", dir, false},
		{"nonexistent path", "does-not-exist-xyz/file.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfigFileExists(tt.path); got != tt.want {
				t.Errorf("ConfigFileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDirName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"unix path", "/home/user/projecto", "projecto"},
		{"relative path", "projecto", "projecto"},
		{"trailing slash", "/home/user/projecto/", "projecto"},
		{"relative with dot", "./projecto", "projecto"},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name string
			path string
			want string
		}{"windows path", `C:\Users\user\projecto`, "projecto"})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DirName(tt.path); got != tt.want {
				t.Errorf("DirName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCurrentDir(t *testing.T) {
	path, name := CurrentDir()

	if path == "" {
		t.Error("CurrentDir() path should not be empty")
	}

	want := filepath.Base(path)
	if name != want {
		t.Errorf("CurrentDir() name = %q, want %q", name, want)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	if path != wd {
		t.Errorf("CurrentDir() path = %q, want %q", path, wd)
	}
}

func TestUserConfigDirXDGPreference(t *testing.T) {
	t.Run("XDG_CONFIG_HOME absolute path is honored", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		got, err := userConfigDir()
		if err != nil {
			t.Fatalf("userConfigDir() error: %v", err)
		}
		if got != filepath.Clean(dir) {
			t.Errorf("userConfigDir() = %q, want %q", got, filepath.Clean(dir))
		}
	})

	t.Run("XDG_CONFIG_HOME relative path is rejected", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "relative/path")

		if _, err := userConfigDir(); err == nil {
			t.Error("userConfigDir() with relative XDG_CONFIG_HOME should fail")
		}
	})

	t.Run("falls back to os.UserConfigDir when unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		want, wantErr := os.UserConfigDir()
		got, err := userConfigDir()

		if wantErr != nil {
			// os.UserConfigDir fails when HOME is unset; mirror that.
			if err == nil {
				t.Error("userConfigDir() should fail when os.UserConfigDir fails")
			}
			return
		}
		if err != nil {
			t.Fatalf("userConfigDir() error: %v", err)
		}
		if got != want {
			t.Errorf("userConfigDir() = %q, want %q", got, want)
		}
	})
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()

	want := Projecto{
		CommandToOpen: "code",
		Projects: []Project{
			{Name: "projecto", Path: "/home/user/projecto", Editor: "atom"},
			{Name: "dotfiles", Path: "/home/user/dotfiles"},
		},
	}

	WriteConfigFile(want, dir)

	if !ConfigFileExists(filepath.Join(dir, configFileName)) {
		t.Fatal("config file was not created by WriteConfigFile")
	}

	got := ReadConfigFile(dir)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip mismatch:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestProjectEditorOmittedWhenEmpty(t *testing.T) {
	data, err := json.Marshal(Project{Name: "dotfiles", Path: "/home/user/dotfiles"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := raw["editor"]; ok {
		t.Error("editor field should be omitted when empty")
	}
	for _, key := range []string{"name", "path"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("field %q should always be present", key)
		}
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	if filepath.Base(path) != configFileName {
		t.Errorf("ConfigPath() = %q, want file name %q", path, configFileName)
	}
	if filepath.Base(filepath.Dir(path)) != appName {
		t.Errorf("ConfigPath() = %q, want parent directory %q", path, appName)
	}
}

func TestMigrateLegacyConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	legacy := filepath.Join(base, configFileName)
	current := filepath.Join(base, appName, configFileName)

	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	t.Run("moves legacy file into projecto subdirectory", func(t *testing.T) {
		write(legacy, "{\"commandToOpen\":\"vim\"}")

		done, err := MigrateLegacyConfig()
		if err != nil {
			t.Fatalf("MigrateLegacyConfig() error: %v", err)
		}
		if !done {
			t.Fatal("migration should have been performed")
		}
		if !ConfigFileExists(current) {
			t.Error("config file should exist at the new location")
		}
		if ConfigFileExists(legacy) {
			t.Error("legacy config file should have been removed")
		}
	})

	t.Run("no-op when there is no legacy file", func(t *testing.T) {
		if err := os.Remove(current); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}

		done, err := MigrateLegacyConfig()
		if err != nil {
			t.Fatalf("MigrateLegacyConfig() error: %v", err)
		}
		if done {
			t.Error("migration should not have been performed")
		}
	})

	t.Run("no-op when the new location already exists", func(t *testing.T) {
		write(legacy, "{\"commandToOpen\":\"code\"}")
		write(current, "{\"commandToOpen\":\"vim\"}")

		done, err := MigrateLegacyConfig()
		if err != nil {
			t.Fatalf("MigrateLegacyConfig() error: %v", err)
		}
		if done {
			t.Error("migration should not overwrite the existing config")
		}
		if got := ReadConfigFile(filepath.Join(base, appName)); got.CommandToOpen != "vim" {
			t.Errorf("existing config was overwritten: commandToOpen = %q", got.CommandToOpen)
		}
	})
}
