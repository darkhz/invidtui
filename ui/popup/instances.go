package popup

import (
	"strings"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ShowInstancesList shows a popup with a list of instances.
func ShowInstancesList() {
	var instancesModal *app.Modal

	app.ShowInfo("Loading instance list", true)

	instances, err := client.GetInstances()
	if err != nil {
		app.ShowError(err)
		return
	}

	instancesView := tview.NewTable()
	instancesView.SetSelectorWrap(true)
	instancesView.SetSelectable(true, false)
	instancesView.SetBackgroundColor(tcell.ColorDefault)
	instancesView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := instancesView.GetSelection()
			if instance, ok := instancesView.GetCell(row, 0).GetReference().(string); ok {
				go checkInstance(instance, instancesView)
			}

		case tcell.KeyEscape:
			instancesModal.Exit(false)
		}

		return event

	})

	app.UI.QueueUpdateDraw(func() {
		var width int

		currentInstance := utils.GetHostname(client.Instance())

		for row, instance := range instances {
			if instance == currentInstance {
				instance += " [white::b](Selected)[-:-:-]"
			}

			if len(instance) > width {
				width = len(instance)
			}

			instancesView.SetCell(row, 0, tview.NewTableCell(instance).
				SetReference(instances[row]).
				SetTextColor(tcell.ColorBlue).
				SetSelectedStyle(app.UI.SelectedStyle),
			)
		}

		instancesModal = app.NewModal("instances", "Available instances", instancesView, len(instances)+4, width+4)
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
			c := table.GetCell(i, 0)

			if ref, ok := c.GetReference().(string); ok {
				if ref == instance {
					cell = c
				}

				text := c.Text
				if strings.Contains(text, "Selected") || strings.Contains(text, "Changed") {
					c.SetText(ref)
				}
			}
		}

		cell.SetText(instance + " [white::b](Changed)[-:-:-]")
	})

	app.ShowInfo("Set client to "+instance, false)
}
