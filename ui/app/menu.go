package app

import (
	"fmt"
	"strings"

	"github.com/darkhz/invidtui/cmd"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Menu describes a menu and its options.
type Menu struct {
	Title   string
	Options []*MenuOption
}

// MenuOption describes a menu option.
type MenuOption struct {
	Title, MenuID string
	Visible       func() bool
}

// MenuArea stores the menu modal and the current context menu.
type MenuArea struct {
	context string

	modal *Modal
	focus tview.Primitive
	items map[string]*Menu
}

var menuArea MenuArea

// InitMenu initializes the menu.
func InitMenu(menuItems map[string]*Menu) {
	menuArea.items = menuItems

	AddMenu("App")
	AddMenu("Player")
}

// AddMenu adds a menu to the menubar.
func AddMenu(menuType string) {
	menu, ok := menuArea.items[menuType]
	if !ok {
		return
	}

	text := UI.Menu.GetText(false)
	if text == "" {
		text = string('\u2261')
	}

	UI.Menu.SetText(menuFormat(text, menuType, menu.Title))
}

// MenuExit closes the menu.
func MenuExit() {
	UI.Menu.Highlight("")
	menuArea.modal.Exit(false)
}

// SetContextMenu sets the context menu.
func SetContextMenu(menuType string, item tview.Primitive) {
	if menuArea.context == menuType && menuArea.focus == item {
		return
	}

	text := string('\u2261')
	menuArea.context = menuType

	regionIDs := UI.Menu.GetRegionIDs()
	for _, region := range regionIDs {
		if strings.Contains(region, "context") {
			break
		}

		text = menuFormat(text, region, menuArea.items[region].Title)
	}

	if option, ok := menuArea.items[menuType]; ok {
		text = menuFormat(text, "context-"+menuType, option.Title)
	}

	menuArea.focus = item
	UI.Menu.SetText(text)
}

// FocusMenu activates the menu bar.
func FocusMenu() {
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

	if strings.Contains(region, "context-") {
		region = strings.Split(region, "-")[1]
	}

	menu, ok := menuArea.items[region]
	if !ok {
		return
	}

	modal := NewMenuModal("menu", x, 1)
	modal.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := modal.Table.GetSelection()
			ref := modal.Table.GetCell(row, 0).GetReference()

			if option, ok := ref.([]string); ok {
				MenuKeybindings(event)

				op := cmd.OperationKey(option[0], option[1])
				if op.Key != tcell.KeyRune {
					op.Rune = rune(op.Key)
				}

				ev := tcell.NewEventKey(op.Key, op.Rune, op.Mod)

				UI.Application.GetInputCapture()(ev)
				if option[0] == "App" || option[0] == "Player" {
					break
				}
				if menuArea.focus != nil {
					menuArea.focus.InputHandler()(ev, nil)
				}
			}

		case tcell.KeyEscape, tcell.KeyTab:
			MenuKeybindings(event)
		}

		return event
	})

	for row, option := range menu.Options {
		if option.Visible != nil && !option.Visible() {
			skipped++
			continue
		}

		op := cmd.OperationKey(menu.Title, option.MenuID)
		ev := tcell.NewEventKey(op.Key, op.Rune, op.Mod)

		keyname := ev.Name()
		if op.Key == tcell.KeyRune {
			if op.Rune == ' ' {
				keyname = "Space"
			} else {
				keyname = string(op.Rune)
			}
		}
		if op.Mod == tcell.ModAlt {
			keyname = "Alt+" + keyname
		}

		opwidth := len(option.Title) + len(keyname) + 10
		if opwidth > width {
			width = opwidth
		}

		modal.Table.SetCell(row-skipped, 0, tview.NewTableCell(option.Title).
			SetExpansion(1).
			SetReference([]string{menu.Title, option.MenuID}).
			SetAttributes(tcell.AttrBold),
		)

		modal.Table.SetCell(row-skipped, 1, tview.NewTableCell(keyname).
			SetExpansion(1).
			SetAlign(tview.AlignRight),
		)
	}

	modal.Width = width
	modal.Height = (len(menu.Options) - skipped) + 2
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

	for _, region := range UI.Menu.GetRegionInfos() {
		if region.ID == added[0] {
			DrawMenu(region.FromX, added[0])
			break
		}
	}
}

// MenuKeybindings describes the menu keybindings.
func MenuKeybindings(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter, tcell.KeyEscape:
		MenuExit()

	case tcell.KeyTab:
		var index int

		highlighted := UI.Menu.GetHighlights()
		if highlighted == nil {
			goto Event
		}

		regions := UI.Menu.GetRegionInfos()
		for i, region := range regions {
			if highlighted[0] == region.ID {
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
		UI.Menu.Highlight(regions[index].ID)
	}

Event:
	return event
}

// menuFormat returns the format for displaying menu names.
func menuFormat(text, region, title string) string {
	return fmt.Sprintf("%s [\"%s\"][::b]%s[\"\"][\"\"]", text, region, title)
}
