package canvas

import (
	"image"
	"log"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"

	"vg/internal/designer/controls"
	"vg/pkg/vgofile"
)

const (
	gridSize      = 8
	handleSize    = 8
	minCtrlWidth  = 20
	minCtrlHeight = 10
)

// Canvas is the form designer surface
type Canvas struct {
	*walk.CustomWidget
	parent             *walk.Composite
	registry           *controls.Registry
	form               *vgofile.Form
	selectedControl    *vgofile.Control
	formSelected       bool
	placementMode      string
	dragging           bool
	resizing           bool
	resizeHandle       int
	dragStart          image.Point
	dragOffset         image.Point
	dragRect           walk.Rectangle // For outline dragging
	onSelectionChanged func(*vgofile.Control)
	onFormSelected     func(*vgofile.Form)
	onControlModified  func(*vgofile.Control)
	clipboard          *vgofile.Control
	snapToGrid         bool
	showGrid           bool
	dpiScale           float64
}

// NewCanvas creates a new designer canvas
func NewCanvas(parent *walk.Composite, registry *controls.Registry) (*Canvas, error) {
	c := &Canvas{
		parent:     parent,
		registry:   registry,
		snapToGrid: true,
		showGrid:   true,
		dpiScale:   1.0,
	}

	cw, err := walk.NewCustomWidget(parent, 0, c.paint)
	if err != nil {
		return nil, err
	}

	c.CustomWidget = cw
	cw.SetClearsBackground(true)
	cw.SetInvalidatesOnResize(true)
	cw.SetPaintMode(walk.PaintNormal)

	cw.MouseDown().Attach(c.onMouseDown)
	cw.MouseMove().Attach(c.onMouseMove)
	cw.MouseUp().Attach(c.onMouseUp)
	cw.KeyDown().Attach(c.onKeyDown)

	return c, nil
}

// scaleForDPI scales a value from physical pixels to logical pixels
func (c *Canvas) scaleForDPI(v int) int {
	if c.dpiScale <= 1.0 {
		return v
	}
	return int(float64(v) / c.dpiScale)
}

// updateDPIScale updates the DPI scale factor
func (c *Canvas) updateDPIScale() {
	dpi := c.DPI()
	c.dpiScale = float64(dpi) / 96.0
	if c.dpiScale < 1.0 {
		c.dpiScale = 1.0
	}
}

// SetForm sets the form being designed
func (c *Canvas) SetForm(form *vgofile.Form) {
	c.form = form
	c.selectedControl = nil
	c.Invalidate()
	c.notifySelectionChanged()
}

// SetPlacementMode sets the control type to place
func (c *Canvas) SetPlacementMode(controlType string) {
	c.placementMode = controlType
	if controlType != "" {
		c.setCursor(win.IDC_CROSS)
	} else {
		c.setCursor(win.IDC_ARROW)
	}
}

// SetOnSelectionChanged sets the callback for selection changes
func (c *Canvas) SetOnSelectionChanged(fn func(*vgofile.Control)) {
	c.onSelectionChanged = fn
}

// SetOnControlModified sets the callback for control modifications
func (c *Canvas) SetOnControlModified(fn func(*vgofile.Control)) {
	c.onControlModified = fn
}

// SetOnFormSelected sets the callback for form selection
func (c *Canvas) SetOnFormSelected(fn func(*vgofile.Form)) {
	c.onFormSelected = fn
}

// SetVisible sets the visibility of the canvas parent container
func (c *Canvas) SetVisible(visible bool) {
	c.parent.SetVisible(visible)
}

// Visible returns whether the canvas parent container is visible
func (c *Canvas) Visible() bool {
	return c.parent.Visible()
}

// RefreshControl refreshes the display of a specific control
func (c *Canvas) RefreshControl(ctrl *vgofile.Control) {
	c.Invalidate()
}

// Refresh redraws the entire canvas
func (c *Canvas) Refresh() {
	c.Invalidate()
}

// CutSelected cuts the selected control to clipboard
func (c *Canvas) CutSelected() {
	if c.selectedControl != nil {
		c.clipboard = c.selectedControl.Clone()
		c.DeleteSelected()
	}
}

