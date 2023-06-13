package popup

import (
	"github.com/darkhz/invidtui/client"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ShowVideoLink shows a popup with Invidious and Youtube
// links for the currently selected video/playlist/channel entry.
func ShowVideoLink() {
	var linkModal *app.Modal

	info, err := app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}

	invlink, ytlink := getLinks(info)
	linkText := "[::u]Invidious link[-:-:-]\n[::b]" + invlink +
		"\n\n[::u]Youtube link[-:-:-]\n[::b]" + ytlink

	linkView := tview.NewTextView()
	linkView.SetText(linkText)
	linkView.SetDynamicColors(true)
	linkView.SetBackgroundColor(tcell.ColorDefault)
	linkView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyEscape:
			linkModal.Exit(false)
		}

		return event
	})

	linkModal = app.NewModal("link", "Copy link", linkView, 10, len(invlink)+10)
	linkModal.Show(false)
}

// getLinks returns the Invidious and Youtube links
// according to the currently selected entry's type (video/playlist/channel).
func getLinks(info inv.SearchData) (string, string) {
	var linkparam string

	invlink := "https://" + client.Instance()
	ytlink := "https://youtube.com"

	switch info.Type {
	case "video":
		linkparam = "/watch?v=" + info.VideoID

	case "playlist":
		linkparam = "/playlist?list=" + info.PlaylistID

	case "channel":
		linkparam = "/channel/" + info.AuthorID
	}

	invlink += linkparam
	ytlink += linkparam

	return invlink, ytlink
}
