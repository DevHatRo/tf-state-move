package main

import (
	"fmt"
	"strings"

	"github.com/jroimartin/gocui"
)

const (
	mainView      = "main"
	selectionView = "selection"
)

type UI struct {
	tree          []*treeNode     // the full module/resource tree
	rows          []*treeNode     // currently visible nodes, flattened top to bottom
	selectedItems map[string]bool // resource address -> selected
	currentIndex  int
	origin        int // tracks the scroll position
	gui           *gocui.Gui
	inStatePath   string
	outStatePath  string
}

func newUI(tree []*treeNode, inPath, outPath string) *UI {
	ui := &UI{
		tree:          tree,
		selectedItems: make(map[string]bool),
		inStatePath:   inPath,
		outStatePath:  outPath,
	}
	ui.rebuildRows()
	return ui
}

// rebuildRows recomputes the visible rows after the tree's expansion state
// changes, keeping the cursor within bounds.
func (ui *UI) rebuildRows() {
	ui.rows = flattenVisible(ui.tree)
	if ui.currentIndex >= len(ui.rows) {
		ui.currentIndex = len(ui.rows) - 1
	}
	if ui.currentIndex < 0 {
		ui.currentIndex = 0
	}
}

func (ui *UI) run() error {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return err
	}
	defer g.Close()

	ui.gui = g
	g.SetManagerFunc(ui.layout)

	if err := ui.keybindings(g); err != nil {
		return err
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func (ui *UI) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Main view (left panel)
	if v, err := g.SetView(mainView, 0, 0, maxX/2-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Resources (↑/↓ or j/k: navigate, Space: select, Enter/l: expand, h: collapse, q: quit)"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		if _, err := g.SetCurrentView(mainView); err != nil {
			return err
		}
	}

	// Selection view (right panel)
	if v, err := g.SetView(selectionView, maxX/2, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Selected Resources (PgUp/PgDn: scroll, Home/End: jump)"
		v.Wrap = true
	}

	return ui.updateViews(g)
}

func (ui *UI) keybindings(g *gocui.Gui) error {
	// Navigation
	if err := g.SetKeybinding(mainView, gocui.KeyArrowUp, gocui.ModNone, ui.cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, gocui.KeyArrowDown, gocui.ModNone, ui.cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, 'k', gocui.ModNone, ui.cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, 'j', gocui.ModNone, ui.cursorDown); err != nil {
		return err
	}

	// Selection and expansion
	if err := g.SetKeybinding(mainView, gocui.KeySpace, gocui.ModNone, ui.toggleSelection); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, gocui.KeyEnter, gocui.ModNone, ui.toggleExpand); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, 'l', gocui.ModNone, ui.toggleExpand); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, 'h', gocui.ModNone, ui.collapseCurrentModule); err != nil {
		return err
	}

	// Quick movement
	if err := g.SetKeybinding(mainView, gocui.KeyHome, gocui.ModNone, ui.goToTop); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, gocui.KeyEnd, gocui.ModNone, ui.goToBottom); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, gocui.KeyPgup, gocui.ModNone, ui.pageUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(mainView, gocui.KeyPgdn, gocui.ModNone, ui.pageDown); err != nil {
		return err
	}

	// Quit
	if err := g.SetKeybinding("", 'q', gocui.ModNone, ui.quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, ui.quit); err != nil {
		return err
	}

	return nil
}

func (ui *UI) updateViews(g *gocui.Gui) error {
	if err := ui.updateMainView(g); err != nil {
		return err
	}
	return ui.updateSelectionView(g)
}