// CopySelected copies the selected control to clipboard
func (c *Canvas) CopySelected() {
	if c.selectedControl != nil {
		c.clipboard = c.selectedControl.Clone()
	}
}

// Paste pastes the clipboard control
func (c *Canvas) Paste() {
	if c.clipboard != nil && c.form != nil {
		newCtrl := c.clipboard.Clone()
		newCtrl.Name = c.form.GenerateControlName(newCtrl.Type)
		newCtrl.Left += gridSize * 2
		newCtrl.Top += gridSize * 2
		c.form.AddControl(newCtrl)
		c.selectedControl = newCtrl
		c.Invalidate()
		c.notifySelectionChanged()
	}
}

// DeleteSelected deletes the selected control
func (c *Canvas) DeleteSelected() {
	if c.selectedControl != nil && c.form != nil {
		c.form.RemoveControl(c.selectedControl)
		c.selectedControl = nil
		c.Invalidate()
		c.notifySelectionChanged()
	}
}

func (c *Canvas) paint(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Canvas: PANIC in paint: %v", r)
		}
	}()

	// Update DPI scale for coordinate calculations
	c.updateDPIScale()

	bounds := c.ClientBounds()

	// Draw canvas background
	bgBrush, _ := walk.NewSolidColorBrush(walk.RGB(240, 240, 240))
	defer bgBrush.Dispose()
	canvas.FillRectangle(bgBrush, bounds)

	// Draw grid
	if c.showGrid {
		c.drawGrid(canvas, bounds)
	}

	// Draw form
	formWidth := 400
	formHeight := 300
	if c.form != nil {
		formWidth = c.form.Width
		formHeight = c.form.Height
	}

	formRect := walk.Rectangle{X: 10, Y: 10, Width: formWidth, Height: formHeight}

	// Form background
	whiteBrush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer whiteBrush.Dispose()
	canvas.FillRectangle(whiteBrush, formRect)

	// Form border
	borderPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(100, 100, 100))
	defer borderPen.Dispose()
	canvas.DrawRectangle(borderPen, formRect)

	// Form title bar
	titleRect := walk.Rectangle{X: 10, Y: 10, Width: formWidth, Height: 25}
	titleBrush, _ := walk.NewSolidColorBrush(walk.RGB(0, 120, 215))
	defer titleBrush.Dispose()
	canvas.FillRectangle(titleBrush, titleRect)

	// Form title text
	if c.form != nil {
		font, _ := walk.NewFont("Segoe UI", 9, 0)
		defer font.Dispose()
		canvas.DrawText(c.form.Text, font, walk.RGB(255, 255, 255), titleRect, walk.TextLeft|walk.TextVCenter)
	}

	// Draw controls
	if c.form != nil {
		for _, ctrl := range c.form.Controls {
			c.drawControl(canvas, ctrl)
		}
	}

	// Draw drag outline if dragging
	if c.dragging && c.dragRect.Width > 0 {
		dragPen, _ := walk.NewCosmeticPen(walk.PenDash, walk.RGB(0, 120, 215))
		defer dragPen.Dispose()
		canvas.DrawRectangle(dragPen, c.dragRect)
	}

	return nil
}

func (c *Canvas) drawGrid(canvas *walk.Canvas, bounds walk.Rectangle) {
	gridColor := walk.RGB(220, 220, 220)
	pen, _ := walk.NewCosmeticPen(walk.PenSolid, gridColor)
	defer pen.Dispose()

	for x := gridSize; x < bounds.Width; x += gridSize {
		for y := gridSize; y < bounds.Height; y += gridSize {
			canvas.DrawLine(pen, walk.Point{X: x, Y: y}, walk.Point{X: x + 1, Y: y})
		}
	}
}

