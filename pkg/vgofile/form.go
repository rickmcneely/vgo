package vgofile

import (
	"fmt"
	"strings"
)

// Form represents a visual form/window
type Form struct {
	Name        string     `json:"name"`
	Text        string     `json:"text"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	MinimizeBox bool       `json:"minimizeBox"`
	MaximizeBox bool       `json:"maximizeBox"`
	ControlBox  bool       `json:"controlBox"`
	Resizable   bool       `json:"resizable"`
	StartPos    string     `json:"startPos,omitempty"` // CenterScreen, CenterParent, Manual
	Controls    []*Control `json:"controls,omitempty"`
}

// NewForm creates a new form with default settings
func NewForm(name string) *Form {
	return &Form{
		Name:        name,
		Text:        name,
		Width:       400,
		Height:      300,
		MinimizeBox: true,
		MaximizeBox: true,
		ControlBox:  true,
		Resizable:   true,
		StartPos:    "CenterScreen",
		Controls:    make([]*Control, 0),
	}
}

// AddControl adds a control to the form
func (f *Form) AddControl(ctrl *Control) {
	f.Controls = append(f.Controls, ctrl)
}

// RemoveControl removes a control from the form
func (f *Form) RemoveControl(ctrl *Control) {
	for i, c := range f.Controls {
		if c == ctrl {
			f.Controls = append(f.Controls[:i], f.Controls[i+1:]...)
			return
		}
	}
}

// FindControl finds a control by name
func (f *Form) FindControl(name string) *Control {
	for _, c := range f.Controls {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// GenerateControlName generates a unique name for a new control of the given type
func (f *Form) GenerateControlName(controlType string) string {
	// Count existing controls of this type
	count := 0
	prefix := strings.ToLower(controlType)
	for _, c := range f.Controls {
		if strings.HasPrefix(strings.ToLower(c.Name), prefix) {
			count++
		}
	}
	return fmt.Sprintf("%s%d", controlType, count+1)
}

// Clone creates a deep copy of the form
func (f *Form) Clone() *Form {
	clone := &Form{
		Name:        f.Name,
		Text:        f.Text,
		Width:       f.Width,
		Height:      f.Height,
		MinimizeBox: f.MinimizeBox,
		MaximizeBox: f.MaximizeBox,
		ControlBox:  f.ControlBox,
		Resizable:   f.Resizable,
		StartPos:    f.StartPos,
		Controls:    make([]*Control, len(f.Controls)),
	}
	for i, c := range f.Controls {
		clone.Controls[i] = c.Clone()
	}
	return clone
}

// String returns a string representation of the form
func (f *Form) String() string {
	return fmt.Sprintf("%s (%dx%d) with %d controls", f.Name, f.Width, f.Height, len(f.Controls))
}
