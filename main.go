package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/rubiin/projecto/helper"
)

func main() {

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	validate := func(input string) error {
		_, err := regexp.Match(`[a-z]`, []byte(input))
		return err
	}

	configDir, err := os.UserConfigDir()

	helper.CheckError(err)

	if !helper.ConfigFileExists(configDir + "/projecto.json") {

		file, err := os.Create(configDir + "/projecto.json")
		helper.CheckError(err)
		file.WriteString(`{
				"commandToOpen": "code",
				"projects": []
					}`)
		file.Close()
	}

	add := flag.Bool("add", false, "Add a project")
	remove := flag.Bool("rm", false, "Remove a project")
	open := flag.Bool("open", false, "Open a project")
	seteditor := flag.String("seteditor", "code", "Sets global editor for project.This is used for projects where editor is not set")
	editor := flag.Bool("editor", false, "Sets an editor for this project.Should be used along with --add")
	rmeditor := flag.Bool("rmeditor", false, "Removes editor for from the project")
	edit := flag.Bool("edit", false, "Opens the config file in default editor")

	flag.Parse()

	if helper.IsFlagPassed("seteditor") {
		projects := helper.ReadConfigFile(configDir)
		projects.CommandToOpen = *seteditor

		helper.WriteConfigFile(projects, configDir)

		fmt.Println(helper.GREEN + "✅ Successfully updated editor" + helper.RESET)

	}

	if *edit {
		helper.OpenConfigFile()
		return
	}

	if *open {

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "\U0001f449{{ .Name | cyan }} ",
			Inactive: "  {{ .Name | cyan }} ",
			Selected: "\U0001f449{{ .Name | red | cyan }}",
		}

		projects := helper.ReadConfigFile(configDir)

		list := promptui.Select{
			Label:     "Available projects",
			Items:     projects.Projects,
			Size:      8,
			Templates: templates,
		}
		index, _, err := list.Run()
		helper.CheckError(err)

		editor := projects.Projects[index].Editor

		if projects.Projects[index].Editor == "" {

			editor = projects.CommandToOpen
		}

		err = exec.Command(editor, projects.Projects[index].Path).Start()
		helper.CheckError(err)

		return
	}

	if *add {

		configFromFile := helper.ReadConfigFile(configDir)

		newProject := helper.Project{
			Path: helper.CurrentDir()[0],
			Name: helper.CurrentDir()[1],
		}

		if *editor {
			editorsList := []string{"Code", "Atom", "Sublime", "Other"}

			list := promptui.Select{
				Label: "Select a global editor",
				Items: editorsList,
			}
			index, _, err := list.Run()
			helper.CheckError(err)

			if index == 3 {
				var cmd string
				prompt := promptui.Prompt{
					Label:     "Spicy Level",
					Templates: templates,
					Validate:  validate,
				}
				cmd, err := prompt.Run()
				helper.CheckError(err)
				newProject.Editor = cmd
			} else {
				newProject.Editor = strings.ToLower(editorsList[index])

			}

		}

		configFromFile.Projects = append(configFromFile.Projects, newProject)

		helper.WriteConfigFile(configFromFile, configDir)

		fmt.Println(helper.GREEN + "✅ Sucessfully added" + helper.RESET)

		return

	}

	if *rmeditor {

		configFromFile := helper.ReadConfigFile(configDir)
		var names []string
		for _, element := range configFromFile.Projects {
			names = append(names, element.Name)
		}
		list := promptui.Select{
			Label: "Available projects",
			Items: names,
		}
		index, _, err := list.Run()
		helper.CheckError(err)
		configFromFile.Projects[index].Editor = ""
		helper.WriteConfigFile(configFromFile, configDir)
		fmt.Println(helper.GREEN + "❌ Successfully removed for the project" + helper.RESET)

		return

	}

	if *remove {

		configFromFile := helper.ReadConfigFile(configDir)

		var names []string
		for _, element := range configFromFile.Projects {
			names = append(names, element.Name)
		}

		list := promptui.Select{
			Label: "Available projects",
			Items: names,
		}
		index, _, err := list.Run()
		helper.CheckError(err)
		configFromFile.Projects = append(configFromFile.Projects[:index], configFromFile.Projects[index+1:]...)

		helper.WriteConfigFile(configFromFile, configDir)
		fmt.Println(helper.GREEN + "❌ Successfully removed" + helper.RESET)

		return

	}

}
