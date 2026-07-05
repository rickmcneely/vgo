package properties

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"vg/internal/designer/controls"
	"vg/internal/ui"
	"vg/pkg/vgofile"
)

// PropertyType for editor selection
type PropertyType int

const (
	TypeString PropertyType = iota
	TypeInt
	TypeBool
	TypeEnum
)

// Panel is the properties panel for editing control properties
type Panel struct {
	composite             *walk.Composite
	tableContainer        *walk.Composite // Container holding the tableView for editor placement
	tableView             *walk.TableView
	listViewHwnd          win.HWND        // Cached actual ListView HWND
	model                 *PropertyModel
	currentControl        *vgofile.Control
	currentForm           *vgofile.Form
	onPropertyChanged     func(*vgofile.Control, string, interface{})
	onFormPropertyChanged func(*vgofile.Form, string, interface{})
	updating              bool
	registry              *controls.Registry

	// Inline editing - using raw HWND to avoid walk layout issues
	editHwnd     win.HWND
	editingRow   int
	editingCol   int
	oldEditProc  uintptr
}

// NewPanel creates a new properties panel
func NewPanel(parent *walk.Composite) (*Panel, error) {
	p := &Panel{
		composite:  parent,
		model:      NewPropertyModel(),
		editingRow: -1,
		editingCol: -1,
	}

	var tv *walk.TableView
	var tableContainer *walk.Composite

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
			// Container for table view (allows placing editors as siblings)
			Composite{
				AssignTo: &tableContainer,
				Layout:   VBox{MarginsZero: true, SpacingZero: true},
				Children: []Widget{
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
			},
		},
	}).Create(builder); err != nil {
		return nil, err
	}

	p.tableView = tv
	p.tableContainer = tableContainer

	// Set up the model with panel reference for callbacks
	p.model.panel = p

	p.tableView.SetModel(p.model)

	// Cache the ListView HWND for position queries (don't try to change row height)
	p.listViewHwnd = p.findListViewChild()

	// Single click on value column starts editing
	p.tableView.ItemActivated().Attach(p.onItemActivated)

	// Also handle mouse click for single-click editing
	p.tableView.MouseDown().Attach(p.onMouseDown)

	return p, nil
}

// setRowHeight sets the row height of the TableView using an ImageList
func (p *Panel) setRowHeight(height int) {
	const (
		LVSIL_SMALL      = 1
		LVM_FIRST        = 0x1000
		LVM_SETIMAGELIST = LVM_FIRST + 3
		ILC_COLOR32      = 0x0020
	)

	// Find and cache the actual ListView child window
	p.listViewHwnd = p.findListViewChild()

	// Create a small image list with 1x[height] pixel images to force row height
	himl := win.ImageList_Create(1, int32(height), ILC_COLOR32, 1, 1)
	if himl != 0 {
		// Apply to the actual ListView child window
		if p.listViewHwnd != 0 {
			win.SendMessage(p.listViewHwnd, LVM_SETIMAGELIST, LVSIL_SMALL, uintptr(himl))
		}
	}
}

// SetRegistry sets the control registry for property type lookup
func (p *Panel) SetRegistry(reg *controls.Registry) {
	p.registry = reg
}

func (p *Panel) onMouseDown(x, y int, button walk.MouseButton) {
	if button != walk.LeftButton {
		return
	}

	// Get the item at the click position
	idx := p.tableView.IndexAt(x, y)
	if idx < 0 {
		p.commitEdit()
		return
	}

	// Check if click is in value column (column 1)
	// Get column widths to determine which column was clicked
	col0Width := p.tableView.Columns().At(0).Width()
	if x > col0Width {
		// Clicked in value column - start editing
		p.startEditing(idx)
	} else {
		p.commitEdit()
	}
}

func (p *Panel) onItemActivated() {
	if p.updating || p.tableView == nil || p.model == nil {
		return
	}

	idx := p.tableView.CurrentIndex()
	if idx < 0 || idx >= len(p.model.items) {
		return
	}

	p.startEditing(idx)
}

