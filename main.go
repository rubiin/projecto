package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rubiin/projecto/helper"
	"github.com/urfave/cli/v3"
)

// editorPresets maps the editor options offered when registering a project
// with a per-project editor to their commands. An empty command prompts for
// a custom one.
var editorPresets = []struct {
	name    string
	command string
}{
	{"VS Code", "code"},
	{"Atom", "atom"},
	{"Sublime Text", "subl"},
	{"Other…", ""},
}

// listStyles holds the lipgloss styles shared by all pickers.
type listStyles struct {
	title             lipgloss.Style
	item              lipgloss.Style
	selected          lipgloss.Style
	prompt            lipgloss.Style
	hint              lipgloss.Style
	match             lipgloss.Style // matched characters in unselected rows
	matchSelected     lipgloss.Style // matched characters in the selected row
	matchPath         lipgloss.Style // matched characters in the path column
	matchPathSelected lipgloss.Style // path-column matches in the selected row
}

// newListStyles returns the picker styles.
func newListStyles() listStyles {
	return listStyles{
		title:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		item:              lipgloss.NewStyle().PaddingLeft(2),
		selected:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).PaddingLeft(0),
		prompt:            lipgloss.NewStyle().Bold(true).MarginBottom(1),
		hint:              lipgloss.NewStyle().Faint(true).MarginTop(1),
		match:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		matchSelected:     lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("12")),
		matchPath:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		matchPathSelected: lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("6")),
	}
}

// keyMap defines the keybindings for the pickers.
type keyMap struct {
	up       key.Binding
	down     key.Binding
	accept   key.Binding
	cancel   key.Binding
	tab      key.Binding
	shiftTab key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		up:       key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		down:     key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		accept:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		cancel:   key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
		tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next editor")),
		shiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev editor")),
	}
}

// fuzzyMatchIndices reports whether pattern matches s as a case-insensitive
// subsequence and, when it does, returns the rune indices of s at which the
// pattern characters matched so callers can highlight them.
func fuzzyMatchIndices(pattern, s string) ([]int, bool) {
	pRunes := []rune(strings.ToLower(pattern))
	sRunes := []rune(strings.ToLower(s))

	var indices []int
	i := 0
	for j, r := range sRunes {
		if i < len(pRunes) && pRunes[i] == r {
			indices = append(indices, j)
			i++
		}
	}
	if i != len(pRunes) {
		return nil, false
	}
	return indices, true
}

// fuzzyMatch reports whether pattern matches s as a case-insensitive
// subsequence: every rune of pattern appears in s in order, e.g. "prj"
// matches "My Project".
func fuzzyMatch(pattern, s string) bool {
	_, matched := fuzzyMatchIndices(pattern, s)
	return matched
}

// selectModel is a Bubble Tea model that shows a filterable list of strings
// and returns the chosen position. Typing narrows the list with fuzzy
// matching over both the choice and its display text.
type selectModel struct {
	label    string
	choices  []string
	displays []string // parallel to choices; what each row shows
	filtered []int    // indices into choices currently shown
	cursor   int      // position within filtered
	input    textinput.Model
	keys     keyMap
	styles   listStyles
	choiceCh chan int // display position of the accepted choice, -1 on cancel
}

// newSelectModel builds a picker for the given choices. When displays is nil
// the choices themselves are shown.
func newSelectModel(label string, choices, displays []string) selectModel {
	if displays == nil {
		displays = choices
	}

	ti := textinput.New()
	ti.Placeholder = "Filter…"
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(40)

	m := selectModel{
		label:    label,
		choices:  choices,
		displays: displays,
		input:    ti,
		keys:     newKeyMap(),
		styles:   newListStyles(),
		choiceCh: make(chan int, 1),
	}
	m.applyFilter()
	return m
}

