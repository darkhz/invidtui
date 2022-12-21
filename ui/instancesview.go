package ui

import (
	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ViewInstances shows a popup with a list of instances.
func ViewInstances() {
	var pg string

	App.QueueUpdateDraw(func() {
		pg, _ = VPage.GetFrontPage()
	})
	if pg == "dashboard" {
		return
	}

	InfoMessage("Loading instance list", true)

	instances, err := lib.GetInstanceList()
	if err != nil {
		ErrorMessage(err)
		return
	}

	instancesTable := tview.NewTable()
	instancesTable.SetSelectorWrap(true)
	instancesTable.SetSelectable(true, false)
	instancesTable.SetBackgroundColor(tcell.ColorDefault)

	title := tview.NewTextView()
	title.SetDynamicColors(true)
	title.SetTextColor(tcell.ColorBlue)
	title.SetTextAlign(tview.AlignCenter)
	title.SetText("[white::bu]Instances list")
	title.SetBackgroundColor(tcell.ColorDefault)

	popup := tview.NewFlex().
		AddItem(title, 1, 0, false).
		AddItem(instancesTable, 10, 10, false).
		SetDirection(tview.FlexRow)

	instancesTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := instancesTable.GetSelection()
			instance := instancesTable.GetCell(row, 0).Text

			go checkInstance(instance)

		case tcell.KeyEscape:
			plExit()
		}

		return event
	})

	App.QueueUpdateDraw(func() {
		for row, instance := range instances {
			instancesTable.SetCell(row, 0, tview.NewTableCell(instance).
				SetTextColor(tcell.ColorBlue).
				SetSelectedStyle(mainStyle),
			)
		}

		MPage.AddAndSwitchToPage(
			"instances",
			statusmodal(popup, instancesTable),
			true,
		).ShowPage("ui")

		App.SetFocus(instancesTable)
	})

	InfoMessage("Instances loaded", false)
}

// checkInstance checks the instance.
func checkInstance(instance string) {
	InfoMessage("Checking "+instance, true)

	instURL, err := lib.CheckInstance(lib.GetClient(), instance)
	if err != nil {
		ErrorMessage(err)
		return
	}

	lib.SetClient(instURL)

	InfoMessage("Set client to "+instance, false)
}
