package ui

import (
	"fmt"
	"strconv"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	dashFeed          *tview.Table
	dashPlaylists     *tview.Table
	dashSubscriptions *tview.Table

	dashPages    *tview.Pages
	dashPageMark *tview.TextView

	dashPrevPage string
	dashPrevItem tview.Primitive

	forceload bool
)

const (
	dashMark    = `[::bu]Dashboard[-:-:-]`
	dashAuthTab = ` ["auth"][darkcyan]Authentication[""]`
	dashTabs    = ` ["feed"][darkcyan]Feed[""] ["playlist"][darkcyan]Playlists[""] ["subscription"]Subscriptions[""]`
)

//gocyclo: ignore
// ShowDashboard shows the dashboard.
func ShowDashboard() {
	dashFeed = tview.NewTable()
	dashFeed.SetSelectorWrap(true)
	dashFeed.SetBackgroundColor(tcell.ColorDefault)
	dashFeed.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		dashTableEvents(event)
		capturePlayerEvent(event)

		switch event.Key() {
		case tcell.KeyEnter:
			go loadFeed(true, false)
		}

		switch event.Rune() {
		case '+':
			go Modify(true)

		case ';':
			showLinkPopup()

		case 'C':
			ShowComments()
		}

		return event
	})

	dashPlaylists = tview.NewTable()
	dashPlaylists.SetSelectorWrap(true)
	dashPlaylists.SetBackgroundColor(tcell.ColorDefault)
	dashPlaylists.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		dashTableEvents(event)
		capturePlayerEvent(event)

		switch event.Rune() {
		case 'i':
			ViewPlaylist(true, event.Modifiers() == tcell.ModAlt)

		case 'c':
			createPlaylistForm()

		case 'e':
			editPlaylistForm()

		case '_':
			go Modify(false)

		case ';':
			showLinkPopup()
		}

		return event
	})

	dashSubscriptions = tview.NewTable()
	dashSubscriptions.SetSelectorWrap(true)
	dashSubscriptions.SetBackgroundColor(tcell.ColorDefault)
	dashSubscriptions.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		dashTableEvents(event)
		capturePlayerEvent(event)

		switch event.Rune() {
		case 'u':
			ViewChannel("video", true, event.Modifiers() == tcell.ModAlt)

		case 'U':
			ViewChannel("playlist", true, event.Modifiers() == tcell.ModAlt)

		case '_':
			go Modify(false)

		case ';':
			showLinkPopup()
		}

		return event
	})

	dashPageMark = tview.NewTextView()
	dashPageMark.SetWrap(false)
	dashPageMark.SetRegions(true)
	dashPageMark.SetDynamicColors(true)
	dashPageMark.SetBackgroundColor(tcell.ColorDefault)
	dashPageMark.SetHighlightedFunc(func(added, removed, remaining []string) {
		if added == nil || added[0] == "" {
			return
		}

		switch added[0] {
		case "feed":
			App.SetFocus(dashFeed)
			dashPages.SwitchToPage("feed")
			go loadFeed(false, !forceload && dashFeed.GetRowCount() > 0)

		case "playlist":
			App.SetFocus(dashPlaylists)
			dashPages.SwitchToPage("playlist")
			go loadPlaylists(!forceload && dashPlaylists.GetRowCount() > 0)

		case "subscription":
			App.SetFocus(dashSubscriptions)
			dashPages.SwitchToPage("subscription")
			go loadSubscriptions(!forceload && dashSubscriptions.GetRowCount() > 0)
		}

		forceload = false
	})

	dashPages = tview.NewPages().
		AddPage("feed", dashFeed, true, false).
		AddPage("playlist", dashPlaylists, true, false).
		AddPage("subscription", dashSubscriptions, true, false)
	dashPages.SetBackgroundColor(tcell.ColorDefault)

	box := tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault)

	dashFlex := tview.NewFlex().
		AddItem(dashPageMark, 1, 0, false).
		AddItem(box, 1, 0, false).
		AddItem(dashPages, 0, 10, true).
		SetDirection(tview.FlexRow)
	dashFlex.SetBackgroundColor(tcell.ColorDefault)

	checkAuth()
	App.QueueUpdateDraw(func() {
		dashPrevPage, dashPrevItem = VPage.GetFrontPage()
		VPage.AddAndSwitchToPage("dashboard", dashFlex, true)
	})
}