// applyFilter recomputes the visible choices from the current input value
// and clamps the cursor to the new range. Filtering matches the choice text
// and its display text.
func (m *selectModel) applyFilter() {
	m.filtered = m.filtered[:0]
	pattern := strings.TrimSpace(m.input.Value())
	for i, choice := range m.choices {
		if pattern == "" || fuzzyMatch(pattern, choice) || fuzzyMatch(pattern, m.displays[i]) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(len(m.filtered)-1, 0)
	}
}

func (m selectModel) Init() tea.Cmd { return textinput.Blink }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.cancel):
			m.choiceCh <- -1
			return m, tea.Quit
		case key.Matches(msg, m.keys.accept):
			if len(m.filtered) > 0 {
				m.choiceCh <- m.cursor
				return m, tea.Quit
			}
			return m, nil // no matches: ignore enter
		case key.Matches(msg, m.keys.up):
			if len(m.filtered) > 0 {
				m.cursor = (m.cursor - 1 + len(m.filtered)) % len(m.filtered)
			}
			return m, nil
		case key.Matches(msg, m.keys.down):
			if len(m.filtered) > 0 {
				m.cursor = (m.cursor + 1) % len(m.filtered)
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m selectModel) View() tea.View {
	return tea.NewView(m.render())
}

func (m selectModel) render() string {
	var b strings.Builder

	b.WriteString(m.styles.prompt.Render(m.label))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString(m.styles.hint.Render("no matches"))
		b.WriteString("\n")
		return b.String()
	}

	// Two-column layout: pad names to the widest visible name so the path
	// column lines up.
	nameWidth := 0
	for _, index := range m.filtered {
		name, _ := splitDisplay(m.displays[index])
		if w := len([]rune(name)); w > nameWidth {
			nameWidth = w
		}
	}

	pathStyle := m.styles.hint
	pattern := strings.TrimSpace(m.input.Value())
	for _, index := range m.filtered {
		display := m.displays[index]
		name, path := splitDisplay(display)
		style := m.styles.item
		matchStyle := m.styles.match
		pathMatchStyle := m.styles.matchPath
		prefix := "  "
		if index == m.filtered[m.cursor] {
			style = m.styles.selected
			matchStyle = m.styles.matchSelected
			pathMatchStyle = m.styles.matchPathSelected
			prefix = "❯ "
		}

		// Highlight the characters matched by the filter, both in the name
		// and in the path column.
		indices, _ := fuzzyMatchIndices(pattern, display)
		nameIdx, pathIdx := partitionMatchIndices(indices, len([]rune(name)))

		padded := name + strings.Repeat(" ", nameWidth-len([]rune(name)))
		b.WriteString(prefix)
		b.WriteString(style.Render(highlight(padded, nameIdx, matchStyle)))
		if path != "" {
			b.WriteString("  ")
			b.WriteString(pathStyle.Render(highlight(path, pathIdx, pathMatchStyle)))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.styles.hint.Render("(type to filter, ↑/↓ to move, enter to select, esc to cancel)"))
	return b.String()
}

// splitDisplay splits a display string of the form "name<sep>path" at the
// two-space separator used by the two-column layout, returning the name and
// the path (possibly empty).
func splitDisplay(display string) (string, string) {
	if name, path, found := strings.Cut(display, "  "); found && name != "" {
		return name, path
	}
	return display, ""
}

// partitionMatchIndices splits display-relative match indices into
// name-relative and path-relative indices. The display has the form
// name + two-space separator + path; positions inside the separator are
// dropped.
func partitionMatchIndices(indices []int, nameLen int) (nameIdx, pathIdx []int) {
	pathOffset := nameLen + 2
	for _, i := range indices {
		switch {
		case i < nameLen:
			nameIdx = append(nameIdx, i)
		case i >= pathOffset:
			pathIdx = append(pathIdx, i-pathOffset)
		}
	}
	return nameIdx, pathIdx
}

