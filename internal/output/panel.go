package output

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarning
	LogError
)

// Panel is the debug/log output panel
type Panel struct {
	composite   *walk.Composite
	textEdit    *walk.TextEdit
	filterCombo *walk.ComboBox
	clearButton *walk.PushButton
	minLevel    LogLevel
}

// NewPanel creates a new debug/log output panel
func NewPanel(parent *walk.Composite) (*Panel, error) {
	p := &Panel{
		composite: parent,
		minLevel:  LogDebug,
	}

	builder := NewBuilder(parent)

	var te *walk.TextEdit
	var cb *walk.ComboBox
	var btn *walk.PushButton

	if err := (Composite{
		Layout: VBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			// Styled header row - uses same colors as ui.Header
			Composite{
				Layout:     HBox{Margins: Margins{Left: 8, Top: 4, Right: 8, Bottom: 4}},
				Background: SolidColorBrush{Color: walk.RGB(120, 120, 120)},
				Children: []Widget{
					Label{
						Text:      "Debug Log",
						Font:      Font{Family: "Segoe UI", PointSize: 9, Bold: true},
						TextColor: walk.RGB(255, 255, 255),
					},
					HSpacer{},
					Label{
						Text:      "Filter:",
						TextColor: walk.RGB(255, 255, 255),
					},
					ComboBox{
						AssignTo:     &cb,
						Model:        []string{"All", "Debug", "Info", "Warning", "Error"},
						CurrentIndex: 0,
						OnCurrentIndexChanged: func() {
							if cb != nil {
								p.minLevel = LogLevel(cb.CurrentIndex())
							}
						},
					},
					PushButton{
						AssignTo: &btn,
						Text:     "Clear",
						MaxSize:  Size{Width: 60},
						OnClicked: func() {
							p.Clear()
						},
					},
				},
			},
			// Log output area
			TextEdit{
				AssignTo: &te,
				ReadOnly: true,
				VScroll:  true,
				HScroll:  true,
				Font:     Font{Family: "Consolas", PointSize: 9},
			},
		},
	}).Create(builder); err != nil {
		return nil, err
	}

	p.textEdit = te
	p.filterCombo = cb
	p.clearButton = btn

	return p, nil
}

// Log adds a log message with the specified level
func (p *Panel) Log(level LogLevel, message string) {
	if level < p.minLevel {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	levelStr := ""

	switch level {
	case LogDebug:
		levelStr = "DEBUG"
	case LogInfo:
		levelStr = "INFO"
	case LogWarning:
		levelStr = "WARN"
	case LogError:
		levelStr = "ERROR"
	}

	// Format: [HH:MM:SS.mmm] [LEVEL] message
	formatted := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, message)
	p.AppendLine(formatted)
}

// Debug logs a debug message
func (p *Panel) Debug(message string) {
	p.Log(LogDebug, message)
}

// Info logs an info message
func (p *Panel) Info(message string) {
	p.Log(LogInfo, message)
}

// Warning logs a warning message
func (p *Panel) Warning(message string) {
	p.Log(LogWarning, message)
}

// Error logs an error message
func (p *Panel) Error(message string) {
	p.Log(LogError, message)
}

// Clear clears all log messages
func (p *Panel) Clear() {
	if p.textEdit != nil {
		p.textEdit.SetText("")
	}
}

// AppendLine appends a line to the output
func (p *Panel) AppendLine(text string) {
	if p.textEdit == nil {
		return
	}
	current := p.textEdit.Text()
	if current != "" {
		current += "\r\n"
	}
	p.textEdit.SetText(current + text)

	// Scroll to end - set selection to end and scroll caret into view
	textLen := len(p.textEdit.Text())
	p.textEdit.SetTextSelection(textLen, textLen)

	// Send EM_SCROLLCARET to ensure the caret (and thus the last line) is visible
	const EM_SCROLLCARET = 0x00B7
	win.SendMessage(p.textEdit.Handle(), EM_SCROLLCARET, 0, 0)
}

// Append appends text to the output without a newline
func (p *Panel) Append(text string) {
	if p.textEdit == nil {
		return
	}
	current := p.textEdit.Text()
	p.textEdit.SetText(current + text)
}

// GetText returns all log text
func (p *Panel) GetText() string {
	if p.textEdit == nil {
		return ""
	}
	return p.textEdit.Text()
}

// SetMinLevel sets the minimum log level to display
func (p *Panel) SetMinLevel(level LogLevel) {
	p.minLevel = level
	if p.filterCombo != nil {
		p.filterCombo.SetCurrentIndex(int(level))
	}
}

// SetVisible shows or hides the panel
func (p *Panel) SetVisible(visible bool) {
	if p.composite != nil {
		p.composite.SetVisible(visible)
	}
}

// Visible returns whether the panel is visible
func (p *Panel) Visible() bool {
	if p.composite != nil {
		return p.composite.Visible()
	}
	return false
}
