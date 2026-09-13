package helper

// Project represents a single registered project directory.
type Project struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Editor     string `json:"editor,omitempty"`
	LastOpened string `json:"lastOpened,omitempty"`
}

// Projecto is the top-level configuration structure persisted to
// projecto.json.
type Projecto struct {
	CommandToOpen string    `json:"commandToOpen"`
	Projects      []Project `json:"projects"`
}
