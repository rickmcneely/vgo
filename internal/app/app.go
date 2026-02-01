package app

import (
	"fmt"
	"log"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"vg/internal/designer/canvas"
	"vg/internal/designer/controls"
	"vg/internal/output"
	"vg/internal/packages"
	"vg/internal/project"
	"vg/internal/properties"
	"vg/internal/toolbox"
	"vg/pkg/vgofile"
)

// App represents the main IDE application
type App struct {
	mainWindow     *walk.MainWindow
	toolboxPanel   *toolbox.Panel
	designerCanvas *canvas.Canvas
	propsPanel     *properties.Panel
	projectPanel   *project.Panel
	outputPanel    *output.Panel
	currentProject *vgofile.Project
	currentForm    *vgofile.Form
	registry       *controls.Registry
	config         *IDEConfig
	mainSplitter   *walk.Splitter
	leftSplitter   *walk.Splitter
	centerSplitter *walk.Splitter
}

// NewApp creates a new IDE application instance
func NewApp() *App {
	// Set up walk's settings for persistence
	settings := walk.NewIniFileSettings("vgeditor.ini")
	settings.SetPortable(true) // Save in exe directory, not AppData
	if err := settings.Load(); err != nil {
		log.Printf("Could not load settings: %v", err)
	}
	walk.App().SetSettings(settings)

	return &App{
		registry: controls.NewRegistry(),
		config:   LoadConfig(),
	}
}

// Run starts the IDE application
func Run() error {
	app := NewApp()
	return app.run()
}

