package popup

import (
	"github.com/darkhz/invidtui/cmd"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ShowThemes shows the theme directory.
func ShowThemes() {
	dir, err := cmd.GetConfigDir("themes")
	if err != nil {
		app.ShowError(err)
		return
	}

	app.UI.QueueUpdateDraw(func() {
		app.UI.FileBrowser.Show(
			"Select theme:",
			ApplyTheme,
			app.FileBrowserOptions{
				SetDir:    dir,
				ResetPath: true,
			},
		)
	})
}

// ApplyTheme applies the theme from the provided file.
func ApplyTheme(themePath string) {
	app.ShowInfo("Applying theme", true)

	if err := theme.ParseFile(themePath); err != nil {
		showErrorModal(err)
		return
	}

	app.UI.QueueUpdateDraw(func() {
		theme.UpdateThemeVersion()
		app.UI.FileBrowser.Hide()
	})

	app.ShowInfo("Theme applied", false)
}

// showErrorModal shows a modal with an error message.
func showErrorModal(err error) {
	var modal *app.Modal

	property := theme.ThemeProperty{
		Item:    theme.ThemePopupBackground,
		Context: theme.ThemeContextApp,
	}

	errorView := theme.NewTextView(property)
	errorView.SetText(
		theme.SetTextStyle("error", err.Error(), property.Context, theme.ThemeErrorMessage),
	)
	errorView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeyClose:
			modal.Exit(false)
		}

		return event
	})
	errorView.SetFocusFunc(func() {
		app.SetContextMenu("", nil)
	})

	lines := tview.WordWrap(errorView.GetText(false), 100)
	height, width := len(lines), 10
	if height > 100 {
		height = 100
	}
	for _, line := range lines {
		if w := tview.TaggedStringWidth(line); w > width {
			width = w
		}
	}

	modal = app.NewModal("error", "Theme Error", errorView, height+4, width+4, property)

	app.UI.QueueUpdateDraw(func() {
		app.ShowInfo("Error applying theme", false)
		modal.Show(false)
	})
}
