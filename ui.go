package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jroimartin/gocui"
)

const (
	mainView      = "main"
	selectionView = "selection"
)

type UI struct {
	items         []SelectionItem
	selectedItems map[string]bool
	currentIndex  int
	origin        int // Add this field to track scroll position
	gui           *gocui.Gui
	inStatePath   string
	outStatePath  string
}

func newUI(choices []SelectionItem, inPath, outPath string) *UI {
	return &UI{
		items:         choices,
		selectedItems: make(map[string]bool),
		currentIndex:  0,
		origin:        0,
		inStatePath:   inPath,
		outStatePath:  outPath,
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
	itemCount := len(ui.items)

	// Adjust origin if cursor moves out of view
	if ui.currentIndex-ui.origin >= maxY {
		ui.origin = ui.currentIndex - maxY + 1
	}
	if ui.currentIndex < ui.origin {
		ui.origin = ui.currentIndex
	}

	// Display only visible items
	endIndex := ui.origin + maxY
	if endIndex > itemCount {
		endIndex = itemCount
	}

	for _, item := range ui.items[ui.origin:endIndex] {
		prefix := strings.Repeat("  ", item.Level)
		if item.IsModule {
			if item.IsExpanded {
				prefix += "[-]"
			} else {
				prefix += "[+]"
			}
		} else {
			prefix += "    "
		}

		selected := ""
		if item.IsSelected {
			selected = "[✓]"
		} else {
			selected = "[ ]"
		}

		line := prefix + " " + selected + " " + item.Display
		if _, err := fmt.Fprintln(v, line); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	// Set cursor relative to origin
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

	// Count selected resources (excluding grouping items)
	selectedCount := 0
	selectedResources := make([]string, 0)

	for value, selected := range ui.selectedItems {
		if selected {
			// Find the corresponding item to check if it's a grouping
			isGrouping := false
			for _, item := range ui.items {
				if item.Value == value && item.IsGrouping {
					isGrouping = true
					break
				}
			}
			if !isGrouping {
				selectedCount++
			}
			selectedResources = append(selectedResources, value)
		}
	}

	// Sort resources for consistent display
	sort.Strings(selectedResources)

	// Show count
	if _, err := fmt.Fprintf(v, "Selected: %d resource(s)\n\n", selectedCount); err != nil {
		return fmt.Errorf("failed to write count: %w", err)
	}

	// Show selected resources with proper indentation
	for _, resource := range selectedResources {
		// Find the corresponding item to get its level
		level := 0
		for _, item := range ui.items {
			if item.Value == resource {
				level = item.Level
				break
			}
		}

		indent := strings.Repeat("    ", level)
		if _, err := fmt.Fprintf(v, "%s• %s\n", indent, resource); err != nil {
			return fmt.Errorf("failed to write resource: %w", err)
		}
	}

	return nil
}

func (ui *UI) quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (ui *UI) cursorDown(g *gocui.Gui, v *gocui.View) error {
	if ui.currentIndex < len(ui.items)-1 {
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

func (ui *UI) toggleSelection(g *gocui.Gui, v *gocui.View) error {
	item := &ui.items[ui.currentIndex]
	item.IsSelected = !item.IsSelected
	ui.selectedItems[item.Value] = item.IsSelected

	// If it's a module, select/deselect all its resources
	if item.IsModule {
		// Use the exact resource paths with indices
		for _, resource := range item.Resources {
			fullPath := item.Value + "." + resource
			ui.selectedItems[fullPath] = item.IsSelected
		}

		// Update visible resource selections if expanded
		if item.IsExpanded {
			for i := ui.currentIndex + 1; i < len(ui.items); i++ {
				if ui.items[i].Level <= item.Level {
					break
				}
				ui.items[i].IsSelected = item.IsSelected
			}
		}
	}
	return ui.updateViews(g)
}

func (ui *UI) toggleExpand(g *gocui.Gui, v *gocui.View) error {
	item := &ui.items[ui.currentIndex]
	if !item.IsModule {
		return nil
	}

	if item.IsExpanded {
		ui.collapseModule(item)
	} else {
		ui.expandModule(item)
	}
	return ui.updateViews(g)
}

func (ui *UI) expandModule(item *SelectionItem) {
	if !item.IsModule || item.IsExpanded {
		return
	}

	// Create new items for module resources
	newItems := make([]SelectionItem, 0, len(item.Resources))

	for _, resource := range item.Resources {
		// The resource string already includes indices from resources.go
		fullPath := "module." + item.ModuleName + "." + resource

		newItems = append(newItems, SelectionItem{
			Display:    resource, // Show just the resource part (which includes indices)
			Value:      fullPath, // Keep full path for selection/state operations
			IsModule:   false,
			ModuleName: item.ModuleName,
			Level:      item.Level + 1,
			IsSelected: ui.selectedItems[fullPath],
		})
	}

	// Find position to insert
	pos := -1
	for i, it := range ui.items {
		if it.Value == item.Value {
			pos = i
			break
		}
	}

	if pos >= 0 {
		ui.items = append(ui.items[:pos+1], append(newItems, ui.items[pos+1:]...)...)
		item.IsExpanded = true
	}
}

func (ui *UI) collapseModule(item *SelectionItem) {
	if !item.IsModule || !item.IsExpanded {
		return
	}

	// Find the range of items to remove
	start := -1
	end := -1
	for i, it := range ui.items {
		if it.Value == item.Value {
			start = i + 1
		} else if start >= 0 && it.Level <= item.Level {
			end = i
			break
		}
	}
	if end == -1 {
		end = len(ui.items)
	}

	if start >= 0 && end > start {
		// Remove the expanded items
		ui.items = append(ui.items[:start], ui.items[end:]...)
		item.IsExpanded = false
	}
}

func (ui *UI) goToTop(g *gocui.Gui, v *gocui.View) error {
	ui.currentIndex = 0
	ui.origin = 0
	return ui.updateViews(g)
}

func (ui *UI) goToBottom(g *gocui.Gui, v *gocui.View) error {
	ui.currentIndex = len(ui.items) - 1
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
	if ui.currentIndex >= len(ui.items) {
		ui.currentIndex = len(ui.items) - 1
	}
	return ui.updateViews(g)
}

func (ui *UI) collapseCurrentModule(g *gocui.Gui, v *gocui.View) error {
	item := &ui.items[ui.currentIndex]
	if item.IsModule && item.IsExpanded {
		ui.collapseModule(item)
		return ui.updateViews(g)
	}

	// If not on a module, try to collapse parent module
	for i := ui.currentIndex - 1; i >= 0; i-- {
		if ui.items[i].IsModule && ui.items[i].IsExpanded && ui.items[i].Level < item.Level {
			ui.collapseModule(&ui.items[i])
			return ui.updateViews(g)
		}
	}

	return nil
}