func (ui *UI) updateMainView(g *gocui.Gui) error {
	v, err := g.View(mainView)
	if err != nil {
		return err
	}
	v.Clear()

	_, maxY := v.Size()
	rowCount := len(ui.rows)

	// Adjust the scroll origin so the cursor stays on screen.
	if ui.currentIndex-ui.origin >= maxY {
		ui.origin = ui.currentIndex - maxY + 1
	}
	if ui.currentIndex < ui.origin {
		ui.origin = ui.currentIndex
	}

	endIndex := ui.origin + maxY
	if endIndex > rowCount {
		endIndex = rowCount
	}

	for _, node := range ui.rows[ui.origin:endIndex] {
		prefix := strings.Repeat("  ", node.level)
		switch {
		case node.isModule && node.expanded:
			prefix += "[-]"
		case node.isModule:
			prefix += "[+]"
		default:
			prefix += "   "
		}

		check := "[ ]"
		if nodeSelected(node, ui.selectedItems) {
			check = "[✓]"
		}

		display := node.label
		if node.isModule {
			display = fmt.Sprintf("%s (%d)", node.label, resourceCount(node))
		}

		if _, err := fmt.Fprintf(v, "%s %s %s\n", prefix, check, display); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	// Place the cursor relative to the scroll origin.
	if err := v.SetCursor(0, ui.currentIndex-ui.origin); err != nil {
		if err != gocui.ErrUnknownView {
			return fmt.Errorf("failed to set cursor: %w", err)
		}
	}
	return nil
}

func (ui *UI) updateSelectionView(g *gocui.Gui) error {
	v, err := g.View(selectionView)
	if err != nil {
		return err
	}
	v.Clear()

	paths := selectedResourcePaths(ui.selectedItems)

	if _, err := fmt.Fprintf(v, "Selected: %d resource(s)\n\n", len(paths)); err != nil {
		return fmt.Errorf("failed to write count: %w", err)
	}
	for _, p := range paths {
		if _, err := fmt.Fprintf(v, "• %s\n", p); err != nil {
			return fmt.Errorf("failed to write resource: %w", err)
		}
	}
	return nil
}

func (ui *UI) quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// currentNode returns the tree node under the cursor, or nil when there are no
// rows.
func (ui *UI) currentNode() *treeNode {
	if ui.currentIndex < 0 || ui.currentIndex >= len(ui.rows) {
		return nil
	}
	return ui.rows[ui.currentIndex]
}

func (ui *UI) cursorDown(g *gocui.Gui, v *gocui.View) error {
	if ui.currentIndex < len(ui.rows)-1 {
		ui.currentIndex++
	}
	return ui.updateViews(g)
}

func (ui *UI) cursorUp(g *gocui.Gui, v *gocui.View) error {
	if ui.currentIndex > 0 {
		ui.currentIndex--
	}
	return ui.updateViews(g)
}

// toggleSelection flips the cursor row. For a module it selects or deselects
// every resource leaf beneath it, at any depth.
func (ui *UI) toggleSelection(g *gocui.Gui, v *gocui.View) error {
	node := ui.currentNode()
	if node == nil {
		return nil
	}

	newState := !nodeSelected(node, ui.selectedItems)
	for _, leaf := range leafValues(node) {
		ui.selectedItems[leaf] = newState
	}
	return ui.updateViews(g)
}

// toggleExpand expands or collapses the module under the cursor.
func (ui *UI) toggleExpand(g *gocui.Gui, v *gocui.View) error {
	node := ui.currentNode()
	if node == nil || !node.isModule {
		return nil
	}

	node.expanded = !node.expanded
	ui.rebuildRows()
	return ui.updateViews(g)
}

// collapseCurrentModule collapses the module under the cursor, or, when the
// cursor is not on an expanded module, the nearest expanded ancestor module.
func (ui *UI) collapseCurrentModule(g *gocui.Gui, v *gocui.View) error {
	node := ui.currentNode()
	if node == nil {
		return nil
	}

	if node.isModule && node.expanded {
		node.expanded = false
		ui.rebuildRows()
		return ui.updateViews(g)
	}

	for i := ui.currentIndex - 1; i >= 0; i-- {
		ancestor := ui.rows[i]
		if ancestor.isModule && ancestor.expanded && ancestor.level < node.level {
			ancestor.expanded = false
			ui.currentIndex = i
			ui.rebuildRows()
			return ui.updateViews(g)
		}
	}
	return nil
}

func (ui *UI) goToTop(g *gocui.Gui, v *gocui.View) error {
	ui.currentIndex = 0
	ui.origin = 0
	return ui.updateViews(g)
}

func (ui *UI) goToBottom(g *gocui.Gui, v *gocui.View) error {
	ui.currentIndex = len(ui.rows) - 1
	if ui.currentIndex < 0 {
		ui.currentIndex = 0
	}
	return ui.updateViews(g)
}

func (ui *UI) pageUp(g *gocui.Gui, v *gocui.View) error {
	_, maxY := v.Size()
	ui.currentIndex -= maxY
	if ui.currentIndex < 0 {
		ui.currentIndex = 0
	}
	return ui.updateViews(g)
}

func (ui *UI) pageDown(g *gocui.Gui, v *gocui.View) error {
	_, maxY := v.Size()
	ui.currentIndex += maxY
	if ui.currentIndex >= len(ui.rows) {
		ui.currentIndex = len(ui.rows) - 1
	}
	if ui.currentIndex < 0 {
		ui.currentIndex = 0
	}
	return ui.updateViews(g)
}
