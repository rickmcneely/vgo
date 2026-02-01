package packages

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// Package represents a Go package with extended info
type Package struct {
	Name       string   `json:"Name"`
	ImportPath string   `json:"ImportPath"`
	Dir        string   `json:"Dir"`
	Doc        string   `json:"Doc"`
	GoFiles    []string `json:"GoFiles"`
	Imports    []string `json:"Imports"`
	Standard   bool     `json:"Standard"`
}

// PackageDetails contains parsed documentation
type PackageDetails struct {
	Summary   string
	Variables []string
	Functions []string
	Types     []string
}

// BrowserDialog is a dialog for browsing Go packages
type BrowserDialog struct {
	dialog      *walk.Dialog
	searchEdit  *walk.LineEdit
	listBox     *walk.ListBox
	detailsText *walk.TextEdit
	packages    []*Package
	filtered    []*Package
	selected    *Package
	onSelected  func(*Package)
	hsplitter   *walk.Splitter
}

// hideConsoleCmd configures a command to hide the console window on Windows
func hideConsoleCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// ShowBrowserDialog shows the package browser dialog
func ShowBrowserDialog(owner walk.Form, onSelected func(*Package)) error {
	b := &BrowserDialog{
		packages:   make([]*Package, 0),
		filtered:   make([]*Package, 0),
		onSelected: onSelected,
	}

	var dlg *walk.Dialog
	var searchEdit *walk.LineEdit
	var listBox *walk.ListBox
	var detailsText *walk.TextEdit
	var hsplitter *walk.Splitter

	err := Dialog{
		AssignTo: &dlg,
		Title:    "Go Package Browser",
		MinSize:  Size{Width: 800, Height: 600},
		Size:     Size{Width: 1000, Height: 700},
		Layout:   VBox{},
		Children: []Widget{
			// Search bar
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: "Search:"},
					LineEdit{
						AssignTo:  &searchEdit,
						CueBanner: "Type to filter packages...",
						OnTextChanged: func() {
							b.search(searchEdit.Text())
						},
					},
				},
			},
			// Main content: list + details
			HSplitter{
				AssignTo: &hsplitter,
				Children: []Widget{
					// Package list (narrow)
					ListBox{
						AssignTo: &listBox,
						OnCurrentIndexChanged: func() {
							idx := listBox.CurrentIndex()
							if idx >= 0 && idx < len(b.filtered) {
								b.showDetails(b.filtered[idx])
							}
						},
						OnItemActivated: func() {
							// Double-click to select and close
							if b.selected != nil && b.onSelected != nil {
								b.onSelected(b.selected)
								dlg.Accept()
							}
						},
					},
					// Details panel (wide)
					Composite{
						Layout: VBox{},
						Children: []Widget{
							Label{
								Text: "Package Details",
								Font: Font{Family: "Segoe UI", PointSize: 9, Bold: true},
							},
							TextEdit{
								AssignTo: &detailsText,
								ReadOnly: true,
								VScroll:  true,
								HScroll:  true,
								Font:     Font{Family: "Consolas", PointSize: 10},
							},
						},
					},
				},
			},
			// Buttons
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Insert Import",
						OnClicked: func() {
							if b.selected != nil && b.onSelected != nil {
								b.onSelected(b.selected)
							}
							dlg.Accept()
						},
					},
					PushButton{
						Text: "Close",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Create(owner)

	if err != nil {
		return err
	}

	b.dialog = dlg
	b.searchEdit = searchEdit
	b.listBox = listBox
	b.detailsText = detailsText
	b.hsplitter = hsplitter

	// Set splitter sizes: narrow left (200px), wide right (rest)
	if hsplitter != nil && hsplitter.Children().Len() >= 1 {
		hsplitter.Children().At(0).SetMinMaxSize(walk.Size{Width: 200, Height: 0}, walk.Size{Width: 250, Height: 0})
	}

	// Set initial details text
	b.detailsText.SetText("Select a package to view details.\r\n\r\nLoading packages...")

	// Load packages in background
	go b.loadPackages()

	dlg.Run()
	return nil
}

func (b *BrowserDialog) loadPackages() {
	// Load standard library packages
	cmd := exec.Command("go", "list", "-json", "std")
	hideConsoleCmd(cmd)
	output, err := cmd.Output()
	if err != nil {
		b.listBox.Synchronize(func() {
			b.detailsText.SetText("Error loading packages: " + err.Error())
		})
		return
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		var pkg Package
		if err := decoder.Decode(&pkg); err != nil {
			continue
		}
		pkg.Standard = true
		b.packages = append(b.packages, &pkg)
	}

	// Update UI
	b.listBox.Synchronize(func() {
		b.filtered = b.packages
		b.updateList()
		b.detailsText.SetText(fmt.Sprintf(
			"Select a package to view details.\r\n\r\n"+
				"Loaded %d standard library packages.\r\n\r\n"+
				"Click a package to see its summary, variables, and functions.\r\n"+
				"Double-click to insert import statement.",
			len(b.packages)))
	})
}

func (b *BrowserDialog) search(query string) {
	if query == "" {
		b.filtered = b.packages
		b.updateList()
		return
	}

	query = strings.ToLower(query)
	b.filtered = make([]*Package, 0)
	for _, pkg := range b.packages {
		if strings.Contains(strings.ToLower(pkg.ImportPath), query) ||
			strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.Doc), query) {
			b.filtered = append(b.filtered, pkg)
		}
	}
	b.updateList()
}