// highlight wraps the runes at the given indices in s with the given style,
// grouping adjacent runes into single styled runs. Without indices or with
// a plain style, s is returned unchanged.
func highlight(s string, indices []int, style lipgloss.Style) string {
	if len(indices) == 0 {
		return s
	}

	runes := []rune(s)
	matched := make(map[int]bool, len(indices))
	for _, i := range indices {
		if i >= 0 && i < len(runes) {
			matched[i] = true
		}
	}

	var b strings.Builder
	for i := 0; i < len(runes); {
		j := i
		for j < len(runes) && matched[j] == matched[i] {
			j++
		}
		seg := string(runes[i:j])
		if matched[i] {
			b.WriteString(style.Render(seg))
		} else {
			b.WriteString(seg)
		}
		i = j
	}
	return b.String()
}

// editorModel is a Bubble Tea model that prompts for a free-form editor
// command with tab-completion over the known presets.
type editorModel struct {
	label    string
	input    textinput.Model
	presets  []string
	cursor   int
	keys     keyMap
	styles   listStyles
	choiceCh chan string
}

// newEditorModel builds the custom editor prompt.
func newEditorModel() editorModel {
	ti := textinput.New()
	ti.Placeholder = "e.g. code, vim, idea"
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(40)

	return editorModel{
		label:    "Enter the editor command",
		input:    ti,
		presets:  []string{"code", "atom", "subl", "vim", "idea"},
		keys:     newKeyMap(),
		styles:   newListStyles(),
		choiceCh: make(chan string, 1),
	}
}

func (m editorModel) Init() tea.Cmd { return textinput.Blink }

func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.cancel):
			m.choiceCh <- ""
			return m, tea.Quit
		case key.Matches(msg, m.keys.accept):
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				return m, nil
			}
			m.choiceCh <- value
			return m, tea.Quit
		case key.Matches(msg, m.keys.tab), key.Matches(msg, m.keys.shiftTab):
			if len(m.presets) == 0 {
				break
			}
			delta := 1
			if key.Matches(msg, m.keys.shiftTab) {
				delta = -1
			}
			m.cursor = (m.cursor + delta + len(m.presets)) % len(m.presets)
			m.input.SetValue(m.presets[m.cursor])
			m.input.CursorEnd()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m editorModel) View() tea.View {
	return tea.NewView(m.render())
}

func (m editorModel) render() string {
	var b strings.Builder

	b.WriteString(m.styles.prompt.Render(m.label))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")

	if m.input.Value() == "" {
		b.WriteString(m.styles.hint.Render(strings.Join(m.presets, " · ")))
		b.WriteString("\n")
	}

	b.WriteString(m.styles.hint.Render("(tab to cycle suggestions, enter to confirm, esc to cancel)"))
	return b.String()
}

// promptSelect shows a picker in the terminal and returns the index of the
// chosen entry in choices, or -1 if the user cancelled. When displays is nil
// the choices themselves are shown; otherwise it must be parallel to choices.
func promptSelect(label string, choices, displays []string) int {
	model := newSelectModel(label, choices, displays)
	p := tea.NewProgram(model)

	final, err := p.Run()
	if err != nil {
		helper.CheckError(err)
	}

	select {
	case pos := <-final.(selectModel).choiceCh:
		chosen := final.(selectModel)
		if pos < 0 || pos >= len(chosen.filtered) {
			return -1
		}
		return chosen.filtered[pos]
	default:
		// The program ended without a choice (e.g. ctrl+c was mapped to quit
		// directly). Treat it as a cancellation.
		return -1
	}
}

// promptEditor shows the custom editor prompt and returns the entered
// command, or an empty string if the user cancelled.
func promptEditor() string {
	model := newEditorModel()
	p := tea.NewProgram(model)

	final, err := p.Run()
	if err != nil {
		helper.CheckError(err)
	}

	select {
	case value := <-final.(editorModel).choiceCh:
		return value
	default:
		return ""
	}
}