func (a *App) run() error {
	var mainWindow *walk.MainWindow
	var mainSplitter *walk.Splitter
	var leftSplitter *walk.Splitter
	var centerSplitter *walk.Splitter
	var toolboxContainer *walk.Composite
	var designerContainer *walk.Composite
	var outputContainer *walk.Composite
	var propsContainer *walk.Composite
	var projectContainer *walk.Composite
	var statusBar *walk.StatusBarItem

	if err := (MainWindow{
		AssignTo: &mainWindow,
		Name:     "mainWindow",
		Title:    "VG Editor - Visual Go Designer",
		MinSize:  Size{Width: 800, Height: 600},
		Size:     Size{Width: a.config.WindowWidth, Height: a.config.WindowHeight},
		Layout:   VBox{MarginsZero: true, SpacingZero: true},
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Action{
						Text:        "&New Project...",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyN},
						OnTriggered: a.onNewProject,
					},
					Action{
						Text:        "&Open Project...",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyO},
						OnTriggered: a.onOpenProject,
					},
					Separator{},
					Action{
						Text:        "&Save",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyS},
						OnTriggered: a.onSave,
					},
					Action{
						Text:        "Save &As...",
						OnTriggered: a.onSaveAs,
					},
					Separator{},
					Action{
						Text:        "E&xit",
						OnTriggered: func() { mainWindow.Close() },
					},
				},
			},
			Menu{
				Text: "&Edit",
				Items: []MenuItem{
					Action{
						Text:        "&Undo",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyZ},
						OnTriggered: a.onUndo,
					},
					Action{
						Text:        "&Redo",
						Shortcut:    Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyZ},
						OnTriggered: a.onRedo,
					},
					Separator{},
					Action{
						Text:        "Cu&t",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyX},
						OnTriggered: a.onCut,
					},
					Action{
						Text:        "&Copy",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyC},
						OnTriggered: a.onCopy,
					},
					Action{
						Text:        "&Paste",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyV},
						OnTriggered: a.onPaste,
					},
					Action{
						Text:        "&Delete",
						Shortcut:    Shortcut{Key: walk.KeyDelete},
						OnTriggered: a.onDelete,
					},
				},
			},
			Menu{
				Text: "&View",
				Items: []MenuItem{
					Action{
						Text:        "&Debug Log",
						OnTriggered: a.onToggleDebugLog,
					},
					Action{
						Text:        "&Package Browser...",
						Shortcut:    Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyP},
						OnTriggered: a.onPackageBrowser,
					},
					Separator{},
					Action{
						Text:        "Save &Layout",
						OnTriggered: a.onSaveLayout,
					},
					Action{
						Text:        "&Reset Layout",
						OnTriggered: a.onResetLayout,
					},
				},
			},
			Menu{
				Text: "&Project",
				Items: []MenuItem{
					Action{
						Text:        "&Add Form...",
						OnTriggered: a.onAddForm,
					},
					Separator{},
					Action{
						Text:        "&Generate Code",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyG},
						OnTriggered: a.onGenerateCode,
					},
					Separator{},
					Action{
						Text:        "&Build",
						Shortcut:    Shortcut{Key: walk.KeyF5},
						OnTriggered: a.onBuild,
					},
					Action{
						Text:        "&Run",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyF5},
						OnTriggered: a.onRun,
					},
				},
			},
			Menu{
				Text: "&Help",
				Items: []MenuItem{
					Action{
						Text:        "&About VG Editor",
						OnTriggered: a.onAbout,
					},
				},
			},
		},
		ToolBar: ToolBar{
			ButtonStyle: ToolBarButtonImageBeforeText,
			Items: []MenuItem{
				Action{Text: "New", OnTriggered: a.onNewProject},
				Action{Text: "Open", OnTriggered: a.onOpenProject},
				Action{Text: "Save", OnTriggered: a.onSave},
				Separator{},
				Action{Text: "Undo", OnTriggered: a.onUndo},
				Action{Text: "Redo", OnTriggered: a.onRedo},
				Separator{},
				Action{Text: "Build", OnTriggered: a.onBuild},
				Action{Text: "Run", OnTriggered: a.onRun},
			},
		},
		Children: []Widget{
			HSplitter{
				AssignTo: &mainSplitter,
				Name:     "mainSplitter",
				Children: []Widget{
					// Left panel: Project Explorer + Toolbox
					VSplitter{
						AssignTo: &leftSplitter,
						Name:     "leftSplitter",
						Children: []Widget{
							Composite{
								AssignTo: &projectContainer,
								Layout:   VBox{MarginsZero: true, SpacingZero: true},
							},
							Composite{
								AssignTo: &toolboxContainer,
								Layout:   VBox{MarginsZero: true, SpacingZero: true},
							},
						},
					},
					// Center: Designer + Output
					VSplitter{
						AssignTo: &centerSplitter,
						Name:     "centerSplitter",
						Children: []Widget{
							Composite{
								Layout: VBox{MarginsZero: true, SpacingZero: true},
								Children: []Widget{
									// Styled header
									Composite{
										Layout:     HBox{Margins: Margins{Left: 8, Top: 4, Right: 8, Bottom: 4}},
										Background: SolidColorBrush{Color: walk.RGB(120, 120, 120)},
										Children: []Widget{
											Label{
												Text:      "Forms",
												Font:      Font{Family: "Segoe UI", PointSize: 9, Bold: true},
												TextColor: walk.RGB(255, 255, 255),
											},
											HSpacer{},
										},
									},
									Composite{
										AssignTo:      &designerContainer,
										Layout:        VBox{Margins: Margins{Left: 4, Top: 4, Right: 4, Bottom: 4}},
										StretchFactor: 1,
									},
								},
							},
							Composite{
								AssignTo: &outputContainer,
								Layout:   VBox{MarginsZero: true, SpacingZero: true},
							},
						},
					},
					// Right panel: Properties
					Composite{
						AssignTo: &propsContainer,
						Layout:   VBox{MarginsZero: true, SpacingZero: true},
					},
				},
			},
		},
		StatusBarItems: []StatusBarItem{
			{
				AssignTo: &statusBar,
				Text:     "Ready",
				Width:    0,
			},
		},
	}).Create(); err != nil {
		return fmt.Errorf("creating main window: %w", err)
	}

	log.Println("Main window created")
	a.mainWindow = mainWindow
	a.mainSplitter = mainSplitter
	a.leftSplitter = leftSplitter
	a.centerSplitter = centerSplitter

	// Initialize panels
	log.Println("Initializing panels...")
	if err := a.initializePanels(toolboxContainer, designerContainer, outputContainer, propsContainer, projectContainer); err != nil {
		return fmt.Errorf("initializing panels: %w", err)
	}
	log.Println("Panels initialized")

	// Apply saved window position/size
	if a.config.WindowX > 0 || a.config.WindowY > 0 {
		mainWindow.SetBounds(walk.Rectangle{
			X:      a.config.WindowX,
			Y:      a.config.WindowY,
			Width:  a.config.WindowWidth,
			Height: a.config.WindowHeight,
		})
	}

	// Create a default new project
	log.Println("Creating default project...")
	a.createDefaultProject()
	log.Println("Default project created")

	// Log startup
	a.outputPanel.Info("VG Editor started")

	// Apply splitter layout after window is fully shown
	// Use Starting event which fires after window is visible
	mainWindow.Starting().Attach(func() {
		a.applyLayout()
	})

	log.Println("Starting main window message loop...")
	mainWindow.Run()
	log.Println("Main window closed")
	return nil
}