func (b *BrowserDialog) updateList() {
	items := make([]string, len(b.filtered))
	for i, pkg := range b.filtered {
		items[i] = pkg.ImportPath
	}
	b.listBox.SetModel(items)
}

func (b *BrowserDialog) showDetails(pkg *Package) {
	b.selected = pkg

	// Show loading message
	b.detailsText.SetText("Loading documentation for " + pkg.ImportPath + "...")

	// Fetch documentation in background
	go func() {
		details := b.fetchPackageDetails(pkg.ImportPath)

		b.detailsText.Synchronize(func() {
			var sb strings.Builder

			// Package name and import path
			sb.WriteString("PACKAGE\r\n")
			sb.WriteString("═══════════════════════════════════════════════════════════════\r\n")
			sb.WriteString(fmt.Sprintf("import \"%s\"\r\n\r\n", pkg.ImportPath))

			// Summary
			if details.Summary != "" {
				sb.WriteString("SUMMARY\r\n")
				sb.WriteString("───────────────────────────────────────────────────────────────\r\n")
				summary := strings.ReplaceAll(details.Summary, "\n", "\r\n")
				sb.WriteString(summary)
				sb.WriteString("\r\n\r\n")
			}

			// Variables
			if len(details.Variables) > 0 {
				sb.WriteString("VARIABLES\r\n")
				sb.WriteString("───────────────────────────────────────────────────────────────\r\n")
				for _, v := range details.Variables {
					sb.WriteString(v)
					sb.WriteString("\r\n")
				}
				sb.WriteString("\r\n")
			}

			// Types
			if len(details.Types) > 0 {
				sb.WriteString("TYPES\r\n")
				sb.WriteString("───────────────────────────────────────────────────────────────\r\n")
				for _, t := range details.Types {
					sb.WriteString(t)
					sb.WriteString("\r\n")
				}
				sb.WriteString("\r\n")
			}

			// Functions
			if len(details.Functions) > 0 {
				sb.WriteString("FUNCTIONS\r\n")
				sb.WriteString("───────────────────────────────────────────────────────────────\r\n")
				for _, f := range details.Functions {
					sb.WriteString(f)
					sb.WriteString("\r\n")
				}
				sb.WriteString("\r\n")
			}

			// Footer
			sb.WriteString("═══════════════════════════════════════════════════════════════\r\n")
			sb.WriteString("Double-click or press 'Insert Import' to add:\r\n")
			sb.WriteString(fmt.Sprintf("    import \"%s\"\r\n", pkg.ImportPath))

			b.detailsText.SetText(sb.String())
		})
	}()
}

func (b *BrowserDialog) fetchPackageDetails(importPath string) PackageDetails {
	details := PackageDetails{
		Variables: make([]string, 0),
		Functions: make([]string, 0),
		Types:     make([]string, 0),
	}

	// Run go doc to get package documentation
	cmd := exec.Command("go", "doc", "-short", importPath)
	hideConsoleCmd(cmd)
	output, err := cmd.Output()
	if err != nil {
		details.Summary = "Unable to load documentation: " + err.Error()
		return details
	}

	lines := strings.Split(string(output), "\n")

	// Parse the output
	inSummary := true
	summaryLines := make([]string, 0)

	// Regex patterns for detecting declarations
	varPattern := regexp.MustCompile(`^var\s+`)
	constPattern := regexp.MustCompile(`^const\s+`)
	funcPattern := regexp.MustCompile(`^func\s+`)
	typePattern := regexp.MustCompile(`^type\s+`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inSummary && len(summaryLines) > 0 {
				inSummary = false
			}
			continue
		}

		if varPattern.MatchString(trimmed) || constPattern.MatchString(trimmed) {
			inSummary = false
			details.Variables = append(details.Variables, trimmed)
		} else if funcPattern.MatchString(trimmed) {
			inSummary = false
			details.Functions = append(details.Functions, trimmed)
		} else if typePattern.MatchString(trimmed) {
			inSummary = false
			details.Types = append(details.Types, trimmed)
		} else if inSummary {
			summaryLines = append(summaryLines, trimmed)
		}
	}

	details.Summary = strings.Join(summaryLines, "\r\n")

	// If we didn't get much, try without -short for more detail
	if len(details.Functions) == 0 && len(details.Variables) == 0 && len(details.Types) == 0 {
		cmd = exec.Command("go", "doc", importPath)
		hideConsoleCmd(cmd)
		output, err = cmd.Output()
		if err == nil {
			lines = strings.Split(string(output), "\n")
			inSummary = true
			summaryLines = make([]string, 0)

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					if inSummary && len(summaryLines) > 0 {
						inSummary = false
					}
					continue
				}

				if varPattern.MatchString(trimmed) || constPattern.MatchString(trimmed) {
					inSummary = false
					details.Variables = append(details.Variables, trimmed)
				} else if funcPattern.MatchString(trimmed) {
					inSummary = false
					details.Functions = append(details.Functions, trimmed)
				} else if typePattern.MatchString(trimmed) {
					inSummary = false
					details.Types = append(details.Types, trimmed)
				} else if inSummary {
					summaryLines = append(summaryLines, trimmed)
				}
			}

			if details.Summary == "" || len(summaryLines) > 0 {
				details.Summary = strings.Join(summaryLines, "\r\n")
			}
		}
	}

	return details
}
