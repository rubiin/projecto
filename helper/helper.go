/*

Package math provides basic constants and mathematical functions.

*/

package helper

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
)

func ConfigFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func CurrentDir() []string {
	path, err := os.Getwd()
	CheckError(err)
	pathArr := strings.Split(path, "/")
	return []string{path, pathArr[len(pathArr)-1]}
}

func CheckError(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func IsFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func ReadConfigFile(homeDir string) Projecto {
	var config string

	f, err := os.OpenFile(homeDir+"/projecto.json", os.O_RDONLY, 0644)
	CheckError(err)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		config += scanner.Text()
	}

	var configFromFile Projecto
	err = json.Unmarshal([]byte(config), &configFromFile)
	CheckError(err)
	return configFromFile

}

func WriteConfigFile(configFromFile Projecto, homeDir string) {

	bs, err := json.MarshalIndent(configFromFile, "", "\t")
	CheckError(err)
	f, err := os.Create(homeDir + "/projecto.json")
	defer f.Close()
	CheckError(err)

	_, e := f.WriteString(string(bs))
	CheckError(e)

}
