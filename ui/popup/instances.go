package popup

import (
	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ShowInstancesList shows a popup with a list of instances.
func ShowInstancesList() {
	var instancesModal *app.Modal

	app.ShowInfo("Loading instance list", true)

	property := theme.ThemeProperty{
		Context: theme.ThemeContextInstances,
		Item:    theme.ThemePopupBackground,
	}

	instances, err := client.GetInstances()
	if err != nil {
		app.ShowError(err)
		return
	}

	instancesView := theme.NewTable(property)
	instancesView.SetSelectable(true, false)
	instancesView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeySelect:
			row, _ := instancesView.GetSelection()
			if instance, ok := instancesView.GetCell(row, 0).GetReference().(string); ok {
				go checkInstance(instance, instancesView)
			}

		case keybinding.KeyClose:
			instancesModal.Exit(false)
		}

		return event
	})
	instancesView.SetFocusFunc(func() {
		app.SetContextMenu("", nil)
	})

	app.UI.QueueUpdateDraw(func() {
		var width int

		currentInstance := utils.GetHostname(client.Instance())

		for row, instance := range instances {
			selected := ""
			if instance == currentInstance {
				selected = "(Selected)"
			}

			if len(instance) > width {
				width = len(instance)
			}

			instancesView.SetCell(row, 0, theme.NewTableCell(
				theme.ThemeContextInstances,
				theme.ThemeInstanceURI,
				instance,
			).
				SetReference(instances[row]),
			)

			instancesView.SetCell(row, 1, theme.NewTableCell(
				theme.ThemeContextInstances,
				theme.ThemeTagChanged,
				selected,
			).
				SetSelectable(true),
			)
		}

		instancesModal = app.NewModal("instances", "Available instances", instancesView, len(instances)+4, width+15, property)
		instancesModal.Show(false)
	})

	app.ShowInfo("Instances loaded", false)
}

// checkInstance checks the instance.
func checkInstance(instance string, table *tview.Table) {
	if instance == utils.GetHostname(client.Instance()) {
		return
	}

	app.ShowInfo("Checking "+instance, true)

	instURL, err := client.CheckInstance(instance)
	if err != nil {
		app.ShowError(err)
		return
	}

	client.SetHost(instURL)

	app.UI.QueueUpdateDraw(func() {
		var cell *tview.TableCell

		for i := 0; i < table.GetRowCount(); i++ {
			if ref, ok := table.GetCell(i, 0).GetReference().(string); ok {
				c := table.GetCell(i, 1)
				if ref == instance {
					cell = c
				}

				c.SetText("")
			}
		}

		cell.SetText("(Changed)")
	})

	app.ShowInfo("Set client to "+instance, false)
}