func (c *Canvas) drawControl(canvas *walk.Canvas, ctrl *vgofile.Control) {
	offsetX := 10
	offsetY := 35

	rect := walk.Rectangle{
		X:      ctrl.Left + offsetX,
		Y:      ctrl.Top + offsetY,
		Width:  ctrl.Width,
		Height: ctrl.Height,
	}

	switch ctrl.Type {
	case "Button":
		c.drawButton(canvas, ctrl, rect)
	case "Label":
		c.drawLabel(canvas, ctrl, rect)
	case "TextBox":
		c.drawTextBox(canvas, ctrl, rect)
	case "CheckBox":
		c.drawCheckBox(canvas, ctrl, rect)
	case "RadioButton":
		c.drawRadioButton(canvas, ctrl, rect)
	case "ComboBox":
		c.drawComboBox(canvas, ctrl, rect)
	case "ListBox":
		c.drawListBox(canvas, ctrl, rect)
	case "TreeView":
		c.drawTreeView(canvas, ctrl, rect)
	case "GroupBox":
		c.drawGroupBox(canvas, ctrl, rect)
	case "Panel":
		c.drawPanel(canvas, ctrl, rect)
	case "PictureBox":
		c.drawPictureBox(canvas, ctrl, rect)
	case "ProgressBar":
		c.drawProgressBar(canvas, ctrl, rect)
	default:
		brush, _ := walk.NewSolidColorBrush(walk.RGB(200, 200, 200))
		defer brush.Dispose()
		canvas.FillRectangle(brush, rect)
	}

	if ctrl == c.selectedControl {
		c.drawSelectionHandles(canvas, rect)
	}
}

func (c *Canvas) drawButton(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(225, 225, 225))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(173, 173, 173))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)

	text := ctrl.GetStringProperty("Text", "Button")
	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()
	canvas.DrawText(text, font, walk.RGB(0, 0, 0), rect, walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}

func (c *Canvas) drawLabel(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	text := ctrl.GetStringProperty("Text", "Label")
	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()
	canvas.DrawText(text, font, walk.RGB(0, 0, 0), rect, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
}

func (c *Canvas) drawTextBox(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(122, 122, 122))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)

	text := ctrl.GetStringProperty("Text", "")
	if text != "" {
		font, _ := walk.NewFont("Segoe UI", 9, 0)
		defer font.Dispose()
		textRect := walk.Rectangle{X: rect.X + 3, Y: rect.Y, Width: rect.Width - 6, Height: rect.Height}
		canvas.DrawText(text, font, walk.RGB(0, 0, 0), textRect, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
	}
}

func (c *Canvas) drawCheckBox(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	boxSize := 13
	boxRect := walk.Rectangle{X: rect.X, Y: rect.Y + (rect.Height-boxSize)/2, Width: boxSize, Height: boxSize}

	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer brush.Dispose()
	canvas.FillRectangle(brush, boxRect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(100, 100, 100))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, boxRect)

	if ctrl.GetBoolProperty("Checked", false) {
		checkPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
		defer checkPen.Dispose()
		canvas.DrawLine(checkPen, walk.Point{X: boxRect.X + 3, Y: boxRect.Y + 6}, walk.Point{X: boxRect.X + 5, Y: boxRect.Y + 9})
		canvas.DrawLine(checkPen, walk.Point{X: boxRect.X + 5, Y: boxRect.Y + 9}, walk.Point{X: boxRect.X + 10, Y: boxRect.Y + 3})
	}

	text := ctrl.GetStringProperty("Text", "CheckBox")
	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()
	textRect := walk.Rectangle{X: rect.X + boxSize + 4, Y: rect.Y, Width: rect.Width - boxSize - 4, Height: rect.Height}
	canvas.DrawText(text, font, walk.RGB(0, 0, 0), textRect, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
}

func (c *Canvas) drawRadioButton(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	circleSize := 13
	circleX := rect.X
	circleY := rect.Y + (rect.Height-circleSize)/2

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(100, 100, 100))
	defer pen.Dispose()

	circleRect := walk.Rectangle{X: circleX, Y: circleY, Width: circleSize, Height: circleSize}
	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer brush.Dispose()
	canvas.FillRectangle(brush, circleRect)
	canvas.DrawRectangle(pen, circleRect)

	if ctrl.GetBoolProperty("Checked", false) {
		fillBrush, _ := walk.NewSolidColorBrush(walk.RGB(0, 120, 215))
		defer fillBrush.Dispose()
		innerRect := walk.Rectangle{X: circleX + 3, Y: circleY + 3, Width: circleSize - 6, Height: circleSize - 6}
		canvas.FillRectangle(fillBrush, innerRect)
	}

	text := ctrl.GetStringProperty("Text", "RadioButton")
	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()
	textRect := walk.Rectangle{X: rect.X + circleSize + 4, Y: rect.Y, Width: rect.Width - circleSize - 4, Height: rect.Height}
	canvas.DrawText(text, font, walk.RGB(0, 0, 0), textRect, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
}