func (p *Panel) startEditing(row int) {
	if p.updating || row < 0 || row >= len(p.model.items) {
		return
	}

	// Commit any existing edit first
	p.commitEdit()

	item := p.model.items[row]
	if item == nil {
		return
	}

	// Don't allow editing of Type property
	if item.Name == "Type" {
		return
	}

	// Get cell bounds using Win32 API
	cellRect := p.getCellBounds(row, 1) // Column 1 is the value column
	if cellRect.Width == 0 {
		return
	}

	// Create appropriate editor based on property type
	switch item.PropType {
	case TypeBool:
		p.createComboEditor(row, cellRect, []string{"false", "true"}, item.Value)
	case TypeEnum:
		p.createComboEditor(row, cellRect, item.EnumValues, item.Value)
	default:
		p.createLineEditor(row, cellRect, item.Value)
	}
}

// findListViewChild finds the actual ListView HWND inside the TableView
func (p *Panel) findListViewChild() win.HWND {
	// Walk's TableView wraps ListView controls - find the actual one
	var listViewHwnd win.HWND

	// Get class name buffer
	className := make([]uint16, 256)

	// Enumerate child windows to find SysListView32
	child := win.GetWindow(p.tableView.Handle(), win.GW_CHILD)
	for child != 0 {
		win.GetClassName(child, &className[0], 256)
		name := syscall.UTF16ToString(className)
		if name == "SysListView32" {
			listViewHwnd = child
			break
		}
		// Check grandchildren too
		grandchild := win.GetWindow(child, win.GW_CHILD)
		for grandchild != 0 {
			win.GetClassName(grandchild, &className[0], 256)
			name = syscall.UTF16ToString(className)
			if name == "SysListView32" {
				listViewHwnd = grandchild
				break
			}
			grandchild = win.GetWindow(grandchild, win.GW_HWNDNEXT)
		}
		if listViewHwnd != 0 {
			break
		}
		child = win.GetWindow(child, win.GW_HWNDNEXT)
	}

	return listViewHwnd
}

// getCellBounds returns the bounds of a cell based on column widths
func (p *Panel) getCellBounds(row, col int) walk.Rectangle {
	// Get column 0 width from walk's API
	col0Width := p.tableView.Columns().At(0).Width()

	// Get column 1 actual width (not stretched beyond visible area)
	col1Width := p.tableView.Columns().At(1).Width()

	// Ensure it doesn't exceed visible area minus scrollbar
	tvBounds := p.tableView.ClientBoundsPixels()
	maxCol1Width := tvBounds.Width - col0Width - 20 // account for scrollbar
	if col1Width > maxCol1Width {
		col1Width = maxCol1Width
	}
	if col1Width < 50 {
		col1Width = 50 // minimum width
	}

	// Try to get actual row position from the real ListView
	type RECT struct {
		Left, Top, Right, Bottom int32
	}

	const (
		LVM_FIRST       = 0x1000
		LVM_GETITEMRECT = LVM_FIRST + 14
		LVIR_BOUNDS     = 0
	)

	y := 0
	rowHeight := 21

	// Use the cached ListView HWND to get actual row position
	if p.listViewHwnd != 0 {
		rect := RECT{Left: LVIR_BOUNDS}
		if win.SendMessage(p.listViewHwnd, LVM_GETITEMRECT, uintptr(row), uintptr(unsafe.Pointer(&rect))) != 0 {
			y = int(rect.Top)
			rowHeight = int(rect.Bottom - rect.Top)
			if rowHeight < 10 {
				rowHeight = 21
			}
		}
	}

	// Fallback to calculated position
	if y == 0 && row > 0 {
		headerHeight := 21
		y = headerHeight + row*rowHeight
	}

	x := 0
	width := col0Width
	if col == 1 {
		x = col0Width + 20 // aligned with divider line
		width = col1Width + 28 // match visible cell width
	}

	return walk.Rectangle{
		X:      x,
		Y:      y,
		Width:  width,
		Height: rowHeight,
	}
}

