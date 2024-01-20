package popup

import (
	"github.com/darkhz/invidtui/client"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/gdamore/tcell/v2"
)

// ShowLink shows a popup with Invidious and Youtube
// links for the currently selected video/playlist/channel entry.
func ShowLink() {
	var linkModal *app.Modal

	property := theme.ThemeProperty{
		Item:    theme.ThemePopupBackground,
		Context: theme.ThemeContextLinks,
	}

	info, err := app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}

	builder := theme.NewTextBuilder(property.Context)
	invlink, ytlink := getLinks(info)

	builder.Format(theme.ThemeText, "header", "Invidious link\n")
	builder.Format(theme.ThemeInvidiousURI, "invidious", "%s\n\n", invlink)

	builder.Format(theme.ThemeText, "header", "Youtube link\n")
	builder.Format(theme.ThemeYoutubeURI, "youtube", "%s", ytlink)

	linkView := theme.NewTextView(property)
	linkView.SetText(builder.Get())
	linkView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeySelect, keybinding.KeyClose:
			linkModal.Exit(false)
		}

		return event
	})
	linkView.SetFocusFunc(func() {
		app.SetContextMenu("", nil)
	})

	linkModal = app.NewModal("link", "Copy link", linkView, 10, len(invlink)+10, property)
	linkModal.Show(false)
}

// getLinks returns the Invidious and Youtube links
// according to the currently selected entry's type (video/playlist/channel).
func getLinks(info inv.SearchData) (string, string) {
	var linkparam string

	invlink := client.Instance()
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