func (a *App) initializePanels(toolboxContainer, designerContainer, outputContainer, propsContainer, projectContainer *walk.Composite) error {
	var err error

	// Initialize project panel
	log.Println("Creating project panel...")
	a.projectPanel, err = project.NewPanel(projectContainer)
	if err != nil {
		return fmt.Errorf("project panel: %w", err)
	}
	a.projectPanel.SetOnFormSelected(a.onFormSelected)
	log.Println("Project panel created")

	// Initialize toolbox
	log.Println("Creating toolbox panel...")
	a.toolboxPanel, err = toolbox.NewPanel(toolboxContainer, a.registry)
	if err != nil {
		return fmt.Errorf("toolbox panel: %w", err)
	}
	a.toolboxPanel.SetOnControlSelected(a.onToolboxControlSelected)
	log.Println("Toolbox panel created")

	// Initialize properties panel
	log.Println("Creating properties panel...")
	a.propsPanel, err = properties.NewPanel(propsContainer)
	if err != nil {
		return fmt.Errorf("properties panel: %w", err)
	}
	a.propsPanel.SetOnPropertyChanged(a.onPropertyChanged)
	a.propsPanel.SetOnFormPropertyChanged(a.onFormPropertyChanged)
	log.Println("Properties panel created")

	// Initialize output panel
	log.Println("Creating output panel...")
	a.outputPanel, err = output.NewPanel(outputContainer)
	if err != nil {
		return fmt.Errorf("output panel: %w", err)
	}
	log.Println("Output panel created")

	// Initialize designer canvas
	log.Println("Creating designer canvas...")
	a.designerCanvas, err = canvas.NewCanvas(designerContainer, a.registry)
	if err != nil {
		return fmt.Errorf("designer canvas: %w", err)
	}
	a.designerCanvas.SetOnSelectionChanged(a.onDesignerSelectionChanged)
	a.designerCanvas.SetOnControlModified(a.onControlModified)
	a.designerCanvas.SetOnFormSelected(a.onDesignerFormSelected)
	log.Println("Designer canvas created")

	return nil
}

func (a *App) applyLayout() {
	log.Println("ApplyLayout: restoring splitter states...")

	// Use walk's built-in RestoreState for splitters
	if a.mainSplitter != nil {
		if err := a.mainSplitter.RestoreState(); err != nil {
			log.Printf("Could not restore mainSplitter state: %v", err)
		}
	}

	if a.leftSplitter != nil {
		if err := a.leftSplitter.RestoreState(); err != nil {
			log.Printf("Could not restore leftSplitter state: %v", err)
		}
	}

	if a.centerSplitter != nil {
		if err := a.centerSplitter.RestoreState(); err != nil {
			log.Printf("Could not restore centerSplitter state: %v", err)
		}
	}

	log.Println("ApplyLayout: done")
}

