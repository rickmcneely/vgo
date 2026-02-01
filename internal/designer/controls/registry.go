package controls

import (
	"fmt"
	"image"
	"sort"
)

// PropertyType defines the type of a property
type PropertyType int

const (
	PropString PropertyType = iota
	PropInt
	PropBool
	PropColor
	PropFont
	PropEnum
)

// PropertyDefinition defines a control property
type PropertyDefinition struct {
	Name         string
	Type         PropertyType
	DefaultValue interface{}
	EnumValues   []string // For PropEnum type
}

// ControlDefinition defines a control type
type ControlDefinition struct {
	Type        string
	DisplayName string
	Category    string
	DefaultSize image.Point
	Properties  []PropertyDefinition
}

// Registry holds all available control definitions
type Registry struct {
	controls   map[string]*ControlDefinition
	categories map[string][]*ControlDefinition
}

// NewRegistry creates a new control registry with standard controls
func NewRegistry() *Registry {
	r := &Registry{
		controls:   make(map[string]*ControlDefinition),
		categories: make(map[string][]*ControlDefinition),
	}
	r.registerStandardControls()
	return r
}

// Register adds a control definition to the registry
func (r *Registry) Register(def *ControlDefinition) {
	r.controls[def.Type] = def
	r.categories[def.Category] = append(r.categories[def.Category], def)
}

// Get returns a control definition by type
func (r *Registry) Get(controlType string) (*ControlDefinition, error) {
	if def, ok := r.controls[controlType]; ok {
		return def, nil
	}
	return nil, fmt.Errorf("unknown control type: %s", controlType)
}

// Categories returns a sorted list of category names
func (r *Registry) Categories() []string {
	cats := make([]string, 0, len(r.categories))
	for cat := range r.categories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}

// GetByCategory returns all control definitions in a category
func (r *Registry) GetByCategory(category string) []*ControlDefinition {
	return r.categories[category]
}

// All returns all control definitions
func (r *Registry) All() []*ControlDefinition {
	defs := make([]*ControlDefinition, 0, len(r.controls))
	for _, def := range r.controls {
		defs = append(defs, def)
	}
	return defs
}

func (r *Registry) registerStandardControls() {
	// Common Controls
	r.Register(&ControlDefinition{
		Type:        "Button",
		DisplayName: "Button",
		Category:    "Common Controls",
		DefaultSize: image.Point{X: 80, Y: 25},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: "Button"},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "Label",
		DisplayName: "Label",
		Category:    "Common Controls",
		DefaultSize: image.Point{X: 100, Y: 20},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: "Label"},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "TextBox",
		DisplayName: "TextBox",
		Category:    "Common Controls",
		DefaultSize: image.Point{X: 120, Y: 23},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: ""},
			{Name: "MaxLength", Type: PropInt, DefaultValue: 0},
			{Name: "ReadOnly", Type: PropBool, DefaultValue: false},
			{Name: "PasswordMode", Type: PropBool, DefaultValue: false},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "CheckBox",
		DisplayName: "CheckBox",
		Category:    "Common Controls",
		DefaultSize: image.Point{X: 100, Y: 20},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: "CheckBox"},
			{Name: "Checked", Type: PropBool, DefaultValue: false},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "RadioButton",
		DisplayName: "RadioButton",
		Category:    "Common Controls",
		DefaultSize: image.Point{X: 100, Y: 20},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: "RadioButton"},
			{Name: "Checked", Type: PropBool, DefaultValue: false},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "ComboBox",
		DisplayName: "ComboBox",
		Category:    "Common Controls",
		DefaultSize: image.Point{X: 120, Y: 23},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: ""},
			{Name: "Editable", Type: PropBool, DefaultValue: true},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	// List Controls
	r.Register(&ControlDefinition{
		Type:        "ListBox",
		DisplayName: "ListBox",
		Category:    "List Controls",
		DefaultSize: image.Point{X: 120, Y: 100},
		Properties: []PropertyDefinition{
			{Name: "MultiSelect", Type: PropBool, DefaultValue: false},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "TreeView",
		DisplayName: "TreeView",
		Category:    "List Controls",
		DefaultSize: image.Point{X: 150, Y: 150},
		Properties: []PropertyDefinition{
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	// Containers
	r.Register(&ControlDefinition{
		Type:        "GroupBox",
		DisplayName: "GroupBox",
		Category:    "Containers",
		DefaultSize: image.Point{X: 200, Y: 150},
		Properties: []PropertyDefinition{
			{Name: "Text", Type: PropString, DefaultValue: "GroupBox"},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "Panel",
		DisplayName: "Panel",
		Category:    "Containers",
		DefaultSize: image.Point{X: 200, Y: 150},
		Properties: []PropertyDefinition{
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	// Display Controls
	r.Register(&ControlDefinition{
		Type:        "PictureBox",
		DisplayName: "PictureBox",
		Category:    "Display",
		DefaultSize: image.Point{X: 100, Y: 100},
		Properties: []PropertyDefinition{
			{Name: "ImagePath", Type: PropString, DefaultValue: ""},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	r.Register(&ControlDefinition{
		Type:        "ProgressBar",
		DisplayName: "ProgressBar",
		Category:    "Display",
		DefaultSize: image.Point{X: 150, Y: 20},
		Properties: []PropertyDefinition{
			{Name: "Value", Type: PropInt, DefaultValue: 0},
			{Name: "Minimum", Type: PropInt, DefaultValue: 0},
			{Name: "Maximum", Type: PropInt, DefaultValue: 100},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})

	// Input Controls
	r.Register(&ControlDefinition{
		Type:        "Slider",
		DisplayName: "Slider",
		Category:    "Input Controls",
		DefaultSize: image.Point{X: 150, Y: 25},
		Properties: []PropertyDefinition{
			{Name: "Value", Type: PropInt, DefaultValue: 0},
			{Name: "Minimum", Type: PropInt, DefaultValue: 0},
			{Name: "Maximum", Type: PropInt, DefaultValue: 100},
			{Name: "Orientation", Type: PropEnum, DefaultValue: "Horizontal", EnumValues: []string{"Horizontal", "Vertical"}},
			{Name: "Enabled", Type: PropBool, DefaultValue: true},
			{Name: "Visible", Type: PropBool, DefaultValue: true},
		},
	})
}
