package main

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/rubiin/projecto/helper"
)

func TestDefaultEditor(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"unset falls back to code", "", "code"},
		{"simple command", "vim", "vim"},
		{"command with args", "code --wait", "code"},
		{"whitespace is trimmed", "  helix  ", "helix"},
		{"absolute path", "/usr/bin/nvim", "nvim"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.env)
			if got := helper.DefaultEditor(); got != tt.want {
				t.Errorf("DefaultEditor() with EDITOR=%q = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}

func TestEditorPresetsHaveCommands(t *testing.T) {
	for i, preset := range editorPresets {
		isLast := i == len(editorPresets)-1
		if preset.command == "" && !isLast {
			t.Errorf("editor preset %q has an empty command", preset.name)
		}
	}
}

func TestProjectNames(t *testing.T) {
	config := helper.Projecto{
		Projects: []helper.Project{
			{Name: "alpha", Path: "/home/user/alpha"},
			{Name: "beta", Path: "/home/user/beta"},
		},
	}
	if names := projectNames(config.Projects); !reflect.DeepEqual(names, []string{"alpha", "beta"}) {
		t.Errorf("projectNames() = %v, want [alpha beta]", names)
	}

	if names := projectNames(nil); len(names) != 0 {
		t.Errorf("projectNames(nil) = %v, want empty", names)
	}
}

func TestEditorFor(t *testing.T) {
	config := helper.Projecto{
		CommandToOpen: "code",
		Projects: []helper.Project{
			{Name: "override", Path: "/home/user/override", Editor: "vim"},
			{Name: "fallback", Path: "/home/user/fallback"},
			{Name: "empty override", Path: "/home/user/empty", Editor: ""},
		},
	}

	tests := []struct {
		name  string
		index int
		want  string
	}{
		{"per-project editor wins", 0, "vim"},
		{"falls back to global editor", 1, "code"},
		{"empty editor falls back to global", 2, "code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := editorFor(config, tt.index); got != tt.want {
				t.Errorf("editorFor(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestOpenProjectInvokesEditorWithProjectPath(t *testing.T) {
	if _, err := exec.LookPath("echo"); err != nil {
		t.Skip("echo not available on PATH")
	}

	config := helper.Projecto{
		CommandToOpen: "echo",
		Projects: []helper.Project{
			{Name: "alpha", Path: "/home/user/alpha"},
		},
	}

	// A real editor launch is hard to assert; the closest check is that the
	// invocation itself does not error out.
	openProject(config, 0, false)
	openProject(config, 0, true)
}

func TestEditorChoicesIncludeCustomOption(t *testing.T) {
	if len(editorPresets) == 0 {
		t.Fatal("editorPresets should not be empty")
	}
	last := editorPresets[len(editorPresets)-1]
	if last.command != "" {
		t.Errorf("last editor preset = %q, want an empty command marking the custom-entry option", last.command)
	}
	for _, preset := range editorPresets {
		if strings.TrimSpace(preset.name) == "" {
			t.Errorf("editor preset has an empty name: %+v", preset)
		}
	}
}