// setupConfig creates the configuration file with defaults if it is missing
// and returns the configuration directory. The default global editor is taken
// from $EDITOR when set, falling back to "code".
func setupConfig() (string, error) {
	// Move a pre-existing config from the old location (the user config
	// directory root) into the projecto subdirectory, if needed.
	if _, err := helper.MigrateLegacyConfig(); err != nil {
		return "", err
	}

	configPath, err := helper.ConfigPath()
	if err != nil {
		return "", err
	}

	configDir := filepath.Dir(configPath)

	if helper.ConfigFileExists(configPath) {
		return configDir, nil
	}

	config := helper.Projecto{
		CommandToOpen: helper.DefaultEditor(),
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}

	helper.WriteConfigFile(config, configDir)
	return configDir, nil
}

// chooseEditor lets the user pick a preset or enter a custom editor command,
// returning the corresponding shell command. Returns false if cancelled.
func chooseEditor() (string, bool) {
	choices := make([]string, len(editorPresets))
	for i, preset := range editorPresets {
		choices[i] = preset.name
	}

	index := promptSelect("Select an editor for this project", choices, nil)
	if index < 0 {
		return "", false
	}

	if command := editorPresets[index].command; command != "" {
		return command, true
	}

	command := promptEditor()
	if command == "" {
		return "", false
	}
	return command, true
}

// projectNames extracts the names of all registered projects.
func projectNames(projects []helper.Project) []string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names
}

// recentOrder returns the indices of projects sorted most-recently-used
// first. Projects never opened (empty LastOpened) keep their registration
// order and come last, among themselves.
func recentOrder(projects []helper.Project) []int {
	indices := make([]int, len(projects))
	for i := range indices {
		indices[i] = i
	}

	sort.SliceStable(indices, func(a, b int) bool {
		// a and b are positions within indices, so the projects being
		// compared live at indices[a] and indices[b].
		pa, pb := projects[indices[a]], projects[indices[b]]
		ta, taErr := parseTimestamp(pa.LastOpened)
		tb, tbErr := parseTimestamp(pb.LastOpened)
		switch {
		case taErr == nil && tbErr == nil:
			return ta.After(tb)
		case taErr == nil:
			return true
		case tbErr == nil:
			return false
		default:
			return false // keep registration order
		}
	})
	return indices
}

// parseTimestamp parses a stored lastOpened value in RFC3339 format.
func parseTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

// touchProject stamps the project at the given index as just opened and
// persists the configuration.
func touchProject(configDir string, config helper.Projecto, index int) {
	config.Projects[index].LastOpened = time.Now().UTC().Format(time.RFC3339)
	helper.WriteConfigFile(config, configDir)
}

// selectProject shows an interactive list of projects (name and path
// columns, registration order) and returns the chosen index, or -1 if the
// user cancelled.
func selectProject(label string, projects []helper.Project) int {
	names := projectNames(projects)
	displays := make([]string, len(projects))
	for i, project := range projects {
		displays[i] = names[i] + "  " + helper.ShortenHome(project.Path)
	}
	return promptSelect(label, names, displays)
}

// selectProjectDetailed shows the interactive picker with a name column and
// a home-abbreviated path column, ordered most-recently-used first. It
// returns the index into projects, or -1 if the user cancelled.
func selectProjectDetailed(label string, projects []helper.Project) int {
	order := recentOrder(projects)
	names := projectNames(projects)
	choices := make([]string, len(order))
	displays := make([]string, len(order))
	for pos, index := range order {
		choices[pos] = names[index]
		displays[pos] = names[index] + "  " + helper.ShortenHome(projects[index].Path)
	}

	pos := promptSelect(label, choices, displays)
	if pos < 0 {
		return -1
	}
	return order[pos]
}

