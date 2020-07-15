package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/manifoldco/promptui"
)

type project struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Editor string `json:"editor,omitempty"`
}

type projecto struct {
	CommandToOpen string    `json:"commandToOpen"`
	Projects      []project `json:"projects"`
}

var red = "\033[31m"
var green = "\033[32m"
var yellow = "\033[33m"
var blue = "\033[34m"
var reset = "\033[0m"

func writeConfigFile(configFromFile projecto, homeDir string) {

	bs, err := json.MarshalIndent(configFromFile, "", "\t")
	check(err)
	f, err := os.Create(homeDir + "/projecto.json")
	defer f.Close()
	check(err)

	_, e := f.WriteString(string(bs))
	check(e)

}

func readConfigFile(homeDir string) projecto {
	var config string

	f, err := os.OpenFile(homeDir+"/projecto.json", os.O_RDONLY, 0644)
	check(err)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		config += scanner.Text()
	}

	check(err)
	var configFromFile projecto

	err = json.Unmarshal([]byte(config), &configFromFile)
	check(err)
	return configFromFile

}

func check(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func currentDir() []string {
	path, err := os.Getwd()
	check(err)
	pathArr := strings.Split(path, "/")
	return []string{path, pathArr[len(pathArr)-1]}
}

func configFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func main() {
	homeDir, err := os.UserHomeDir()

	check(err)

	if !configFileExists(homeDir + "/projecto.json") {

		file, err := os.Create(homeDir + "/projecto.json")
		check(err)
		file.WriteString(`{
				"commandToOpen": "code",
				"projects": []
					}`)
		file.Close()
	}

	add := flag.Bool("add", false, "Add a project")
	remove := flag.Bool("remove", false, "Remove a project")
	open := flag.Bool("open", false, "Open a project")
	seteditor := flag.String("seteditor", "code", "Sets default editor for project")
	editor := flag.Bool("editor", false, "Sets editor for this project")

	flag.Parse()

	if isFlagPassed("seteditor") {
		projects := readConfigFile(homeDir)
		projects.CommandToOpen = *seteditor

		writeConfigFile(projects, homeDir)

		fmt.Println(green + "✅ Sucessfully updated editor" + reset)

	}

	if *open {
		projects := readConfigFile(homeDir)
		var names []string
		for _, element := range projects.Projects {
			names = append(names, element.Name)
		}

		list := promptui.Select{
			Label: "Available projects",
			Items: names,
		}
		idx, _, err := list.Run()
		check(err)

		err = exec.Command(projects.Projects[idx].Editor, projects.Projects[idx].Path).Start()
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	if *add {

		configFromFile := readConfigFile(homeDir)

		newProject := project{
			Path: currentDir()[0],
			Name: currentDir()[1],
		}

		if *editor {
			editorsList := []string{"Code", "Atom", "Sublime", "Other"}

			list := promptui.Select{
				Label: "Select an editor for this project",
				Items: editorsList,
			}
			index, _, err := list.Run()
			check(err)

			if index == 3 {
				var cmd string
				fmt.Println("Enter" + yellow + " command " + reset + "that you use to open Editor from Terminal")
				fmt.Scanf("%s", &cmd)
				newProject.Editor = cmd
			} else {
				newProject.Editor = strings.ToLower(editorsList[index])

			}

		}

		configFromFile.Projects = append(configFromFile.Projects, newProject)

		writeConfigFile(configFromFile, homeDir)

		fmt.Println(green + "✅ Sucessfully added" + reset)

		return

	}

	if *remove {

		configFromFile := readConfigFile(homeDir)

		var names []string
		for _, element := range configFromFile.Projects {
			names = append(names, element.Name)
		}

		list := promptui.Select{
			Label: "Available projects",
			Items: names,
		}
		index, _, err := list.Run()
		check(err)
		configFromFile.Projects = append(configFromFile.Projects[:index], configFromFile.Projects[index+1:]...)

		writeConfigFile(configFromFile, homeDir)
		fmt.Println(green + "❌ Sucessfully removed" + green)

		return

	}

}
