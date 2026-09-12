package helper

import (
	"encoding/json"
	"flag"
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

func TestIsFlagPassed(t *testing.T) {
	old := flag.CommandLine
	defer func() { flag.CommandLine = old }()

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	flag.Bool("add", false, "test flag")
	flag.String("editor", "", "test flag")

	if err := flag.CommandLine.Parse([]string{"--add"}); err != nil {
		t.Fatalf("flag parse failed: %v", err)
	}

	tests := []struct {
		name string
		flag string
		want bool
	}{
		{"passed flag", "add", true},
		{"declared but not passed", "editor", false},
		{"undeclared flag", "nope", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFlagPassed(tt.flag); got != tt.want {
				t.Errorf("IsFlagPassed(%q) = %v, want %v", tt.flag, got, tt.want)
			}
		})
	}
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
}
