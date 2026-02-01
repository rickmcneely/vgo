package project

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"vg/internal/ui"
	"vg/pkg/vgofile"
)

// Panel is the project explorer panel
type Panel struct {
	composite      *walk.Composite
	treeView       *walk.TreeView
	model          *ProjectModel
	currentProject *vgofile.Project
	onFormSelected func(*vgofile.Form)
}

// NewPanel creates a new project explorer panel
func NewPanel(parent *walk.Composite) (*Panel, error) {
	p := &Panel{
		composite: parent,
	}

	var tv *walk.TreeView

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
						Text:      "Project",
						Font:      Font{Family: ui.HeaderFontFamily, PointSize: ui.HeaderFontSize, Bold: true},
						TextColor: ui.HeaderTextColor,
					},
					HSpacer{},
				},
			},
			// Tree view
			TreeView{
				AssignTo: &tv,
			},
		},
	}).Create(builder); err != nil {
		return nil, err
	}

	// Double-click to select form
	tv.ItemActivated().Attach(func() {
		if item := tv.CurrentItem(); item != nil {
			if formItem, ok := item.(*FormItem); ok {
				if p.onFormSelected != nil {
					p.onFormSelected(formItem.form)
				}
			}
		}
	})

	p.treeView = tv
	return p, nil
}

// SetOnFormSelected sets the callback when a form is selected
func (p *Panel) SetOnFormSelected(fn func(*vgofile.Form)) {
	p.onFormSelected = fn
}

// SetProject sets the project to display
func (p *Panel) SetProject(proj *vgofile.Project) {
	p.currentProject = proj
	p.model = newProjectModel(proj)
	p.treeView.SetModel(p.model)

	// Expand the project node by default
	if p.model != nil && p.model.project != nil {
		p.treeView.SetExpanded(p.model.project, true)
	}
}

// RefreshProject refreshes the project tree
func (p *Panel) RefreshProject() {
	if p.currentProject != nil {
		p.model = newProjectModel(p.currentProject)
		p.treeView.SetModel(p.model)
		if p.model != nil && p.model.project != nil {
			p.treeView.SetExpanded(p.model.project, true)
		}
	}
}

// ProjectModel implements walk.TreeModel for the project
type ProjectModel struct {
	walk.TreeModelBase
	project *ProjectItem
}

func newProjectModel(proj *vgofile.Project) *ProjectModel {
	if proj == nil {
		return &ProjectModel{}
	}

	projectItem := &ProjectItem{
		name:  proj.Name,
		forms: make([]*FormItem, 0, len(proj.Forms)),
	}

	for _, form := range proj.Forms {
		formItem := &FormItem{
			form:   form,
			parent: projectItem,
		}
		projectItem.forms = append(projectItem.forms, formItem)
	}

	return &ProjectModel{
		project: projectItem,
	}
}

func (m *ProjectModel) RootCount() int {
	if m.project == nil {
		return 0
	}
	return 1
}

func (m *ProjectModel) RootAt(index int) walk.TreeItem {
	if m.project == nil || index != 0 {
		return nil
	}
	return m.project
}

// ProjectItem represents the project root in the tree
type ProjectItem struct {
	name  string
	forms []*FormItem
}

func (p *ProjectItem) Text() string {
	return p.name
}

func (p *ProjectItem) Parent() walk.TreeItem {
	return nil
}

func (p *ProjectItem) ChildCount() int {
	return len(p.forms)
}

func (p *ProjectItem) ChildAt(index int) walk.TreeItem {
	if index < 0 || index >= len(p.forms) {
		return nil
	}
	return p.forms[index]
}

func (p *ProjectItem) Image() interface{} {
	return nil
}

// FormItem represents a form in the tree
type FormItem struct {
	form   *vgofile.Form
	parent *ProjectItem
}

func (f *FormItem) Text() string {
	return f.form.Name
}

func (f *FormItem) Parent() walk.TreeItem {
	return f.parent
}

func (f *FormItem) ChildCount() int {
	return 0
}

func (f *FormItem) ChildAt(index int) walk.TreeItem {
	return nil
}

func (f *FormItem) Image() interface{} {
	return nil
}
