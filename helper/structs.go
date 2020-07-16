/*

Package math provides basic constants and mathematical functions.

*/

package helper



type Project struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Editor string `json:"editor,omitempty"`
}

type Projecto struct {
	CommandToOpen string    `json:"commandToOpen"`
	Projects      []Project `json:"projects"`
}