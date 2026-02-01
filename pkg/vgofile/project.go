package vgofile

import (
	"encoding/json"
	"fmt"
	"os"
)

// Project represents a VGO project
type Project struct {
	Name     string  `json:"name"`
	Forms    []*Form `json:"forms,omitempty"`
	FilePath string  `json:"-"` // Not serialized
}

// NewProject creates a new project with the given name
func NewProject(name string) *Project {
	return &Project{
		Name:  name,
		Forms: make([]*Form, 0),
	}
}

// AddForm adds a form to the project
func (p *Project) AddForm(form *Form) {
	p.Forms = append(p.Forms, form)
}

// RemoveForm removes a form from the project
func (p *Project) RemoveForm(form *Form) {
	for i, f := range p.Forms {
		if f == form {
			p.Forms = append(p.Forms[:i], p.Forms[i+1:]...)
			return
		}
	}
}

// FindForm finds a form by name
func (p *Project) FindForm(name string) *Form {
	for _, f := range p.Forms {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// Save saves the project to its file path
func (p *Project) Save() error {
	if p.FilePath == "" {
		return fmt.Errorf("no file path set")
	}
	return p.SaveTo(p.FilePath)
}

// SaveTo saves the project to the specified path
func (p *Project) SaveTo(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling project: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing project file: %w", err)
	}
	p.FilePath = path
	return nil
}

// LoadProject loads a project from the specified path
func LoadProject(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading project file: %w", err)
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshaling project: %w", err)
	}
	p.FilePath = path
	return &p, nil
}

// String returns a string representation of the project
func (p *Project) String() string {
	return fmt.Sprintf("%s with %d forms", p.Name, len(p.Forms))
}