// ShowAuthPage shows the authentication page.
func ShowAuthPage() {
	InfoMessage("Authentication required", false)

	App.QueueUpdateDraw(func() {
		dashPageMark.SetText(dashMark + dashAuthTab)
		dashPageMark.Highlight("auth")

		if dashPages.HasPage("auth") {
			dashPages.SwitchToPage("auth")
			return
		}

		authText := "No authorization token found or token is invalid.\n\n" +
			"To authenticate, do either of the listed steps:\n\n" +
			"- Navigate to [::b]https://" + lib.GetClient().SelectedInstance() + "/token_manager[-:-:-] " +
			"and copy the [::u]SID[-:-:-] (the base64 string on top of a red background)\n\n" +
			"- Navigate to [::b]" + lib.GetAuthLink() + "[-:-:-] and click 'OK' when prompted for confirmation, " +
			"then copy the [::u]session token[-:-:-]" +
			"\n\nPaste the SID or Token in the inputbox below and press Enter."

		dashAuth := tview.NewTextView()
		dashAuth.SetWrap(true)
		dashAuth.SetDynamicColors(true)
		dashAuth.SetText(authText)
		dashAuth.SetBackgroundColor(tcell.ColorDefault)

		dashToken := tview.NewInputField()
		dashToken.SetLabel("[white::b]Token: ")
		dashToken.SetBackgroundColor(tcell.ColorDefault)
		dashToken.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape:
				dashTableEvents(event)

			case tcell.KeyEnter:
				App.SetFocus(dashAuth)
				go checkToken(dashToken)
			}

			return event
		})

		dashAuthFlex := tview.NewFlex().
			AddItem(dashAuth, 10, 0, false).
			AddItem(nil, 1, 0, false).
			AddItem(dashToken, 6, 0, true).
			SetDirection(tview.FlexRow)

		dashPages.AddAndSwitchToPage("auth", dashAuthFlex, true)

		App.SetFocus(dashToken)
	})
}

// loadFeed loads the user's feed.
func loadFeed(getmore, loadskip bool) {
	var skipped int

	if loadskip {
		return
	}

	InfoMessage("Loading feed", true)

	feed, err := lib.GetClient().Feed(getmore)
	if err != nil {
		ErrorMessage(err)
		return
	}

	App.QueueUpdateDraw(func() {
		if !getmore {
			dashFeed.Clear()
			dashFeed.SetSelectable(false, false)
		}

		pos := -1
		_, _, width, _ := VPage.GetRect()
		rows := dashFeed.GetRowCount()

		for i, video := range feed.Videos {
			if video.LengthSeconds == 0 {
				skipped++
			}

			if pos < 0 {
				pos = (rows + i) - skipped
			}

			sref := lib.SearchResult{
				Type:     "video",
				Title:    video.Title,
				VideoID:  video.VideoID,
				AuthorID: video.AuthorID,
				Author:   video.Author,
			}

			dashFeed.SetCell((rows+i)-skipped, 0, tview.NewTableCell("[blue::b]"+tview.Escape(video.Title)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)).
				SetSelectedStyle(mainStyle),
			)

			dashFeed.SetCell((rows+i)-skipped, 1, tview.NewTableCell("[pink]"+lib.FormatDuration(video.LengthSeconds)).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}

		dashFeed.SetSelectable(true, false)

		if pos > 0 {
			dashFeed.Select(pos, 0)
		}
	})

	InfoMessage("Feed loaded", false)
}