// editorFor returns the editor command for the project at the given index:
// the per-project editor when set, the global editor otherwise.
func editorFor(config helper.Projecto, index int) string {
	if editor := config.Projects[index].Editor; editor != "" {
		return editor
	}
	return config.CommandToOpen
}

// openProject launches the configured editor for the selected project. When
// verbose is true, the command being invoked is printed first.
func openProject(config helper.Projecto, index int, verbose bool) {
	project := config.Projects[index]
	editor := editorFor(config, index)

	if verbose {
		fmt.Println(helper.BLUE + "→ Invoking: " + editor + " " + project.Path + helper.RESET)
	}

	if err := exec.Command(editor, project.Path).Start(); err != nil {
		helper.CheckError(err)
	}
	fmt.Println(helper.GREEN + "✅ Opened " + project.Name + helper.RESET)
}

// openProjectFlow runs the full open flow: shows the picker ordered
// most-recently-used first and launches the chosen project.
func openProjectFlow(configDir string, verbose bool) {
	config := helper.ReadConfigFile(configDir)

	if len(config.Projects) == 0 {
		fmt.Println(helper.YELLOW + "No projects registered; add one with 'projecto add'" + helper.RESET)
		return
	}

	index := selectProjectDetailed("Available projects", config.Projects)
	if index < 0 {
		fmt.Println(helper.YELLOW + "Cancelled" + helper.RESET)
		return
	}

	openProject(config, index, verbose)
	touchProject(configDir, config, index)
}

// indexOfProjectPath returns the index of the first project registered with
// the given path, or -1 if none is. Both sides are cleaned so that cosmetic
// differences (e.g. a trailing slash) do not create duplicates.
func indexOfProjectPath(projects []helper.Project, path string) int {
	path = filepath.Clean(path)
	for i, project := range projects {
		if filepath.Clean(project.Path) == path {
			return i
		}
	}
	return -1
}

// addProject registers the current directory as a new project, optionally
// prompting for a per-project editor. Directories that are already
// registered are skipped with a notice.
func addProject(configDir string, withEditor bool) {
	config := helper.ReadConfigFile(configDir)

	path, name := helper.CurrentDir()

	if index := indexOfProjectPath(config.Projects, path); index >= 0 {
		fmt.Println(helper.YELLOW + "⚠️  Already registered as '" + config.Projects[index].Name + "'" + helper.RESET)
		return
	}

	var editor string

	if withEditor {
		chosen, ok := chooseEditor()
		if !ok {
			fmt.Println(helper.YELLOW + "Cancelled" + helper.RESET)
			return
		}
		editor = chosen
	}

	newProject := helper.Project{
		Path: path,
		Name: name,
	}
	if editor != "" {
		newProject.Editor = editor
	}

	config.Projects = append(config.Projects, newProject)
	helper.WriteConfigFile(config, configDir)

	fmt.Println(helper.GREEN + "✅ Successfully added" + helper.RESET)
}

// listProjects prints a non-interactive table of registered projects with
// their configured editors. Projects without their own editor show
// "(global)", meaning the fallback editor applies.
func listProjects(config helper.Projecto) {
	if len(config.Projects) == 0 {
		fmt.Println(helper.YELLOW + "No projects registered" + helper.RESET)
		return
	}

	// Determine column widths so the output is aligned regardless of name
	// and editor lengths.
	nameWidth := len("NAME")
	pathWidth := len("PATH")
	editorWidth := len("EDITOR")
	for _, project := range config.Projects {
		if w := len(project.Name); w > nameWidth {
			nameWidth = w
		}
		if w := len(project.Path); w > pathWidth {
			pathWidth = w
		}
		editor := project.Editor
		if editor == "" {
			editor = "(global)"
		}
		if w := len(editor); w > editorWidth {
			editorWidth = w
		}
	}

	header := fmt.Sprintf("%-*s  %-*s  %-*s", nameWidth, "NAME", pathWidth, "PATH", editorWidth, "EDITOR")
	fmt.Println(helper.BLUE + header + helper.RESET)

	for _, project := range config.Projects {
		editor := project.Editor
		if editor == "" {
			editor = "(global)"
		}
		fmt.Printf("%-*s  %-*s  %-*s\n", nameWidth, project.Name, pathWidth, project.Path, editorWidth, editor)
	}
}

