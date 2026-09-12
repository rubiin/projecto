package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/rubiin/projecto/helper"
)

func TestConfigTemplateIsValidJSON(t *testing.T) {
	var config struct {
		CommandToOpen string   `json:"commandToOpen"`
		Projects      []string `json:"projects"`
	}
	if err := json.Unmarshal([]byte(configTemplate), &config); err != nil {
		t.Errorf("configTemplate is not valid JSON: %v", err)
	}
	if config.CommandToOpen != "code" {
		t.Errorf("configTemplate default editor = %q, want %q", config.CommandToOpen, "code")
	}
	if config.Projects == nil || len(config.Projects) != 0 {
		t.Error("configTemplate projects should be an empty list")
	}
}

func TestValidateCustomEditor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple command", "code", false},
		{"command with args", "code --wait", false},
		{"empty input", "", true},
		{"whitespace only", "   ", true},
		{"tab only", "\t", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomEditor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCustomEditor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestProjectNames(t *testing.T) {
	config := helper.Projecto{
		Projects: []helper.Project{
			{Name: "alpha", Path: "/home/user/alpha"},
			{Name: "beta", Path: "/home/user/beta"},
		},
	}
	if names := projectNames(config.Projects); !reflect.DeepEqual(names, []string{"alpha", "beta"}) {
		t.Errorf("projectNames() = %v, want [alpha beta]", names)
	}

	if names := projectNames(nil); len(names) != 0 {
		t.Errorf("projectNames(nil) = %v, want empty", names)
	}
}

func TestEditorChoicesIncludeCustomOption(t *testing.T) {
	last := editorChoices[len(editorChoices)-1]
	if !strings.EqualFold(last, "Other") {
		t.Errorf("last editor choice = %q, want an %q option", last, "Other")
	}
}
