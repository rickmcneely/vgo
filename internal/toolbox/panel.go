package toolbox

import (
	"sort"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"vg/internal/designer/controls"
	"vg/internal/ui"
)

// Panel is the toolbox panel containing available controls
type Panel struct {
	composite         *walk.Composite
	registry          *controls.Registry
	listBox           *walk.ListBox
	controlTypes      []string
	onControlSelected func(string)
}

// NewPanel creates a new toolbox panel
func NewPanel(parent *walk.Composite, registry *controls.Registry) (*Panel, error) {
	p := &Panel{
		composite: parent,
		registry:  registry,
	}

	// Build flat list of all controls sorted by display name
	allControls := registry.All()
	sort.Slice(allControls, func(i, j int) bool {
		return allControls[i].DisplayName < allControls[j].DisplayName
	})

	names := make([]string, len(allControls))
	p.controlTypes = make([]string, len(allControls))
	for i, def := range allControls {
		names[i] = def.DisplayName
		p.controlTypes[i] = def.Type
	}

	var lb *walk.ListBox

	builder := NewBuilder(parent)

	if err := (Composite{
		Layout: VBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			// Styled header
			Composite{
				Layout:     HBox{Margins: Margins{Left: ui.HeaderMarginH, Top: ui.HeaderMarginV, Right: ui.HeaderMarginH, Bottom: ui.HeaderMarginV}},
				Background: SolidColorBrush{Color: ui.HeaderBackgroundColor},
				Children: []Widget{
					Label{
						Text:      "Toolbox",
						Font:      Font{Family: ui.HeaderFontFamily, PointSize: ui.HeaderFontSize, Bold: true},
						TextColor: ui.HeaderTextColor,
					},
					HSpacer{},
				},
			},
			// List box
			ListBox{
				AssignTo: &lb,
				Model:    names,
			},
		},
	}).Create(builder); err != nil {
		return nil, err
	}

	// Single-click to select control (VB6 style)
	lb.CurrentIndexChanged().Attach(func() {
		idx := lb.CurrentIndex()
		if idx >= 0 && idx < len(p.controlTypes) {
			if p.onControlSelected != nil {
				p.onControlSelected(p.controlTypes[idx])
			}
		}
	})

	p.listBox = lb
	return p, nil
}

// SetOnControlSelected sets the callback when a control is selected
func (p *Panel) SetOnControlSelected(fn func(string)) {
	p.onControlSelected = fn
}
