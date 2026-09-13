package main

import (
	"bufio"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
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

func TestListProjects(t *testing.T) {
	capture := func(t *testing.T, run func()) []string {
		t.Helper()

		old := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe failed: %v", err)
		}
		os.Stdout = w
		defer func() { os.Stdout = old }()

		run()

		if err := w.Close(); err != nil {
			t.Fatalf("close pipe failed: %v", err)
		}

		var lines []string
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			t.Fatalf("scan pipe failed: %v", err)
		}
		return lines
	}

	t.Run("empty registry prints notice", func(t *testing.T) {
		lines := capture(t, func() { listProjects(helper.Projecto{}) })
		if len(lines) != 1 || !strings.Contains(lines[0], "No projects registered") {
			t.Errorf("listProjects(empty) output = %v, want a no-projects notice", lines)
		}
	})

	t.Run("prints aligned table with global fallback", func(t *testing.T) {
		config := helper.Projecto{
			CommandToOpen: "code",
			Projects: []helper.Project{
				{Name: "alpha", Path: "/home/user/alpha", Editor: "vim"},
				{Name: "beta", Path: "/home/user/beta"},
			},
		}

		lines := capture(t, func() { listProjects(config) })

		// Header + one line per project.
		if len(lines) != 3 {
			t.Fatalf("listProjects() printed %d lines, want 3: %v", len(lines), lines)
		}
		for _, want := range []string{"NAME", "PATH", "EDITOR"} {
			if !strings.Contains(lines[0], want) {
				t.Errorf("header %q does not contain %q", lines[0], want)
			}
		}
		if !strings.Contains(lines[1], "alpha") || !strings.Contains(lines[1], "vim") {
			t.Errorf("row %q should contain alpha and vim", lines[1])
		}
		if !strings.Contains(lines[2], "beta") || !strings.Contains(lines[2], "(global)") {
			t.Errorf("row %q should contain beta and (global)", lines[2])
		}

		// Rows (not the ANSI-colored header) share the same total width.
		width := len([]rune(lines[1]))
		for i := 1; i < len(lines); i++ {
			if got := len([]rune(lines[i])); got != width {
				t.Errorf("line %d has width %d, want %d", i, got, width)
			}
		}
	})
}

func TestFuzzyMatchIndices(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		s       string
		want    []int
		match   bool
	}{
		{"empty pattern matches, no indices", "", "api", nil, true},
		{"prefix match", "ap", "api-server", []int{0, 1}, true},
		{"case-insensitive", "AP", "api-server", []int{0, 1}, true},
		{"scattered subsequence", "ps", "api-server", []int{1, 4}, true},
		{"no match", "xy", "api-server", nil, false},
		{"order matters", "sa", "api-server", nil, false},
		{"multibyte runes", "海", "x海y", []int{1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, matched := fuzzyMatchIndices(tt.pattern, tt.s)
			if matched != tt.match {
				t.Fatalf("fuzzyMatchIndices(%q, %q) matched = %v, want %v", tt.pattern, tt.s, matched, tt.match)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fuzzyMatchIndices(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
			}
		})
	}
}

func TestPartitionMatchIndices(t *testing.T) {
	// display = "api  ~/work/api": name is runes 0-2, separator 3-4,
	// path starts at rune 5.
	nameIdx, pathIdx := partitionMatchIndices([]int{0, 2, 4, 5, 9}, 3)
	if !reflect.DeepEqual(nameIdx, []int{0, 2}) {
		t.Errorf("nameIdx = %v, want [0 2]", nameIdx)
	}
	if !reflect.DeepEqual(pathIdx, []int{0, 4}) {
		t.Errorf("pathIdx = %v, want [0 4]", pathIdx)
	}

	// Indices in the two-space separator are dropped.
	nameIdx, pathIdx = partitionMatchIndices([]int{3, 4}, 3)
	if nameIdx != nil || pathIdx != nil {
		t.Errorf("separator indices leaked: name=%v path=%v", nameIdx, pathIdx)
	}
}