// loadPlaylists loads the user's playlist.
func loadPlaylists(loadskip bool) {
	if loadskip {
		return
	}

	InfoMessage("Loading playlists", true)

	playlists, err := lib.GetClient().AuthPlaylists()
	if err != nil {
		ErrorMessage(err)
		return
	}

	App.QueueUpdateDraw(func() {
		_, _, width, _ := VPage.GetRect()

		dashPlaylists.SetSelectable(false, false)

		for i, playlist := range playlists {
			sref := lib.SearchResult{
				Type:       "playlist",
				Title:      playlist.Title,
				PlaylistID: playlist.PlaylistID,
				AuthorID:   playlist.AuthorID,
				Author:     playlist.Author,
			}

			dashPlaylists.SetCell(i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(playlist.Title)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)).
				SetSelectedStyle(mainStyle),
			)

			dashPlaylists.SetCell(i, 1, tview.NewTableCell("[pink]"+strconv.Itoa(playlist.VideoCount)+" videos").
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}

		dashPlaylists.SetSelectable(true, false)
	})

	InfoMessage("Playlists loaded", false)
}

// loadSubscriptions loads the user's subscriptions.
func loadSubscriptions(loadskip bool) {
	if loadskip {
		return
	}

	InfoMessage("Loading subscriptions", true)

	subscriptions, err := lib.GetClient().Subscriptions()
	if err != nil {
		ErrorMessage(err)
		return
	}

	App.QueueUpdateDraw(func() {
		_, _, width, _ := VPage.GetRect()

		dashSubscriptions.SetSelectable(false, false)

		for i, subscription := range subscriptions {
			sref := lib.SearchResult{
				Type:     "channel",
				Author:   subscription.Author,
				AuthorID: subscription.AuthorID,
			}

			dashSubscriptions.SetCell(i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(subscription.Author)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)).
				SetSelectedStyle(mainStyle),
			)
		}

		dashSubscriptions.SetSelectable(true, false)
	})

	InfoMessage("Subscriptions loaded", false)
}

// checkAuth checks whether the instance is authenticated.
// If not, it shows the authentication page.
func checkAuth() {
	InfoMessage("Loading dashboard", true)

	if lib.IsAuthInstance() && lib.AuthTokenValid() {
		setDashboard()
		return
	}

	ShowAuthPage()
}

// checkToken checks whether a session token is valid.
func checkToken(input *tview.InputField) {
	token := input.GetText()

	InfoMessage("Checking token", true)

	if !lib.TokenValid(token) {
		ErrorMessage(fmt.Errorf("Token is invalid"))
		App.QueueUpdateDraw(func() {
			App.SetFocus(input)
		})

		return
	}

	lib.AddCurrentAuth(token)
	setDashboard()
}

// setDashboard sets the dashboard tabs.
func setDashboard() {
	App.QueueUpdateDraw(func() {
		dashPageMark.SetText(dashMark + dashTabs)
		dashPageMark.Highlight("feed")
	})
}

// inAuthPage checks whether the currently showing
// page is the authentication page.
func inAuthPage() bool {
	var dashpg string

	pg, _ := VPage.GetFrontPage()
	if dashPages != nil {
		dashpg, _ = dashPages.GetFrontPage()
	}

	return pg == "dashboard" && dashpg == "auth"
}

// dashTableEvents handles the input events for the
// feed, playlist and subscription tables.
func dashTableEvents(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyTab:
		switchDashTabs()

	case tcell.KeyEscape:
		App.SetFocus(dashPrevItem)
		VPage.SwitchToPage(dashPrevPage)

	case tcell.KeyCtrlD:
		forceload = true
		pgmark := dashPageMark.GetHighlights()

		dashPageMark.Highlight("")
		dashPageMark.Highlight(pgmark[0])
	}
}

// switchDashTabs switches the dashboard tabs.
func switchDashTabs() {
	switch dashPageMark.GetHighlights()[0] {
	case "feed":
		dashPageMark.Highlight("playlist")

	case "playlist":
		dashPageMark.Highlight("subscription")

	case "subscription":
		dashPageMark.Highlight("feed")
	}
}

// exitFormPage exits the form page.
func exitFormPage(form string) {
	dashPages.RemovePage(form)

	dashPages.SwitchToPage("playlist")
	App.SetFocus(dashPlaylists)
}