func (c *Canvas) drawComboBox(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(122, 122, 122))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)

	btnRect := walk.Rectangle{X: rect.X + rect.Width - 20, Y: rect.Y, Width: 20, Height: rect.Height}
	btnBrush, _ := walk.NewSolidColorBrush(walk.RGB(225, 225, 225))
	defer btnBrush.Dispose()
	canvas.FillRectangle(btnBrush, btnRect)

	arrowPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	defer arrowPen.Dispose()
	midX := btnRect.X + btnRect.Width/2
	midY := btnRect.Y + btnRect.Height/2
	canvas.DrawLine(arrowPen, walk.Point{X: midX - 4, Y: midY - 2}, walk.Point{X: midX, Y: midY + 2})
	canvas.DrawLine(arrowPen, walk.Point{X: midX, Y: midY + 2}, walk.Point{X: midX + 4, Y: midY - 2})
}

func (c *Canvas) drawListBox(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(122, 122, 122))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)

	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()
	for i := 0; i < 3 && i*20 < rect.Height-4; i++ {
		itemRect := walk.Rectangle{X: rect.X + 2, Y: rect.Y + 2 + i*20, Width: rect.Width - 4, Height: 18}
		canvas.DrawText("(Item)", font, walk.RGB(128, 128, 128), itemRect, walk.TextLeft|walk.TextVCenter)
	}
}

func (c *Canvas) drawTreeView(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(122, 122, 122))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)

	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()

	itemHeight := 18
	indent := 16

	if itemHeight < rect.Height-4 {
		boxX := rect.X + 4
		boxY := rect.Y + 4 + 4
		canvas.DrawRectangle(pen, walk.Rectangle{X: boxX, Y: boxY, Width: 9, Height: 9})
		canvas.DrawLine(pen, walk.Point{X: boxX + 2, Y: boxY + 4}, walk.Point{X: boxX + 7, Y: boxY + 4})

		itemRect := walk.Rectangle{X: rect.X + 18, Y: rect.Y + 4, Width: rect.Width - 22, Height: itemHeight}
		canvas.DrawText("(Root)", font, walk.RGB(0, 0, 0), itemRect, walk.TextLeft|walk.TextVCenter)
	}

	for i := 1; i < 3 && (i+1)*itemHeight < rect.Height-4; i++ {
		itemRect := walk.Rectangle{X: rect.X + 4 + indent, Y: rect.Y + 4 + i*itemHeight, Width: rect.Width - 8 - indent, Height: itemHeight}
		canvas.DrawText("(Item)", font, walk.RGB(128, 128, 128), itemRect, walk.TextLeft|walk.TextVCenter)
	}
}

func (c *Canvas) drawGroupBox(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(180, 180, 180))
	defer pen.Dispose()

	text := ctrl.GetStringProperty("Text", "GroupBox")
	textWidth := 60

	canvas.DrawLine(pen, walk.Point{X: rect.X, Y: rect.Y + 8}, walk.Point{X: rect.X + 8, Y: rect.Y + 8})
	canvas.DrawLine(pen, walk.Point{X: rect.X + 10 + textWidth, Y: rect.Y + 8}, walk.Point{X: rect.X + rect.Width, Y: rect.Y + 8})
	canvas.DrawLine(pen, walk.Point{X: rect.X, Y: rect.Y + 8}, walk.Point{X: rect.X, Y: rect.Y + rect.Height})
	canvas.DrawLine(pen, walk.Point{X: rect.X, Y: rect.Y + rect.Height}, walk.Point{X: rect.X + rect.Width, Y: rect.Y + rect.Height})
	canvas.DrawLine(pen, walk.Point{X: rect.X + rect.Width, Y: rect.Y + 8}, walk.Point{X: rect.X + rect.Width, Y: rect.Y + rect.Height})

	font, _ := walk.NewFont("Segoe UI", 9, 0)
	defer font.Dispose()
	textRect := walk.Rectangle{X: rect.X + 10, Y: rect.Y, Width: rect.Width - 20, Height: 16}
	canvas.DrawText(text, font, walk.RGB(0, 0, 0), textRect, walk.TextLeft|walk.TextVCenter)
}

