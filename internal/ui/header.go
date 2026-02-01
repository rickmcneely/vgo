package ui

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// Header styling constants - change these to update all headers consistently
var (
	HeaderBackgroundColor = walk.RGB(120, 120, 120) // Light grey
	HeaderTextColor       = walk.RGB(255, 255, 255) // White
	HeaderFontFamily      = "Segoe UI"
	HeaderFontSize        = 9
	HeaderMarginH         = 8 // Horizontal margin (left/right)
	HeaderMarginV         = 4 // Vertical margin (top/bottom)
)

// CreateHeader creates a consistently styled header composite with a label
// Uses declarative API to ensure proper stretching behavior
// Returns the header composite and label, or an error
func CreateHeader(parent walk.Container, title string) (*walk.Composite, *walk.Label, error) {
	var headerComposite *walk.Composite
	var label *walk.Label

	builder := NewBuilder(parent)

	if err := (Composite{
		AssignTo:   &headerComposite,
		Layout:     HBox{Margins: Margins{Left: HeaderMarginH, Top: HeaderMarginV, Right: HeaderMarginH, Bottom: HeaderMarginV}},
		Background: SolidColorBrush{Color: HeaderBackgroundColor},
		Children: []Widget{
			Label{
				AssignTo:  &label,
				Text:      title,
				Font:      Font{Family: HeaderFontFamily, PointSize: HeaderFontSize, Bold: true},
				TextColor: HeaderTextColor,
			},
		},
	}).Create(builder); err != nil {
		return nil, nil, err
	}

	return headerComposite, label, nil
}

// SetupPanelLayout ensures the parent composite has the correct layout for panels
// Call this before adding any children if the layout isn't already set up
func SetupPanelLayout(parent *walk.Composite) {
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 0, VNear: 0, HFar: 0, VFar: 0})
	layout.SetSpacing(0)
	parent.SetLayout(layout)
}
