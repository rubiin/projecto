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

func configFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type project struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Editor string `json:"editor,omitempty"`
}

type projecto struct {
	CommandToOpen string    `json:"commandToOpen"`
	Projects      []project `json:"projects"`
}

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

func currentDir() []string {
	path, err := os.Getwd()
	check(err)
	pathArr := strings.Split(path, "/")
	return []string{path, pathArr[len(pathArr)-1]}
}

func main() {
	homeDir, err := os.UserHomeDir()

	check(err)

	if !configFileExists(homeDir + "/projecto.json") {

		file, err := os.Create(homeDir + "/projecto.json")
		check(err)
		file.Close()
	}

	add := flag.Bool("add", false, "Add a project")
	remove := flag.Bool("remove", false, "Remove a project")

	flag.Parse()

	if len(os.Args) == 1 {
		projects := readConfigFile(homeDir)
		var names []string
		for _, element := range projects.Projects {
			names = append(names, element.Name)
		}
		fmt.Println(names)

		list := promptui.Select{
			Label: "Available projects",
			Items: names,
		}
		idx, _, err := list.Run()
		check(err)

		err = exec.Command("code", projects.Projects[idx].Path).Start()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(projects.Projects[idx].Path)

		return
	}

	if *add {

		configFromFile := readConfigFile(homeDir)

		configFromFile.Projects = append(configFromFile.Projects, project{
			Path: currentDir()[0],
			Name: currentDir()[1],
		})

		writeConfigFile(configFromFile, homeDir)
		fmt.Println("Sucessfully added")

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
		fmt.Println("Sucessfully removed")

		return

	}

	// flag.Parse()

}