func (c *Canvas) drawPanel(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(240, 240, 240))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(180, 180, 180))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)
}

func (c *Canvas) drawPictureBox(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(230, 230, 230))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(180, 180, 180))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)
	canvas.DrawLine(pen, walk.Point{X: rect.X, Y: rect.Y}, walk.Point{X: rect.X + rect.Width, Y: rect.Y + rect.Height})
	canvas.DrawLine(pen, walk.Point{X: rect.X + rect.Width, Y: rect.Y}, walk.Point{X: rect.X, Y: rect.Y + rect.Height})
}

func (c *Canvas) drawProgressBar(canvas *walk.Canvas, ctrl *vgofile.Control, rect walk.Rectangle) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(230, 230, 230))
	defer brush.Dispose()
	canvas.FillRectangle(brush, rect)

	value := ctrl.GetIntProperty("Value", 50)
	min := ctrl.GetIntProperty("Minimum", 0)
	max := ctrl.GetIntProperty("Maximum", 100)
	if max > min {
		progress := float64(value-min) / float64(max-min)
		fillWidth := int(float64(rect.Width-2) * progress)
		fillRect := walk.Rectangle{X: rect.X + 1, Y: rect.Y + 1, Width: fillWidth, Height: rect.Height - 2}
		fillBrush, _ := walk.NewSolidColorBrush(walk.RGB(6, 176, 37))
		defer fillBrush.Dispose()
		canvas.FillRectangle(fillBrush, fillRect)
	}

	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(180, 180, 180))
	defer pen.Dispose()
	canvas.DrawRectangle(pen, rect)
}

func (c *Canvas) drawSelectionHandles(canvas *walk.Canvas, rect walk.Rectangle) {
	selPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 120, 215))
	defer selPen.Dispose()
	canvas.DrawRectangle(selPen, rect)

	handleFillBrush, _ := walk.NewSolidColorBrush(walk.RGB(0, 120, 215))
	defer handleFillBrush.Dispose()
	handleBorderPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(255, 255, 255))
	defer handleBorderPen.Dispose()

	handles := c.getHandleRects(rect)
	for _, h := range handles {
		canvas.FillRectangle(handleFillBrush, h)
		canvas.DrawRectangle(handleBorderPen, h)
	}
}

func (c *Canvas) getHandleRects(rect walk.Rectangle) []walk.Rectangle {
	hs := handleSize / 2
	return []walk.Rectangle{
		{X: rect.X - hs, Y: rect.Y - hs, Width: handleSize, Height: handleSize},
		{X: rect.X + rect.Width/2 - hs, Y: rect.Y - hs, Width: handleSize, Height: handleSize},
		{X: rect.X + rect.Width - hs, Y: rect.Y - hs, Width: handleSize, Height: handleSize},
		{X: rect.X + rect.Width - hs, Y: rect.Y + rect.Height/2 - hs, Width: handleSize, Height: handleSize},
		{X: rect.X + rect.Width - hs, Y: rect.Y + rect.Height - hs, Width: handleSize, Height: handleSize},
		{X: rect.X + rect.Width/2 - hs, Y: rect.Y + rect.Height - hs, Width: handleSize, Height: handleSize},
		{X: rect.X - hs, Y: rect.Y + rect.Height - hs, Width: handleSize, Height: handleSize},
		{X: rect.X - hs, Y: rect.Y + rect.Height/2 - hs, Width: handleSize, Height: handleSize},
	}
}

