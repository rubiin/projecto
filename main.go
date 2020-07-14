package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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

func writeConfigFile(content []byte, homeDir string) {
	f, err := os.Create(homeDir + "/projecto.json")
	defer f.Close()
	check(err)

	_, e := f.WriteString(string(content))
	check(e)
	fmt.Println("Sucessfully added")

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

	arg := os.Args[1]

	if arg == "add" {

		configFromFile := readConfigFile(homeDir)

		configFromFile.Projects = append(configFromFile.Projects, project{
			Name: currentDir()[0],
			Path: currentDir()[1],
		})

		fmt.Println(configFromFile)

		bs, err := json.MarshalIndent(configFromFile, "", "\t")
		check(err)

		writeConfigFile(bs, homeDir)

	}

	// flag.Parse()

}