// Global map to store panel references for subclassed edit controls
var editPanelMap = make(map[win.HWND]*Panel)

func (p *Panel) createLineEditor(row int, bounds walk.Rectangle, value string) {
	// Create a raw Win32 EDIT control to avoid walk layout issues
	className, _ := syscall.UTF16PtrFromString("EDIT")
	text, _ := syscall.UTF16PtrFromString(value)

	const (
		WS_CHILD         = 0x40000000
		WS_VISIBLE       = 0x10000000
		WS_TABSTOP       = 0x00010000
		ES_AUTOHSCROLL   = 0x0080
		ES_LEFT          = 0x0000
		WS_EX_CLIENTEDGE = 0x00000200
	)

	hwnd := win.CreateWindowEx(
		WS_EX_CLIENTEDGE,
		className,
		text,
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL|ES_LEFT,
		int32(bounds.X), int32(bounds.Y), int32(bounds.Width), int32(bounds.Height),
		p.tableView.Handle(),
		0,
		0,
		nil,
	)

	if hwnd == 0 {
		log.Printf("Failed to create edit control")
		return
	}

	// Set font to match tableview
	win.SendMessage(hwnd, win.WM_SETFONT, uintptr(win.GetStockObject(win.DEFAULT_GUI_FONT)), 1)

	// Select all text
	win.SendMessage(hwnd, win.EM_SETSEL, 0, uintptr(len(value)))

	// Subclass the edit control to handle Enter/Escape keys
	editPanelMap[hwnd] = p
	p.oldEditProc = win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, syscall.NewCallback(editSubclassProc))

	// Set focus to the edit control
	win.SetFocus(hwnd)

	// Bring to front
	win.SetWindowPos(hwnd, win.HWND_TOP, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE)

	p.editHwnd = hwnd
	p.editingRow = row
	p.editingCol = 1
}

// editSubclassProc handles keyboard input for the edit control
func editSubclassProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	panel, ok := editPanelMap[hwnd]
	if !ok {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case win.WM_KEYDOWN:
		switch wParam {
		case win.VK_RETURN:
			// Commit edit on Enter
			panel.commitEdit()
			return 0
		case win.VK_ESCAPE:
			// Cancel edit on Escape
			panel.cancelEdit()
			return 0
		case win.VK_TAB:
			// Commit and move to next row
			row := panel.editingRow
			panel.commitEdit()
			if row+1 < len(panel.model.items) {
				panel.tableView.SetCurrentIndex(row + 1)
				panel.startEditing(row + 1)
			}
			return 0
		}
	case win.WM_KILLFOCUS:
		// Commit on focus loss (but only if we're still editing)
		if panel.editHwnd == hwnd {
			panel.commitEdit()
		}
		return 0
	}

	// Call the original window procedure
	return win.CallWindowProc(panel.oldEditProc, hwnd, msg, wParam, lParam)
}