func (c *Canvas) onMouseDown(x, y int, button walk.MouseButton) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Canvas: PANIC in onMouseDown: %v", r)
		}
	}()

	if button != walk.LeftButton {
		return
	}

	c.SetFocus()

	// Scale mouse coordinates for DPI
	c.updateDPIScale()
	x = c.scaleForDPI(x)
	y = c.scaleForDPI(y)

	pt := image.Point{X: x, Y: y}

	if c.placementMode != "" {
		c.placeNewControl(x, y)
		return
	}

	if c.selectedControl != nil {
		handle := c.hitTestHandles(pt)
		if handle >= 0 {
			c.resizing = true
			c.resizeHandle = handle
			c.dragStart = pt
			win.SetCapture(c.Handle())
			return
		}
	}

	ctrl := c.hitTestControl(pt)
	if ctrl != nil {
		c.selectedControl = ctrl
		c.dragging = true
		c.dragStart = pt
		c.dragOffset = image.Point{X: x - ctrl.Left - 10, Y: y - ctrl.Top - 35}
		// Initialize drag rectangle for outline dragging
		c.dragRect = walk.Rectangle{
			X:      ctrl.Left + 10,
			Y:      ctrl.Top + 35,
			Width:  ctrl.Width,
			Height: ctrl.Height,
		}
		win.SetCapture(c.Handle())
		c.Invalidate()
		c.notifySelectionChanged()
		return
	}

	if c.hitTestForm(pt) {
		c.selectedControl = nil
		c.formSelected = true
		c.Invalidate()
		c.notifyFormSelected()
		return
	}

	c.selectedControl = nil
	c.formSelected = false
	c.Invalidate()
	c.notifySelectionChanged()
}

func (c *Canvas) onMouseMove(x, y int, button walk.MouseButton) {
	// Scale mouse coordinates for DPI
	x = c.scaleForDPI(x)
	y = c.scaleForDPI(y)

	pt := image.Point{X: x, Y: y}

	if c.dragging && c.selectedControl != nil {
		newX := x - c.dragOffset.X
		newY := y - c.dragOffset.Y

		if c.snapToGrid {
			newX = (newX / gridSize) * gridSize
			newY = (newY / gridSize) * gridSize
		}

		// Update drag rectangle for outline (don't move actual control yet)
		c.dragRect = walk.Rectangle{
			X:      newX,
			Y:      newY,
			Width:  c.selectedControl.Width,
			Height: c.selectedControl.Height,
		}
		c.Invalidate()
		return
	}

	if c.resizing && c.selectedControl != nil {
		c.resizeControl(pt)
		c.Invalidate()
		return
	}

	if c.selectedControl != nil {
		handle := c.hitTestHandles(pt)
		if handle >= 0 {
			c.setResizeCursor(handle)
			return
		}
	}

	if c.placementMode != "" {
		c.setCursor(win.IDC_CROSS)
	} else {
		c.setCursor(win.IDC_ARROW)
	}
}

func (c *Canvas) onMouseUp(x, y int, button walk.MouseButton) {
	win.ReleaseCapture()

	if c.dragging && c.selectedControl != nil {
		// Apply the final position from drag rectangle
		c.selectedControl.Left = c.dragRect.X - 10
		c.selectedControl.Top = c.dragRect.Y - 35
		if c.onControlModified != nil {
			c.onControlModified(c.selectedControl)
		}
	} else if c.resizing && c.selectedControl != nil {
		if c.onControlModified != nil {
			c.onControlModified(c.selectedControl)
		}
	}

	c.dragging = false
	c.resizing = false
	c.dragRect = walk.Rectangle{}
	c.Invalidate()
}

func (c *Canvas) onKeyDown(key walk.Key) {
	if c.selectedControl == nil {
		return
	}

	switch key {
	case walk.KeyDelete:
		c.DeleteSelected()
	case walk.KeyLeft:
		c.selectedControl.Left -= gridSize
		c.Invalidate()
	case walk.KeyRight:
		c.selectedControl.Left += gridSize
		c.Invalidate()
	case walk.KeyUp:
		c.selectedControl.Top -= gridSize
		c.Invalidate()
	case walk.KeyDown:
		c.selectedControl.Top += gridSize
		c.Invalidate()
	}
}

func (c *Canvas) placeNewControl(x, y int) {
	if c.form == nil {
		return
	}

	def, err := c.registry.Get(c.placementMode)
	if err != nil {
		return
	}

	ctrlX := x - 10
	ctrlY := y - 35

	if c.snapToGrid {
		ctrlX = (ctrlX / gridSize) * gridSize
		ctrlY = (ctrlY / gridSize) * gridSize
	}

	ctrl := vgofile.NewControl(c.placementMode)
	ctrl.Width = def.DefaultSize.X
	ctrl.Height = def.DefaultSize.Y

	for _, prop := range def.Properties {
		if prop.DefaultValue != nil {
			ctrl.SetProperty(prop.Name, prop.DefaultValue)
		}
	}

	ctrl.Name = c.form.GenerateControlName(c.placementMode)
	ctrl.Left = ctrlX
	ctrl.Top = ctrlY

	c.form.AddControl(ctrl)
	c.selectedControl = ctrl
	c.placementMode = ""
	c.setCursor(win.IDC_ARROW)
	c.Invalidate()
	c.notifySelectionChanged()
}