func (a *App) createDefaultProject() {
	a.currentProject = vgofile.NewProject("NewProject")
	a.currentForm = vgofile.NewForm("Form1")
	a.currentProject.AddForm(a.currentForm)

	a.projectPanel.SetProject(a.currentProject)
	a.designerCanvas.SetForm(a.currentForm)
}

// Event handlers
func (a *App) onToolboxControlSelected(controlType string) {
	a.designerCanvas.SetPlacementMode(controlType)
	a.outputPanel.Debug("Selected control: " + controlType)
}

func (a *App) onDesignerSelectionChanged(ctrl *vgofile.Control) {
	if a.propsPanel != nil {
		a.propsPanel.SetControl(ctrl)
	}
	if ctrl != nil {
		a.outputPanel.Debug("Selected: " + ctrl.Name)
	}
}

func (a *App) onDesignerFormSelected(form *vgofile.Form) {
	if a.propsPanel != nil {
		a.propsPanel.SetForm(form)
	}
}

func (a *App) onControlModified(ctrl *vgofile.Control) {
	a.propsPanel.RefreshControl(ctrl)
}

func (a *App) onPropertyChanged(ctrl *vgofile.Control, propName string, value interface{}) {
	if ctrl != nil {
		ctrl.SetProperty(propName, value)
		a.designerCanvas.RefreshControl(ctrl)
	}
}

func (a *App) onFormPropertyChanged(form *vgofile.Form, propName string, value interface{}) {
	if form == nil {
		return
	}
	switch propName {
	case "Name":
		if s, ok := value.(string); ok {
			form.Name = s
		}
	case "Text":
		if s, ok := value.(string); ok {
			form.Text = s
		}
	case "Width":
		if i, ok := value.(int); ok {
			form.Width = i
		}
	case "Height":
		if i, ok := value.(int); ok {
			form.Height = i
		}
	}
	a.designerCanvas.Refresh()
	a.projectPanel.RefreshProject()
}

func (a *App) onFormSelected(form *vgofile.Form) {
	a.currentForm = form
	a.designerCanvas.SetForm(form)
}

// Menu handlers
func (a *App) onNewProject() {
	a.createDefaultProject()
	a.outputPanel.Info("New project created")
}

func (a *App) onOpenProject() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Open Project"
	dlg.Filter = "VG Projects (*.vgo)|*.vgo|All Files (*.*)|*.*"

	if ok, _ := dlg.ShowOpen(a.mainWindow); ok {
		proj, err := vgofile.LoadProject(dlg.FilePath)
		if err != nil {
			walk.MsgBox(a.mainWindow, "Error", err.Error(), walk.MsgBoxIconError)
			return
		}
		a.currentProject = proj
		a.projectPanel.SetProject(proj)
		if len(proj.Forms) > 0 {
			a.onFormSelected(proj.Forms[0])
		}
		a.outputPanel.Info("Opened project: " + dlg.FilePath)
	}
}

func (a *App) onSave() {
	if a.currentProject == nil {
		return
	}
	if a.currentProject.FilePath == "" {
		a.onSaveAs()
		return
	}
	if err := a.currentProject.Save(); err != nil {
		walk.MsgBox(a.mainWindow, "Error", err.Error(), walk.MsgBoxIconError)
	} else {
		a.outputPanel.Info("Project saved")
	}
}

func (a *App) onSaveAs() {
	if a.currentProject == nil {
		return
	}
	dlg := new(walk.FileDialog)
	dlg.Title = "Save Project As"
	dlg.Filter = "VG Projects (*.vgo)|*.vgo"

	if ok, _ := dlg.ShowSave(a.mainWindow); ok {
		a.currentProject.FilePath = dlg.FilePath
		if err := a.currentProject.Save(); err != nil {
			walk.MsgBox(a.mainWindow, "Error", err.Error(), walk.MsgBoxIconError)
		} else {
			a.outputPanel.Info("Project saved as: " + dlg.FilePath)
		}
	}
}

func (a *App) onUndo() {
	a.outputPanel.Debug("Undo")
}

func (a *App) onRedo() {
	a.outputPanel.Debug("Redo")
}

