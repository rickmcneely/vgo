package properties

import (
	"fmt"
	"log"
	"sort"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"vg/internal/ui"
	"vg/pkg/vgofile"
)

// Panel is the properties panel for editing control properties
type Panel struct {
	composite             *walk.Composite
	tableView             *walk.TableView
	model                 *PropertyModel
	currentControl        *vgofile.Control
	currentForm           *vgofile.Form
	onPropertyChanged     func(*vgofile.Control, string, interface{})
	onFormPropertyChanged func(*vgofile.Form, string, interface{})
	updating              bool
}

// NewPanel creates a new properties panel
func NewPanel(parent *walk.Composite) (*Panel, error) {
	p := &Panel{
		composite: parent,
		model:     NewPropertyModel(),
	}

	var tv *walk.TableView

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
						Text:      "Properties",
						Font:      Font{Family: ui.HeaderFontFamily, PointSize: ui.HeaderFontSize, Bold: true},
						TextColor: ui.HeaderTextColor,
					},
					HSpacer{},
				},
			},
			// Table view
			TableView{
				AssignTo:            &tv,
				AlternatingRowBG:    true,
				ColumnsOrderable:    false,
				LastColumnStretched: true,
				Columns: []TableViewColumn{
					{Title: "Property", Width: 80},
					{Title: "Value", Width: 100},
				},
			},
		},
	}).Create(builder); err != nil {
		return nil, err
	}

	p.tableView = tv

	// Set up the model with panel reference for callbacks
	p.model.panel = p

	p.tableView.SetModel(p.model)

	// Double-click to edit value in cell
	p.tableView.ItemActivated().Attach(p.onItemActivated)

	return p, nil
}

func (p *Panel) onItemActivated() {
	if p.updating || p.tableView == nil || p.model == nil {
		return
	}

	idx := p.tableView.CurrentIndex()
	if idx < 0 || idx >= len(p.model.items) {
		return
	}

	item := p.model.items[idx]
	if item == nil {
		return
	}

	// Use inline input dialog for editing
	newValue, ok := inputBox(p.composite.Form(), "Edit Property", item.Name+":", item.Value)
	if !ok {
		return
	}

	if newValue != item.Value {
		item.Value = newValue
		p.model.PublishRowChanged(idx)
		p.onValueChanged(item.Name, newValue)
	}
}

// inputBox shows a simple input dialog
func inputBox(owner walk.Form, title, label, value string) (string, bool) {
	var dlg *walk.Dialog
	var edit *walk.LineEdit
	var result string
	var accepted bool

	dlg, err := walk.NewDialog(owner)
	if err != nil {
		return "", false
	}

	dlg.SetLayout(walk.NewVBoxLayout())
	dlg.SetTitle(title)
	dlg.SetMinMaxSize(walk.Size{Width: 300, Height: 120}, walk.Size{Width: 500, Height: 200})

	lbl, _ := walk.NewLabel(dlg)
	lbl.SetText(label)

	edit, _ = walk.NewLineEdit(dlg)
	edit.SetText(value)

	btnComposite, _ := walk.NewComposite(dlg)
	btnComposite.SetLayout(walk.NewHBoxLayout())

	okBtn, _ := walk.NewPushButton(btnComposite)
	okBtn.SetText("OK")
	okBtn.Clicked().Attach(func() {
		result = edit.Text()
		accepted = true
		dlg.Accept()
	})

	cancelBtn, _ := walk.NewPushButton(btnComposite)
	cancelBtn.SetText("Cancel")
	cancelBtn.Clicked().Attach(func() {
		dlg.Cancel()
	})

	dlg.Run()
	return result, accepted
}

// SetOnPropertyChanged sets the callback for property changes
func (p *Panel) SetOnPropertyChanged(fn func(*vgofile.Control, string, interface{})) {
	p.onPropertyChanged = fn
}

// SetOnFormPropertyChanged sets the callback for form property changes
func (p *Panel) SetOnFormPropertyChanged(fn func(*vgofile.Form, string, interface{})) {
	p.onFormPropertyChanged = fn
}

// SetForm sets the form to display properties for
func (p *Panel) SetForm(form *vgofile.Form) {
	p.currentControl = nil
	p.currentForm = form

	p.composite.Synchronize(func() {
		p.updating = true
		defer func() { p.updating = false }()

		if form == nil {
			p.model.items = nil
		} else {
			p.model.SetPropertiesFromForm(form)
		}
		p.tableView.SetModel(p.model)
	})
}

// SetControl sets the control to display properties for
func (p *Panel) SetControl(ctrl *vgofile.Control) {
	p.currentControl = ctrl
	p.currentForm = nil

	p.composite.Synchronize(func() {
		p.updating = true
		defer func() { p.updating = false }()

		if ctrl == nil {
			p.model.items = nil
		} else {
			p.model.SetPropertiesFromControl(ctrl)
		}
		p.tableView.SetModel(p.model)
	})
}