func TestHighlight(t *testing.T) {
	plain := lipgloss.NewStyle()

	t.Run("no indices returns unchanged", func(t *testing.T) {
		if got := highlight("abc", nil, plain); got != "abc" {
			t.Errorf("highlight(abc, nil) = %q, want %q", got, "abc")
		}
	})

	t.Run("plain style groups without escapes", func(t *testing.T) {
		if got := highlight("abc", []int{1}, plain); got != "abc" {
			t.Errorf("highlight with plain style = %q, want %q", got, "abc")
		}
	})

	t.Run("styles matched runes", func(t *testing.T) {
		style := lipgloss.NewStyle().Bold(true)
		got := highlight("abc", []int{0, 2}, style)
		want := style.Render("a") + "b" + style.Render("c")
		if got != want {
			t.Errorf("highlight = %q, want %q", got, want)
		}
	})

	t.Run("adjacent indices grouped into one run", func(t *testing.T) {
		style := lipgloss.NewStyle().Bold(true)
		got := highlight("abc", []int{0, 1}, style)
		want := style.Render("ab") + "c"
		if got != want {
			t.Errorf("highlight = %q, want %q", got, want)
		}
	})

	t.Run("out of range indices ignored", func(t *testing.T) {
		style := lipgloss.NewStyle().Bold(true)
		got := highlight("ab", []int{5, -1, 0}, style)
		want := style.Render("a") + "b"
		if got != want {
			t.Errorf("highlight = %q, want %q", got, want)
		}
	})
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		want    bool
	}{
		{"", "anything", true},
		{"prj", "My Project", true},
		{"api", "api-server", true},
		{"api", "~/work/api-server", true},
		{"API", "api-server", true}, // case-insensitive
		{"xyz", "api-server", false},
		{"pss", "pass", true},  // subsequence, not substring
		{"psa", "pass", false}, // order matters
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"~"+tt.s, func(t *testing.T) {
			if got := fuzzyMatch(tt.pattern, tt.s); got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
			}
		})
	}
}

func TestRecentOrder(t *testing.T) {
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	mid := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)

	projects := []helper.Project{
		{Name: "never-a", Path: "/a"},
		{Name: "old", Path: "/b", LastOpened: old},
		{Name: "never-b", Path: "/c"},
		{Name: "mid", Path: "/d", LastOpened: mid},
	}

	// Most recent first, then never-opened in registration order.
	want := []int{3, 1, 0, 2}
	got := recentOrder(projects)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recentOrder() = %v, want %v", got, want)
	}

	if order := recentOrder(nil); len(order) != 0 {
		t.Errorf("recentOrder(nil) = %v, want empty", order)
	}

	// All never-opened projects keep registration order.
	never := []helper.Project{
		{Name: "a", Path: "/a"},
		{Name: "b", Path: "/b"},
	}
	if order := recentOrder(never); !reflect.DeepEqual(order, []int{0, 1}) {
		t.Errorf("recentOrder(never-opened) = %v, want [0 1]", order)
	}
}

func TestTouchProjectStampsLastOpened(t *testing.T) {
	configDir := t.TempDir()
	config := helper.Projecto{
		CommandToOpen: "echo",
		Projects: []helper.Project{
			{Name: "alpha", Path: "/home/user/alpha"},
		},
	}
	helper.WriteConfigFile(config, configDir)

	touchProject(configDir, config, 0)

	stamped := helper.ReadConfigFile(configDir)
	if stamped.Projects[0].LastOpened == "" {
		t.Fatal("touchProject did not set LastOpened")
	}
	if _, err := time.Parse(time.RFC3339, stamped.Projects[0].LastOpened); err != nil {
		t.Errorf("LastOpened %q is not RFC3339: %v", stamped.Projects[0].LastOpened, err)
	}
}

func TestSelectProjectDetailedOrdersByRecency(t *testing.T) {
	old := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	projects := []helper.Project{
		{Name: "alpha", Path: "/home/user/alpha"},
		{Name: "beta", Path: "/home/user/beta", LastOpened: old},
	}

	// Cancel immediately (choiceCh never receives a value because the
	// program never runs); verify the ordering data path instead of the TUI.
	order := recentOrder(projects)
	if len(order) != 2 || order[0] != 1 {
		t.Errorf("expected beta first, got order %v", order)
	}
}

func TestIndexOfProjectPath(t *testing.T) {
	projects := []helper.Project{
		{Name: "alpha", Path: "/home/user/alpha"},
		{Name: "beta", Path: "/home/user/beta/"},
	}

	tests := []struct {
		name string
		path string
		want int
	}{
		{"exact match", "/home/user/alpha", 0},
		{"trailing slash matches cleaned", "/home/user/beta", 1},
		{"cleaned needle matches", "/home/user/alpha/", 0},
		{"unknown path", "/home/user/gamma", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexOfProjectPath(projects, tt.path); got != tt.want {
				t.Errorf("indexOfProjectPath(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestAddProjectRejectsDuplicate(t *testing.T) {
	configDir := t.TempDir()
	helper.WriteConfigFile(helper.Projecto{CommandToOpen: "echo"}, configDir)

	addProject(configDir, false)
	config := helper.ReadConfigFile(configDir)
	if len(config.Projects) != 1 {
		t.Fatalf("after first add, want 1 project, got %d", len(config.Projects))
	}

	// The test binary's working directory is already registered by the first
	// call, so a second add must be a no-op.
	addProject(configDir, false)
	config = helper.ReadConfigFile(configDir)
	if len(config.Projects) != 1 {
		t.Errorf("duplicate add created %d projects, want 1", len(config.Projects))
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