func (p *Panel) createComboEditor(row int, bounds walk.Rectangle, options []string, value string) {
	// Create a raw Win32 COMBOBOX control to avoid walk layout issues
	className, _ := syscall.UTF16PtrFromString("COMBOBOX")

	const (
		WS_CHILD         = 0x40000000
		WS_VISIBLE       = 0x10000000
		WS_VSCROLL       = 0x00200000
		WS_TABSTOP       = 0x00010000
		CBS_DROPDOWNLIST = 0x0003
		CBS_HASSTRINGS   = 0x0200
		CB_ADDSTRING     = 0x0143
		CB_SETCURSEL     = 0x014E
	)

	// Combo needs extra height for the dropdown list
	dropdownHeight := 200

	hwnd := win.CreateWindowEx(
		0,
		className,
		nil,
		WS_CHILD|WS_VISIBLE|WS_VSCROLL|WS_TABSTOP|CBS_DROPDOWNLIST|CBS_HASSTRINGS,
		int32(bounds.X), int32(bounds.Y), int32(bounds.Width), int32(dropdownHeight),
		p.tableView.Handle(),
		0,
		0,
		nil,
	)

	if hwnd == 0 {
		log.Printf("Failed to create combo control")
		return
	}

	// Set font to match tableview
	win.SendMessage(hwnd, win.WM_SETFONT, uintptr(win.GetStockObject(win.DEFAULT_GUI_FONT)), 1)

	// Add options and select current value
	selectedIdx := 0
	for i, opt := range options {
		optStr, _ := syscall.UTF16PtrFromString(opt)
		win.SendMessage(hwnd, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(optStr)))
		if opt == value {
			selectedIdx = i
		}
	}

	// Select current value
	win.SendMessage(hwnd, CB_SETCURSEL, uintptr(selectedIdx), 0)

	// Set focus and bring to front
	win.SetFocus(hwnd)
	win.SetWindowPos(hwnd, win.HWND_TOP, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE)

	p.editHwnd = hwnd
	p.editingRow = row
	p.editingCol = 1
}

func (p *Panel) commitEdit() {
	if p.editHwnd == 0 || p.editingRow < 0 {
		return
	}

	// Get text from the edit control
	const (
		WM_GETTEXT       = 0x000D
		WM_GETTEXTLENGTH = 0x000E
	)

	textLen := int(win.SendMessage(p.editHwnd, WM_GETTEXTLENGTH, 0, 0))
	if textLen > 0 {
		buf := make([]uint16, textLen+1)
		win.SendMessage(p.editHwnd, WM_GETTEXT, uintptr(textLen+1), uintptr(unsafe.Pointer(&buf[0])))
		newValue := syscall.UTF16ToString(buf)

		// Update the model
		if p.editingRow < len(p.model.items) {
			item := p.model.items[p.editingRow]
			if newValue != item.Value {
				item.Value = newValue
				p.model.PublishRowChanged(p.editingRow)
				p.onValueChanged(item.Name, newValue)
			}
		}
	}

	p.destroyEditor()
}

func (p *Panel) cancelEdit() {
	p.destroyEditor()
}