// RefreshControl refreshes the property display for the current control
func (p *Panel) RefreshControl(ctrl *vgofile.Control) {
	if ctrl == p.currentControl && ctrl != nil {
		p.composite.Synchronize(func() {
			p.updating = true
			defer func() { p.updating = false }()

			p.model.SetPropertiesFromControl(ctrl)
			p.tableView.SetModel(p.model)
		})
	}
}

func (p *Panel) parseValue(name string, value string) interface{} {
	switch name {
	case "Left", "Top", "Width", "Height", "TabIndex", "Value", "Minimum", "Maximum", "MaxLength":
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	case "Enabled", "Visible", "Checked", "ReadOnly", "PasswordMode", "MultiSelect", "Editable":
		return value == "true" || value == "True" || value == "1"
	}
	return value
}

func (p *Panel) onValueChanged(propName, newValue string) {
	if p.updating {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Properties: PANIC in onValueChanged: %v", r)
		}
	}()

	if p.currentControl != nil && p.onPropertyChanged != nil {
		p.onPropertyChanged(p.currentControl, propName, p.parseValue(propName, newValue))
	} else if p.currentForm != nil && p.onFormPropertyChanged != nil {
		p.onFormPropertyChanged(p.currentForm, propName, p.parseValue(propName, newValue))
	}
}

// PropertyItem represents a single property
type PropertyItem struct {
	Name     string
	Value    string
	Category string
}

// PropertyModel implements walk.TableModel for properties
type PropertyModel struct {
	walk.TableModelBase
	walk.SorterBase
	items []*PropertyItem
	panel *Panel
}

// NewPropertyModel creates a new property model
func NewPropertyModel() *PropertyModel {
	return &PropertyModel{
		items: make([]*PropertyItem, 0),
	}
}

// SetProperties sets the properties to display
func (m *PropertyModel) SetProperties(items []*PropertyItem) {
	m.items = items
	m.PublishRowsReset()
}

// SetPropertiesFromControl extracts properties from a control
func (m *PropertyModel) SetPropertiesFromControl(ctrl *vgofile.Control) {
	items := make([]*PropertyItem, 0)

	items = append(items, &PropertyItem{Name: "Name", Value: ctrl.Name, Category: "Design"})
	items = append(items, &PropertyItem{Name: "Type", Value: ctrl.Type, Category: "Design"})
	items = append(items, &PropertyItem{Name: "Left", Value: strconv.Itoa(ctrl.Left), Category: "Layout"})
	items = append(items, &PropertyItem{Name: "Top", Value: strconv.Itoa(ctrl.Top), Category: "Layout"})
	items = append(items, &PropertyItem{Name: "Width", Value: strconv.Itoa(ctrl.Width), Category: "Layout"})
	items = append(items, &PropertyItem{Name: "Height", Value: strconv.Itoa(ctrl.Height), Category: "Layout"})

	propNames := make([]string, 0, len(ctrl.Properties))
	for name := range ctrl.Properties {
		if name == "Name" || name == "Left" || name == "Top" || name == "Width" || name == "Height" {
			continue
		}
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	for _, name := range propNames {
		value := ctrl.Properties[name]
		items = append(items, &PropertyItem{
			Name:     name,
			Value:    fmt.Sprintf("%v", value),
			Category: "Misc",
		})
	}

	for eventName, handlerName := range ctrl.Events {
		items = append(items, &PropertyItem{
			Name:     eventName,
			Value:    handlerName,
			Category: "Events",
		})
	}

	m.SetProperties(items)
}

// SetPropertiesFromForm extracts properties from a form
func (m *PropertyModel) SetPropertiesFromForm(form *vgofile.Form) {
	items := make([]*PropertyItem, 0)

	items = append(items, &PropertyItem{Name: "Name", Value: form.Name, Category: "Design"})
	items = append(items, &PropertyItem{Name: "Text", Value: form.Text, Category: "Design"})
	items = append(items, &PropertyItem{Name: "Width", Value: strconv.Itoa(form.Width), Category: "Layout"})
	items = append(items, &PropertyItem{Name: "Height", Value: strconv.Itoa(form.Height), Category: "Layout"})

	m.SetProperties(items)
}

func (m *PropertyModel) RowCount() int {
	return len(m.items)
}

func (m *PropertyModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.items) {
		return nil
	}
	item := m.items[row]
	switch col {
	case 0:
		return item.Name
	case 1:
		return item.Value
	}
	return nil
}

// SetValue implements in-cell editing - called when user edits a cell
func (m *PropertyModel) SetValue(row, col int, value interface{}) error {
	if row < 0 || row >= len(m.items) || col != 1 {
		return nil
	}

	item := m.items[row]
	if str, ok := value.(string); ok {
		if str != item.Value {
			item.Value = str
			m.PublishRowChanged(row)

			// Notify panel of change
			if m.panel != nil {
				m.panel.onValueChanged(item.Name, str)
			}
		}
	}
	return nil
}