func (a *App) onCut() {
	a.designerCanvas.CutSelected()
}

func (a *App) onCopy() {
	a.designerCanvas.CopySelected()
}

func (a *App) onPaste() {
	a.designerCanvas.Paste()
}

func (a *App) onDelete() {
	a.designerCanvas.DeleteSelected()
}

func (a *App) onToggleDebugLog() {
	if a.outputPanel != nil {
		a.outputPanel.SetVisible(!a.outputPanel.Visible())
	}
}

func (a *App) onPackageBrowser() {
	packages.ShowBrowserDialog(a.mainWindow, func(pkg *packages.Package) {
		if pkg != nil {
			a.outputPanel.Info("Selected package: " + pkg.ImportPath)
		}
	})
}

func (a *App) onAddForm() {
	if a.currentProject == nil {
		return
	}
	formNum := len(a.currentProject.Forms) + 1
	form := vgofile.NewForm(fmt.Sprintf("Form%d", formNum))
	a.currentProject.AddForm(form)
	a.projectPanel.RefreshProject()
	a.onFormSelected(form)
	a.outputPanel.Info("Added new form: " + form.Name)
}

func (a *App) onGenerateCode() {
	if a.currentProject == nil {
		return
	}
	a.outputPanel.Info("Generating code...")
	walk.MsgBox(a.mainWindow, "Generate Code", "Code generation not yet implemented.", walk.MsgBoxIconInformation)
}

func (a *App) onBuild() {
	a.outputPanel.Info("Building project...")
	walk.MsgBox(a.mainWindow, "Build", "Build not yet implemented.", walk.MsgBoxIconInformation)
}

func (a *App) onRun() {
	a.outputPanel.Info("Running project...")
	walk.MsgBox(a.mainWindow, "Run", "Run not yet implemented.", walk.MsgBoxIconInformation)
}

func (a *App) onAbout() {
	walk.MsgBox(a.mainWindow, "About VG Editor",
		"VG Editor - Visual Go Designer\n\nVersion 1.0\n\nA VB6-style visual IDE for creating Go applications using Win32 API.",
		walk.MsgBoxIconInformation)
}

func (a *App) onSaveLayout() {
	// Save splitter states using walk's built-in persistence
	if a.mainSplitter != nil {
		if err := a.mainSplitter.SaveState(); err != nil {
			log.Printf("Could not save mainSplitter state: %v", err)
		}
	}

	if a.leftSplitter != nil {
		if err := a.leftSplitter.SaveState(); err != nil {
			log.Printf("Could not save leftSplitter state: %v", err)
		}
	}

	if a.centerSplitter != nil {
		if err := a.centerSplitter.SaveState(); err != nil {
			log.Printf("Could not save centerSplitter state: %v", err)
		}
	}

	// Save walk settings to file
	if settings := walk.App().Settings(); settings != nil {
		if iniSettings, ok := settings.(*walk.IniFileSettings); ok {
			if err := iniSettings.Save(); err != nil {
				walk.MsgBox(a.mainWindow, "Error", "Failed to save layout: "+err.Error(), walk.MsgBoxIconError)
				return
			}
		}
	}

	a.outputPanel.Info("Layout saved")
	walk.MsgBox(a.mainWindow, "Layout Saved", "IDE layout has been saved.", walk.MsgBoxIconInformation)
}

func (a *App) onResetLayout() {
	// Clear splitter settings by removing their keys
	if settings := walk.App().Settings(); settings != nil {
		settings.Remove("mainSplitter")
		settings.Remove("leftSplitter")
		settings.Remove("centerSplitter")

		if iniSettings, ok := settings.(*walk.IniFileSettings); ok {
			iniSettings.Save()
		}
	}

	walk.MsgBox(a.mainWindow, "Layout Reset", "Layout has been reset to defaults. Restart VG Editor to apply.", walk.MsgBoxIconInformation)
}

// OutputPanel returns the debug/log output panel
func (a *App) OutputPanel() *output.Panel {
	return a.outputPanel
}
