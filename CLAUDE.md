# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Goals

Create a (virtual clone of) VB6-style visual IDE for Go that allows developers to:
1. Design Windows GUI applications visually with drag-and-drop
2. Generate clean, maintainable Go code from visual form designs
3. Build and run projects directly from the IDE
4. Add a package browesr for go packages
## Roadmap

### Phase 1: Core Infrastructure (Complete)
- [x] Basic IDE shell with menu and toolbar
- [x] Debug/log output panel
- [x] Configuration persistence
- [x] Convert designer canvas from Qt to walk

### Phase 2: Visual Designer (Complete)
- [x] Drag-and-drop form designer with CustomWidget
- [x] Widget toolbox panel with TreeView
- [x] Property editor panel with TableView
- [x] Project explorer tree
- [x] Package browser panel

### Phase 3: Code Generation (Current)
- [ ] Generate Go code from form definitions
- [ ] Two-way binding (code changes reflect in designer)
- [ ] Build and run integration

### Phase 4: Polish
- [ ] Undo/redo support
- [x] Copy/paste widgets
- [x] Multiple form support
- [ ] Project templates

## Priorities

1. **Working software over features** - Each commit should produce a runnable application
2. **Win32 purity** - Use only lxn/walk, no Qt or other frameworks
3. **Simple code** - Avoid over-engineering; VB6 was simple, this should be too
4. **Reference existing work** - Use `/home/zditech/windsp/VisualGo/` as a guide for walk patterns

## Coding Standards and Conventions

### Go Style
- Follow standard Go conventions (gofmt, golint)
- Use descriptive names: `outputPanel` not `op`
- Keep functions short and focused
- Error handling: return errors, don't panic (except for truly unrecoverable situations)

### Package Structure
- `internal/` for all non-main packages
- One responsibility per package
- Minimize inter-package dependencies

### Walk/Win32 Patterns
- Use declarative syntax (`walk/declarative`) for initial UI setup
- Use imperative calls for runtime modifications
- Always check for nil before accessing walk widgets
- Remember splitter children include invisible handles at odd indices

### Logging
- Use `log.Printf()` for file logging (goes to vgo.log)
- Use `outputPanel.Debug/Info/Warning/Error()` for user-visible logs
- Include context in log messages: `log.Printf("ERROR loading config: %v", err)`
- Include a logging section window where the VB6 immediate section would be.
## Architecture Decisions

### Why lxn/walk?
- Pure Win32 API wrapper - no external dependencies like Qt
- Produces small, native Windows executables
- Declarative API similar to other Go UI libraries
- Active enough for our needs, well-documented

### Why not raw Win32?
- Walk provides Go-idiomatic abstractions
- Handles message loop, event binding, and common patterns
- Still produces native Win32 calls under the hood

### Configuration Storage
- JSON format for human readability
- Stored next to executable (portable)
- Separate from project files

### Old Qt Code
- Preserved in `_old_qt/` for reference during conversion
- Do not import or use directly
- Delete after all conversions complete

## Things to Avoid

1. **No Qt** - Do not use therecipe/qt or any Qt bindings
2. **No CGO complexity** - walk uses CGO minimally; don't add more
3. **No external GUI frameworks** - Stick to walk for consistency
4. **No premature optimization** - Get it working first
5. **No feature creep** - Stay focused on core VB6-style functionality
6. **No console windows** - Always use `-H windowsgui` linker flag
7. **No hardcoded paths** - Use relative paths or detect executable location

## Build Commands

```bash
# Cross-compile for Windows from Linux/WSL
GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui" -o vgeditor.exe .

# Build on Windows
go build -ldflags "-H windowsgui" -o vgeditor.exe .

# Run tests
go test ./...

# Download dependencies
go mod tidy
go mod download
```

The `-H windowsgui` linker flag prevents a console window from appearing when the application runs.

## Architecture

VG Editor is a VB6-style visual IDE for Go that generates Go code from visual form designs. It uses the `github.com/lxn/walk` Win32 wrapper library for the GUI (pure Win32 API, no Qt).

### Key Packages

- **main.go** - Application entry point with file logging
- **internal/app** - Main IDE controller, window layout, configuration persistence
- **internal/output** - Debug/log output panel with log levels and filtering
- **internal/designer/canvas** - Design canvas using walk.CustomWidget
- **internal/designer/controls** - Control registry with available widget types
- **internal/toolbox** - Control palette panel with TreeView
- **internal/properties** - Property grid panel with TableView
- **internal/project** - Project explorer tree view
- **internal/packages** - Go package browser with search
- **pkg/vgofile** - Project, Form, Control data structures with JSON serialization

### Walk Library Patterns

- **HSplitter/VSplitter**: Children include invisible handles at odd indices (0=panel, 1=handle, 2=panel, etc.)
- **Declarative API**: Use walk/declarative for building UI with declarative syntax
- **Composite**: Container widget for grouping children
- **SetFixed**: Lock splitter panel sizes

### Configuration

IDE layout is saved to `vgeditor.config` in the executable directory:
- Window position and size
- Splitter positions (panel widths/heights)
- Debug panel height

Access via View menu:
- View → Save Layout
- View → Reset Layout

## Conversion Status

The project has been fully converted from Qt to Win32 (walk library). All visual designer components are complete.

### Completed (New Win32-based)
- [x] internal/app - Main application and config
- [x] internal/output - Debug/log panel with filtering
- [x] internal/designer/canvas - Form designer with drag-and-drop
- [x] internal/designer/controls - Control registry
- [x] internal/toolbox - Control palette with TreeView
- [x] internal/properties - Property editor with TableView
- [x] internal/project - Project explorer
- [x] internal/packages - Go package browser
- [x] pkg/vgofile - Project/Form/Control data structures
- [x] main.go - Entry point with file logging
- [x] go.mod - Updated to use lxn/walk

### Old Qt Code (Moved to _old_qt/)
These packages have been preserved for reference but are no longer needed:
- _old_qt/* - All old Qt code (can be deleted)

Refer to `/home/zditech/windsp/VisualGo/` for additional walk-based patterns.

## Debug Panel Features

The debug/log panel (internal/output) provides:
- Log levels: Debug, Info, Warning, Error
- Timestamp on each message
- Filter dropdown to show only certain log levels
- Clear button to reset the log
- Adjustable height via splitter (saved in config)