func (c *Canvas) hitTestControl(pt image.Point) *vgofile.Control {
	if c.form == nil {
		return nil
	}

	offsetX := 10
	offsetY := 35

	for i := len(c.form.Controls) - 1; i >= 0; i-- {
		ctrl := c.form.Controls[i]
		rect := image.Rectangle{
			Min: image.Point{X: ctrl.Left + offsetX, Y: ctrl.Top + offsetY},
			Max: image.Point{X: ctrl.Left + offsetX + ctrl.Width, Y: ctrl.Top + offsetY + ctrl.Height},
		}
		if pt.In(rect) {
			return ctrl
		}
	}
	return nil
}

func (c *Canvas) hitTestHandles(pt image.Point) int {
	if c.selectedControl == nil {
		return -1
	}

	offsetX := 10
	offsetY := 35

	rect := walk.Rectangle{
		X:      c.selectedControl.Left + offsetX,
		Y:      c.selectedControl.Top + offsetY,
		Width:  c.selectedControl.Width,
		Height: c.selectedControl.Height,
	}

	handles := c.getHandleRects(rect)
	for i, h := range handles {
		if pt.X >= h.X && pt.X <= h.X+h.Width && pt.Y >= h.Y && pt.Y <= h.Y+h.Height {
			return i
		}
	}
	return -1
}

func (c *Canvas) hitTestForm(pt image.Point) bool {
	if c.form == nil {
		return false
	}
	formRect := image.Rectangle{
		Min: image.Point{X: 10, Y: 35},
		Max: image.Point{X: 10 + c.form.Width, Y: 35 + c.form.Height},
	}
	return pt.In(formRect)
}

func (c *Canvas) resizeControl(pt image.Point) {
	ctrl := c.selectedControl
	if ctrl == nil {
		return
	}

	offsetX := 10
	offsetY := 35

	dx := pt.X - c.dragStart.X
	dy := pt.Y - c.dragStart.Y
	c.dragStart = pt

	if c.snapToGrid {
		dx = (dx / gridSize) * gridSize
		dy = (dy / gridSize) * gridSize
	}

	left := ctrl.Left + offsetX
	top := ctrl.Top + offsetY
	right := left + ctrl.Width
	bottom := top + ctrl.Height

	switch c.resizeHandle {
	case 0:
		left += dx
		top += dy
	case 1:
		top += dy
	case 2:
		right += dx
		top += dy
	case 3:
		right += dx
	case 4:
		right += dx
		bottom += dy
	case 5:
		bottom += dy
	case 6:
		left += dx
		bottom += dy
	case 7:
		left += dx
	}

	if right-left >= minCtrlWidth {
		ctrl.Left = left - offsetX
		ctrl.Width = right - left
	}
	if bottom-top >= minCtrlHeight {
		ctrl.Top = top - offsetY
		ctrl.Height = bottom - top
	}
}

func (c *Canvas) setResizeCursor(handle int) {
	cursors := []uintptr{
		win.IDC_SIZENWSE, win.IDC_SIZENS, win.IDC_SIZENESW, win.IDC_SIZEWE,
		win.IDC_SIZENWSE, win.IDC_SIZENS, win.IDC_SIZENESW, win.IDC_SIZEWE,
	}
	if handle >= 0 && handle < len(cursors) {
		c.setCursor(cursors[handle])
	}
}

func (c *Canvas) setCursor(cursor uintptr) {
	win.SetCursor(win.LoadCursor(0, (*uint16)(unsafe.Pointer(cursor))))
}

func (c *Canvas) notifySelectionChanged() {
	if c.onSelectionChanged != nil {
		c.onSelectionChanged(c.selectedControl)
	}
}

func (c *Canvas) notifyFormSelected() {
	if c.onFormSelected != nil {
		c.onFormSelected(c.form)
	}
}
