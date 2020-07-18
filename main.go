package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/rubiin/projecto/helper"
)

func main() {
	homeDir, err := os.UserHomeDir()

	helper.CheckError(err)

	if !helper.ConfigFileExists(homeDir + "/projecto.json") {

		file, err := os.Create(homeDir + "/projecto.json")
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

	flag.Parse()

	if helper.IsFlagPassed("seteditor") {
		projects := helper.ReadConfigFile(homeDir)
		projects.CommandToOpen = *seteditor

		helper.WriteConfigFile(projects, homeDir)

		fmt.Println(helper.GREEN + "✅ Sucessfully updated editor" + helper.RESET)

	}

	if *open {
		projects := helper.ReadConfigFile(homeDir)
		var names []string
		for _, element := range projects.Projects {
			names = append(names, element.Name)
		}

		list := promptui.Select{
			Label: "Available projects",
			Items: names,
		}
		index, _, err := list.Run()
		helper.CheckError(err)

		editor := projects.Projects[index].Editor

		if projects.Projects[index].Editor == "" {

			editor = projects.CommandToOpen
		}

		err = exec.Command(editor, projects.Projects[index].Path).Start()
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	if *add {

		configFromFile := helper.ReadConfigFile(homeDir)

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
				fmt.Println("Enter" + helper.YELLOW + " command " + helper.RESET + "that you use to open Editor from Terminal")
				fmt.Scanf("%s", &cmd)
				newProject.Editor = cmd
			} else {
				newProject.Editor = strings.ToLower(editorsList[index])

			}

		}

		configFromFile.Projects = append(configFromFile.Projects, newProject)

		helper.WriteConfigFile(configFromFile, homeDir)

		fmt.Println(helper.GREEN + "✅ Sucessfully added" + helper.RESET)

		return

	}

	if *rmeditor {

		configFromFile := helper.ReadConfigFile(homeDir)
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
		helper.WriteConfigFile(configFromFile, homeDir)
		fmt.Println(helper.GREEN + "❌ Sucessfully removed for the project" + helper.RESET)

		return

	}

	if *remove {

		configFromFile := helper.ReadConfigFile(homeDir)

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

		helper.WriteConfigFile(configFromFile, homeDir)
		fmt.Println(helper.GREEN + "❌ Sucessfully removed" + helper.RESET)

		return

	}

}