// removeProject deletes the selected project from the configuration.
func removeProject(configDir string) {
	config := helper.ReadConfigFile(configDir)
	index := selectProject("Select a project to remove", config.Projects)
	if index < 0 {
		fmt.Println(helper.YELLOW + "Cancelled" + helper.RESET)
		return
	}

	config.Projects = append(config.Projects[:index], config.Projects[index+1:]...)
	helper.WriteConfigFile(config, configDir)

	fmt.Println(helper.GREEN + "❌ Successfully removed" + helper.RESET)
}

// removeProjectEditor clears the editor configured for the selected project.
func removeProjectEditor(configDir string) {
	config := helper.ReadConfigFile(configDir)
	index := selectProject("Select a project to remove its editor", config.Projects)
	if index < 0 {
		fmt.Println(helper.YELLOW + "Cancelled" + helper.RESET)
		return
	}
	config.Projects[index].Editor = ""
	helper.WriteConfigFile(config, configDir)

	fmt.Println(helper.GREEN + "❌ Successfully removed editor for the project" + helper.RESET)
}

// setGlobalEditor sets the fallback editor used for projects without a
// per-project editor.
func setGlobalEditor(configDir string, editor string) {
	config := helper.ReadConfigFile(configDir)
	config.CommandToOpen = editor
	helper.WriteConfigFile(config, configDir)

	fmt.Println(helper.GREEN + "✅ Successfully updated editor" + helper.RESET)
}

// loadConfigDir runs setupConfig and terminates the program on failure.
func loadConfigDir() string {
	configDir, err := setupConfig()
	helper.CheckError(err)
	return configDir
}

func newApp() *cli.Command {
	return &cli.Command{
		Name:                  "projecto",
		Usage:                 "Launch your projects in your editor of choice",
		Version:               version,
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add the current directory as a project",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "editor",
						Aliases: []string{"e"},
						Usage:   "Choose a dedicated editor for this project",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					addProject(loadConfigDir(), cmd.Bool("editor"))
					return nil
				},
			},
			{
				Name:    "open",
				Aliases: []string{"o"},
				Usage:   "Open a project from an interactive list",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show the editor command being invoked",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					openProjectFlow(loadConfigDir(), cmd.Bool("verbose"))
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List registered projects without opening one",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					listProjects(helper.ReadConfigFile(loadConfigDir()))
					return nil
				},
			},
			{
				Name:    "rm",
				Aliases: []string{"remove"},
				Usage:   "Remove a project",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					removeProject(loadConfigDir())
					return nil
				},
			},
			{
				Name:  "seteditor",
				Usage: "Set the global editor command used for projects without their own editor (defaults to $EDITOR)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					editor := cmd.Args().First()
					if editor == "" {
						editor = helper.DefaultEditor()
					}
					setGlobalEditor(loadConfigDir(), editor)
					return nil
				},
			},
			{
				Name:  "rmeditor",
				Usage: "Remove the editor override from a project",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					removeProjectEditor(loadConfigDir())
					return nil
				},
			},
			{
				Name:  "edit",
				Usage: "Open the configuration file in the default application",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					loadConfigDir()
					helper.OpenConfigFile()
					return nil
				},
			},
		},
	}
}

// version is overridden at build time by GoReleaser.
var version = "dev"

func main() {
	args := os.Args[1:]
	// Bare `projecto` (no subcommand) opens the picker directly.
	if len(args) == 0 {
		args = []string{"open"}
	}

	if err := newApp().Run(context.Background(), append([]string{os.Args[0]}, args...)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