func (p *Panel) destroyEditor() {
	if p.editHwnd != 0 {
		// Clean up subclass map
		delete(editPanelMap, p.editHwnd)
		win.DestroyWindow(p.editHwnd)
		p.editHwnd = 0
		p.oldEditProc = 0
	}
	p.editingRow = -1
	p.editingCol = -1

	// Refresh the table view
	win.InvalidateRect(p.tableView.Handle(), nil, true)
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
	p.commitEdit()
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
	p.commitEdit()
	p.currentControl = ctrl
	p.currentForm = nil

	p.composite.Synchronize(func() {
		p.updating = true
		defer func() { p.updating = false }()

		if ctrl == nil {
			p.model.items = nil
		} else {
			p.model.SetPropertiesFromControl(ctrl, p.registry)
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

			p.model.SetPropertiesFromControl(ctrl, p.registry)
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
	case "Enabled", "Visible", "Checked", "ReadOnly", "PasswordMode", "MultiSelect", "Editable",
		"ControlBox", "MinimizeBox", "MaximizeBox", "Resizable":
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
	Name       string
	Value      string
	Category   string
	PropType   PropertyType
	EnumValues []string
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
func (m *PropertyModel) SetPropertiesFromControl(ctrl *vgofile.Control, registry *controls.Registry) {
	items := make([]*PropertyItem, 0)

	// Get control definition for property types
	var ctrlDef *controls.ControlDefinition
	if registry != nil {
		ctrlDef, _ = registry.Get(ctrl.Type)
	}

	// Helper to get property type
	getPropType := func(name string) (PropertyType, []string) {
		// Check standard properties
		switch name {
		case "Left", "Top", "Width", "Height", "TabIndex", "Value", "Minimum", "Maximum", "MaxLength":
			return TypeInt, nil
		case "Enabled", "Visible", "Checked", "ReadOnly", "PasswordMode", "MultiSelect", "Editable":
			return TypeBool, nil
		}

		// Check control-specific properties
		if ctrlDef != nil {
			for _, prop := range ctrlDef.Properties {
				if prop.Name == name {
					switch prop.Type {
					case controls.PropBool:
						return TypeBool, nil
					case controls.PropInt:
						return TypeInt, nil
					case controls.PropEnum:
						return TypeEnum, prop.EnumValues
					}
				}
			}
		}

		return TypeString, nil
	}

	items = append(items, &PropertyItem{Name: "Name", Value: ctrl.Name, Category: "Design", PropType: TypeString})
	items = append(items, &PropertyItem{Name: "Type", Value: ctrl.Type, Category: "Design", PropType: TypeString})

	propType, _ := getPropType("Left")
	items = append(items, &PropertyItem{Name: "Left", Value: strconv.Itoa(ctrl.Left), Category: "Layout", PropType: propType})
	propType, _ = getPropType("Top")
	items = append(items, &PropertyItem{Name: "Top", Value: strconv.Itoa(ctrl.Top), Category: "Layout", PropType: propType})
	propType, _ = getPropType("Width")
	items = append(items, &PropertyItem{Name: "Width", Value: strconv.Itoa(ctrl.Width), Category: "Layout", PropType: propType})
	propType, _ = getPropType("Height")
	items = append(items, &PropertyItem{Name: "Height", Value: strconv.Itoa(ctrl.Height), Category: "Layout", PropType: propType})

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
		propType, enumVals := getPropType(name)
		items = append(items, &PropertyItem{
			Name:       name,
			Value:      fmt.Sprintf("%v", value),
			Category:   "Misc",
			PropType:   propType,
			EnumValues: enumVals,
		})
	}

	for eventName, handlerName := range ctrl.Events {
		items = append(items, &PropertyItem{
			Name:     eventName,
			Value:    handlerName,
			Category: "Events",
			PropType: TypeString,
		})
	}

	m.SetProperties(items)
}

// SetPropertiesFromForm extracts properties from a form
func (m *PropertyModel) SetPropertiesFromForm(form *vgofile.Form) {
	items := make([]*PropertyItem, 0)

	// Design properties
	items = append(items, &PropertyItem{Name: "Name", Value: form.Name, Category: "Design", PropType: TypeString})
	items = append(items, &PropertyItem{Name: "Text", Value: form.Text, Category: "Design", PropType: TypeString})

	// Layout properties
	items = append(items, &PropertyItem{Name: "Width", Value: strconv.Itoa(form.Width), Category: "Layout", PropType: TypeInt})
	items = append(items, &PropertyItem{Name: "Height", Value: strconv.Itoa(form.Height), Category: "Layout", PropType: TypeInt})

	// Window style properties
	items = append(items, &PropertyItem{Name: "ControlBox", Value: strconv.FormatBool(form.ControlBox), Category: "Window", PropType: TypeBool})
	items = append(items, &PropertyItem{Name: "MinimizeBox", Value: strconv.FormatBool(form.MinimizeBox), Category: "Window", PropType: TypeBool})
	items = append(items, &PropertyItem{Name: "MaximizeBox", Value: strconv.FormatBool(form.MaximizeBox), Category: "Window", PropType: TypeBool})
	items = append(items, &PropertyItem{Name: "Resizable", Value: strconv.FormatBool(form.Resizable), Category: "Window", PropType: TypeBool})
	items = append(items, &PropertyItem{
		Name:       "StartPosition",
		Value:      form.StartPos,
		Category:   "Window",
		PropType:   TypeEnum,
		EnumValues: []string{"CenterScreen", "CenterParent", "Manual"},
	})

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
