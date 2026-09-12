package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/rubiin/projecto/helper"
)

// configTemplate is the default configuration written when no configuration
// file exists yet.
const configTemplate = `{
	"commandToOpen": "code",
	"projects": []
}`

// editorChoices are the editor options offered when registering a project
// with a per-project editor. The last entry prompts for a custom command.
var editorChoices = []string{"Code", "Atom", "Sublime", "Other"}

// promptTemplate defines the shared promptui styling for text prompts.
var promptTemplate = &promptui.PromptTemplates{
	Prompt:  "{{ . }} ",
	Valid:   "{{ . | green }} ",
	Invalid: "{{ . | red }} ",
	Success: "{{ . | bold }} ",
}

// selectTemplate defines the promptui styling for project selection lists.
var selectTemplate = &promptui.SelectTemplates{
	Label:    "{{ . }}?",
	Active:   "\U0001F449 {{ .Name | cyan }}",
	Inactive: "   {{ .Name | cyan }}",
	Selected: "\U0001F449 {{ .Name | cyan }}",
}

// validateCustomEditor requires at least one letter so an empty custom
// editor command cannot be stored.
func validateCustomEditor(input string) error {
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("editor command cannot be empty")
	}
	return nil
}

// setupConfig creates the configuration file with defaults if it is missing
// and returns the configuration directory.
func setupConfig() (string, error) {
	configPath, err := helper.ConfigPath()
	if err != nil {
		return "", err
	}

	configDir := filepath.Dir(configPath)

	if helper.ConfigFileExists(configPath) {
		return configDir, nil
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(configPath, []byte(configTemplate), 0o644); err != nil {
		return "", err
	}
	return configDir, nil
}

// chooseEditor prompts the user to pick an editor for the project being
// added and returns the corresponding command.
func chooseEditor() string {
	list := promptui.Select{
		Label: "Select an editor for this project",
		Items: editorChoices,
	}
	index, _, err := list.Run()
	helper.CheckError(err)

	if index != len(editorChoices)-1 {
		return strings.ToLower(editorChoices[index])
	}

	prompt := promptui.Prompt{
		Label:     "Enter the editor command",
		Templates: promptTemplate,
		Validate:  validateCustomEditor,
	}
	command, err := prompt.Run()
	helper.CheckError(err)
	return command
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
// chosen index.
func selectProject(label string, projects []helper.Project) int {
	list := promptui.Select{
		Label: label,
		Items: projectNames(projects),
	}
	index, _, err := list.Run()
	helper.CheckError(err)
	return index
}

// openProject launches the configured editor for the selected project.
func openProject(config helper.Projecto, index int) {
	project := config.Projects[index]

	editor := project.Editor
	if editor == "" {
		editor = config.CommandToOpen
	}

	if err := exec.Command(editor, project.Path).Start(); err != nil {
		helper.CheckError(err)
	}
	fmt.Println(helper.GREEN + "✅ Opened " + project.Name + helper.RESET)
}

// addProject registers the current directory as a new project, optionally
// prompting for a per-project editor.
func addProject(configDir string, withEditor bool) {
	config := helper.ReadConfigFile(configDir)

	path, name := helper.CurrentDir()

	newProject := helper.Project{
		Path: path,
		Name: name,
	}

	if withEditor {
		newProject.Editor = chooseEditor()
	}

	config.Projects = append(config.Projects, newProject)
	helper.WriteConfigFile(config, configDir)

	fmt.Println(helper.GREEN + "✅ Successfully added" + helper.RESET)
}

// removeProject deletes the selected project from the configuration.
func removeProject(configDir string) {
	config := helper.ReadConfigFile(configDir)
	index := selectProject("Select a project to remove", config.Projects)

	config.Projects = append(config.Projects[:index], config.Projects[index+1:]...)
	helper.WriteConfigFile(config, configDir)

	fmt.Println(helper.GREEN + "❌ Successfully removed" + helper.RESET)
}

// removeProjectEditor clears the editor configured for the selected project.
func removeProjectEditor(configDir string) {
	config := helper.ReadConfigFile(configDir)
	index := selectProject("Select a project to remove its editor", config.Projects)

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

func main() {
	configDir, err := setupConfig()
	helper.CheckError(err)

	add := flag.Bool("add", false, "Add the current directory as a project")
	remove := flag.Bool("rm", false, "Remove a project")
	open := flag.Bool("open", false, "Open a project")
	seteditor := flag.String("seteditor", "code", "Set the global editor command used for projects without their own editor")
	editor := flag.Bool("editor", false, "Set an editor for this project (use with --add)")
	rmeditor := flag.Bool("rmeditor", false, "Remove the editor from a project")
	edit := flag.Bool("edit", false, "Open the config file in the default editor")

	flag.Parse()

	switch {
	case helper.IsFlagPassed("seteditor"):
		setGlobalEditor(configDir, *seteditor)
	case *edit:
		helper.OpenConfigFile()
	case *open:
		config := helper.ReadConfigFile(configDir)

		list := promptui.Select{
			Label:     "Available projects",
			Items:     config.Projects,
			Size:      8,
			Templates: selectTemplate,
		}
		index, _, err := list.Run()
		helper.CheckError(err)

		openProject(config, index)
	case *add:
		addProject(configDir, *editor)
	case *rmeditor:
		removeProjectEditor(configDir)
	case *remove:
		removeProject(configDir)
	}
}
