package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	title    lipgloss.Style
	item     lipgloss.Style
	selected lipgloss.Style
	prompt   lipgloss.Style
	hint     lipgloss.Style
}

// newListStyles returns the picker styles.
func newListStyles() listStyles {
	return listStyles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		item:     lipgloss.NewStyle().PaddingLeft(2),
		selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).PaddingLeft(0),
		prompt:   lipgloss.NewStyle().Bold(true).MarginBottom(1),
		hint:     lipgloss.NewStyle().Faint(true).MarginTop(1),
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
		up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		accept:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		cancel:   key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
		tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next editor")),
		shiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev editor")),
	}
}

// selectModel is a Bubble Tea model that shows a list of strings and returns
// the chosen index.
type selectModel struct {
	label    string
	choices  []string
	cursor   int
	keys     keyMap
	styles   listStyles
	choiceCh chan string
}

// newSelectModel builds a picker for the given choices.
func newSelectModel(label string, choices []string) selectModel {
	return selectModel{
		label:    label,
		choices:  choices,
		keys:     newKeyMap(),
		styles:   newListStyles(),
		choiceCh: make(chan string, 1),
	}
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.cancel):
			m.choiceCh <- ""
			return m, tea.Quit
		case key.Matches(msg, m.keys.accept):
			m.choiceCh <- m.choices[m.cursor]
			return m, tea.Quit
		case key.Matches(msg, m.keys.up):
			m.cursor = (m.cursor - 1 + len(m.choices)) % len(m.choices)
		case key.Matches(msg, m.keys.down):
			m.cursor = (m.cursor + 1) % len(m.choices)
		}
	}
	return m, nil
}

func (m selectModel) View() tea.View {
	return tea.NewView(m.render())
}

func (m selectModel) render() string {
	var b strings.Builder

	b.WriteString(m.styles.prompt.Render(m.label))
	b.WriteString("\n")

	for i, choice := range m.choices {
		style := m.styles.item
		if i == m.cursor {
			style = m.styles.selected
			b.WriteString("❯ ")
		} else {
			b.WriteString("  ")
		}
		b.WriteString(style.Render(choice))
		b.WriteString("\n")
	}

	b.WriteString(m.styles.hint.Render("(↑/↓ to move, enter to select, esc to cancel)"))
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

// promptSelect shows a picker in the terminal and returns the chosen index,
// or -1 if the user cancelled.
func promptSelect(label string, choices []string) int {
	model := newSelectModel(label, choices)
	p := tea.NewProgram(model)

	final, err := p.Run()
	if err != nil {
		helper.CheckError(err)
	}

	choice := final.(selectModel).choiceCh
	select {
	case c := <-choice:
		for i, candidate := range choices {
			if candidate == c {
				return i
			}
		}
		return -1
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

	index := promptSelect("Select an editor for this project", choices)
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

// selectProject shows an interactive list of project names and returns the
// chosen index, or -1 if the user cancelled.
func selectProject(label string, projects []helper.Project) int {
	return promptSelect(label, projectNames(projects))
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

// addProject registers the current directory as a new project, optionally
// prompting for a per-project editor.
func addProject(configDir string, withEditor bool) {
	var editor string

	if withEditor {
		chosen, ok := chooseEditor()
		if !ok {
			fmt.Println(helper.YELLOW + "Cancelled" + helper.RESET)
			return
		}
		editor = chosen
	}

	config := helper.ReadConfigFile(configDir)

	path, name := helper.CurrentDir()

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
					configDir := loadConfigDir()
					config := helper.ReadConfigFile(configDir)

					if len(config.Projects) == 0 {
						return fmt.Errorf("no projects registered; add one with 'projecto add'")
					}

					index := promptSelect("Available projects", projectNames(config.Projects))
					if index < 0 {
						fmt.Println(helper.YELLOW + "Cancelled" + helper.RESET)
						return nil
					}

					openProject(config, index, cmd.Bool("verbose"))
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
	if err := newApp().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
