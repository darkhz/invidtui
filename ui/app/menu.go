package app

import (
	"fmt"
	"strings"

	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// MenuData stores the menu items and handlers.
type MenuData struct {
	Visible map[keybinding.Key]func(menuType string) bool
	Items   map[keybinding.KeyContext][]keybinding.Key
}

// MenuArea stores the menu modal and the current context menu.
type MenuArea struct {
	context keybinding.KeyContext

	modal *Modal
	data  *MenuData
	focus tview.Primitive
}

var menuArea MenuArea

// InitMenu initializes the menu.
func InitMenu(data *MenuData) {
	menuArea.data = data

	AddMenu("App")
	AddMenu("Player")
}

// AddMenu adds a menu to the menubar.
func AddMenu(menuType keybinding.KeyContext) {
	_, ok := menuArea.data.Items[menuType]
	if !ok {
		return
	}

	text := UI.Menu.GetText(false)
	if text == "" {
		text = string('\u2261')
	}

	UI.Menu.SetText(menuFormat(text, string(menuType), string(menuType)))
}

// MenuExit closes the menu.
func MenuExit() {
	UI.Menu.Highlight("")
	menuArea.modal.Exit(false)
}

// SetContextMenu sets the context menu.
func SetContextMenu(menuType keybinding.KeyContext, item tview.Primitive) {
	if menuArea.context == menuType && menuArea.focus == item {
		return
	}

	menuArea.context = menuType

	text := UI.Menu.GetText(false)
	if text == "" {
		text = string('\u2261')
	}

	regions := strings.Split(text, " ")
	for i, region := range regions {
		if strings.Contains(region, "context-") {
			regions = regions[0:i]
			text = strings.Join(regions, " ")
		}
	}

	if _, ok := menuArea.data.Items[keybinding.KeyContext(menuType)]; ok {
		text = menuFormat(text, "context-"+string(menuType), string(menuType))
	}

	menuArea.focus = item
	UI.Menu.SetText(text)
}

// FocusMenu activates the menu bar.
func FocusMenu() {
	if len(UI.Menu.GetHighlights()) > 0 {
		return
	}

	regions := UI.Menu.GetRegionIDs()
	if regions == nil {
		return
	}

	region := regions[0]
	for _, r := range regions {
		if strings.Contains(r, "context-") && !strings.Contains(r, "Start") {
			region = r
			break
		}
	}

	UI.Menu.Highlight(region)
}

// DrawMenu renders the menu.
//
//gocyclo:ignore
func DrawMenu(x int, region string) {
	var skipped, width int

	region, _, ok := theme.GetThemeRegion(region)
	if !ok {
		return
	}
	if strings.Contains(region, "context-") {
		region = strings.Split(region, "-")[1]
	}

	menuItems, ok := menuArea.data.Items[keybinding.KeyContext(region)]
	if !ok {
		return
	}

	modal := NewMenuModal("menu", x, 1, theme.ThemeProperty{
		Context: theme.ThemeContextMenu,
		Item:    theme.ThemePopupBackground,
	})
	modal.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event) {
		case keybinding.KeySelect:
			row, _ := modal.Table.GetSelection()
			ref := modal.Table.GetCell(row, 0).GetReference()

			if op, ok := ref.(*keybinding.KeyData); ok {
				MenuKeybindings(event)

				if op.Kb.Key != tcell.KeyRune {
					op.Kb.Rune = rune(op.Kb.Key)
				}

				ev := tcell.NewEventKey(op.Kb.Key, op.Kb.Rune, op.Kb.Mod)

				UI.Application.GetInputCapture()(ev)
				if op.Global {
					break
				}
				if menuArea.focus != nil {
					menuArea.focus.InputHandler()(ev, nil)
				}
			}

		case keybinding.KeyClose, keybinding.KeySwitch:
			MenuKeybindings(event)
		}

		return event
	})

	for row, item := range menuItems {
		if visible, ok := menuArea.data.Visible[item]; ok && !visible(region) {
			skipped++
			continue
		}

		op := keybinding.OperationData(item)
		keyname := keybinding.KeyName(op.Kb)

		opwidth := len(op.Title) + len(keyname) + 10
		if opwidth > width {
			width = opwidth
		}

		modal.Table.SetCell(row-skipped, 0, theme.NewTableCell(
			theme.ThemeContextMenu,
			theme.ThemeDescription,
			op.Title,
		).
			SetExpansion(1).
			SetReference(op),
		)

		modal.Table.SetCell(row-skipped, 1, theme.NewTableCell(
			theme.ThemeContextMenu,
			theme.ThemeKeybinding,
			keyname,
		).
			SetExpansion(1).
			SetAlign(tview.AlignRight),
		)
	}

	modal.Table.ScrollToBeginning()

	modal.Width = width
	modal.Height = (len(menuItems) - skipped) + 2
	if modal.Height > 10 {
		modal.Height = 10
	}

	menuArea.modal = modal
	modal.Show(false)
}

// MenuHighlightHandler draws the menu based on which menu name is highlighted.
func MenuHighlightHandler(added, removed, remaining []string) {
	if added == nil {
		return
	}

	for _, region := range UI.Menu.GetRegionIDs() {
		if region == added[0] {
			DrawMenu(UI.Menu.GetRegionStart(region), added[0])
			break
		}
	}
}

// MenuKeybindings describes the menu keybindings.
func MenuKeybindings(event *tcell.EventKey) *tcell.EventKey {
	switch keybinding.KeyOperation(event, keybinding.KeyContextApp) {
	case keybinding.KeySelect, keybinding.KeyClose:
		MenuExit()

	case keybinding.KeySwitch:
		var index int

		highlighted := UI.Menu.GetHighlights()
		if highlighted == nil {
			goto Event
		}

		regions := UI.Menu.GetRegionIDs()
		for i, region := range regions {
			if highlighted[0] == region {
				index = i
				break
			}
		}

		if index == len(regions)-1 {
			index = 0
		} else {
			index++
		}

		MenuExit()
		UI.Menu.Highlight(regions[index])
	}

Event:
	return event
}

// menuFormat returns the format for displaying menu names.
func menuFormat(text, region, title string) string {
	menuName := theme.SetTextStyle(
		region, title,
		theme.ThemeContextMenu, theme.ThemeName,
	)

	return fmt.Sprintf("%s %s", text, menuName)
}
