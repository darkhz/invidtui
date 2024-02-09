package view

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/darkhz/invidtui/client"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/popup"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// DashboardView describes the layout for a dashboard view.
type DashboardView struct {
	init, auth  bool
	currentType string

	modifyMap map[string]*semaphore.Weighted

	message  *tview.TextView
	token    *tview.InputField
	views    *tview.Pages
	flex     *tview.Flex
	tableMap map[string]*DashboardTable

	property theme.ThemeProperty

	lock  *semaphore.Weighted
	mutex sync.Mutex
}

// DashboardTable describes the properties of a dashboard table.
type DashboardTable struct {
	loaded bool

	page  int
	table *tview.Table
}

// Dashboard stores the dashboard view properties.
var Dashboard DashboardView

// Name returns the name of the dashboard view.
func (d *DashboardView) Name() string {
	return "Dashboard"
}

// Init initializes the dashboard view.
func (d *DashboardView) Init() bool {
	if d.init {
		return true
	}

	d.property = d.ThemeProperty()

	d.currentType = "feed"

	d.message = theme.NewTextView(d.property)
	d.message.SetWrap(true)

	d.token = theme.NewInputField(d.property, "Token:")
	d.token.SetFocusFunc(func() {
		app.SetContextMenu("", nil)
	})

	box := theme.NewBox(d.property)
	d.flex = theme.NewFlex(d.property).
		SetDirection(tview.FlexRow).
		AddItem(d.message, 10, 0, false).
		AddItem(box, 1, 0, false).
		AddItem(d.token, 6, 0, true).
		AddItem(box, 0, 1, false)

	d.views = theme.NewPages(d.property)
	d.views.AddPage("Authentication", d.flex, true, false)

	d.queueWrite(func() {
		d.tableMap = make(map[string]*DashboardTable)

		kbMap := map[string]func(e *tcell.EventKey) *tcell.EventKey{
			"Feed":          d.feedKeybindings,
			"Playlists":     d.plKeybindings,
			"Subscriptions": d.subKeybindings,
		}

		for _, info := range d.Tabs().Info {
			table := theme.NewTable(d.property)
			table.SetTitle(info.Title)
			table.SetInputCapture(kbMap[info.Title])
			table.SetFocusFunc(func() {
				app.SetContextMenu(keybinding.KeyContextDashboard, table)
			})

			d.tableMap[info.Title] = &DashboardTable{
				table: table,
			}

			d.views.AddPage(info.Title, table, true, false)
		}
	})

	d.modifyMap = make(map[string]*semaphore.Weighted)
	for _, mt := range []string{
		"video",
		"playlist",
		"channel",
	} {
		d.modifyMap[mt] = semaphore.NewWeighted(1)
	}

	d.lock = semaphore.NewWeighted(1)

	d.init = true

	return true
}

// Exit closes the dashboard view.
func (d *DashboardView) Exit() bool {
	return true
}

// Tabs describes the tab layout for the dashboard view.
func (d *DashboardView) Tabs() app.Tab {
	tab := app.Tab{Title: "Dashboard"}

	if d.auth {
		tab.Selected = "auth"
		tab.Info = []app.TabInfo{
			{ID: "auth", Title: "Authentication"},
		}
	} else {
		tab.Selected = d.currentType
		tab.Info = []app.TabInfo{
			{ID: "feed", Title: "Feed"},
			{ID: "playlists", Title: "Playlists"},
			{ID: "subscriptions", Title: "Subscriptions"},
		}
	}

	return tab
}

// Primitive returns the primitive for the dashboard view.
func (d *DashboardView) Primitive() tview.Primitive {
	return d.views
}

// ThemeProperty returns the dashboard view's theme property.
func (d *DashboardView) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextDashboard,
		Item:    theme.ThemeBackground,
	}
}

// CurrentPage returns the dashboard's current page.
func (d *DashboardView) CurrentPage(page ...string) string {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if page != nil {
		d.currentType = page[0]
	}

	return d.currentType
}

// IsFocused returns if the dashboard view is focused or not.
func (d *DashboardView) IsFocused() bool {
	return d.views != nil && d.views.HasFocus()
}

// View shows the dashboard view.
func (d *DashboardView) View(auth ...struct{}) {
	if d.views == nil {
		return
	}

	d.auth = auth != nil

	SetView(&Dashboard)
	if auth != nil {
		app.SelectTab("auth")
		d.views.SwitchToPage("Authentication")

		return
	}

	app.SelectTab(d.CurrentPage())

	for _, i := range d.Tabs().Info {
		if i.ID == d.CurrentPage() {
			d.views.SwitchToPage(i.Title)
			app.UI.SetFocus(d.getTableMap()[i.Title].table)

			break
		}
	}
}

// Load loads the dashboard view according to the provided page type.
func (d *DashboardView) Load(pageType string, reload ...struct{}) {
	switch pageType {
	case "feed":
		go d.loadFeed(reload != nil)

	case "playlists":
		go d.loadPlaylists(reload != nil)

	case "subscriptions":
		go d.loadSubscriptions(reload != nil)
	}

	d.CurrentPage(pageType)
	d.View()
}

// EventHandler checks whether authentication is needed
// before showing the dashboard view.
func (d *DashboardView) EventHandler() {
	d.Init()

	if pg, _ := d.views.GetFrontPage(); d.views.HasFocus() && pg != "Authentication" {
		d.Load(d.CurrentPage(), struct{}{})
		return
	}

	go d.checkAuth()
}

// AuthPage shows the authentication page.
func (d *DashboardView) AuthPage() {
	app.ShowInfo("Authentication required", false)

	builder := theme.NewTextBuilder(theme.ThemeContextDashboard)
	builder.Start(theme.ThemeText, "auth")
	builder.AppendText("No authorization token found or token is invalid.\n\n")
	builder.AppendText("To authenticate, do either of the listed steps:\n\n")

	builder.AppendText("- Navigate to ")
	builder.Format(theme.ThemeInstanceURI, "token_manager", "%s/token_manager", client.Instance())
	builder.AppendText(" and copy the SID (the base64 string on top of a red background)\n\n")

	builder.AppendText("- Navigate to ")
	builder.Format(theme.ThemeInstanceURI, "auth_link", "%s", client.AuthLink())
	builder.AppendText(" and click 'OK' when prompted for confirmation, then copy the session token\n\n")

	builder.AppendText("Paste the SID or Token in the inputbox below and press Enter.")
	builder.Finish()

	d.message.SetText(builder.Get())
	d.token.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeySelect:
			app.UI.SetFocus(d.message)
			go d.validateToken()

		case keybinding.KeyClose:
			d.Keybindings(event)
		}

		return event
	})

	d.View(struct{}{})
}

// ModifyHandler handles the following activities:
// - Adding/removing videos to/from a user playlist
// - Deleting user playlists
// - Subscribing/unsubscribing to/from channels
func (d *DashboardView) ModifyHandler(add bool) {
	d.Init()

	info, err := app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}

	if !client.IsAuthInstance() {
		app.ShowInfo("Authentication is required", false)
		return
	}

	go func(i inv.SearchData, lock *semaphore.Weighted, focused bool) {
		if !lock.TryAcquire(1) {
			app.ShowInfo("Operation in progress for "+info.Type, false)
			return
		}
		defer lock.Release(1)

		switch info.Type {
		case "video":
			d.modifyVideoInPlaylist(i, add, lock)

		case "playlist":
			d.modifyPlaylist(i, add, focused)

		case "channel":
			d.modifySubscription(i, add, focused)
		}
	}(info, d.modifyMap[info.Type], d.views.HasFocus())
}

// PlaylistForm displays a form to create/edit a user playlist.
func (d *DashboardView) PlaylistForm(edit bool) {
	var modal *app.Modal
	var info inv.SearchData

	mode := "Create"

	if edit {
		mode = "Edit"
		info, _ = app.FocusedTableReference()
	}

	property := d.property.
		SetItem(theme.ThemePopupBackground)

	form := theme.NewForm(property)
	form.AddInputField("Name:", info.Title, 0, nil, nil)
	form.AddDropDown("Privacy:", []string{"public", "unlisted", "private"}, -1, nil)
	if edit {
		form.AddInputField("Description:", info.Description, 0, nil, nil)
	}
	form.AddButton(mode, func() {
		go d.playlistFormHandler(form, modal, info, mode, edit)
	})
	form.AddButton("Cancel", func() {
		modal.Exit(false)
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeyClose:
			modal.Exit(false)
		}

		return event
	})

	modal = app.NewModal("playlist_editor", mode+" playlist", form, form.GetFormItemCount()+10, 60, property)
	modal.Show(false)
}

// Keybindings defines the keybindings for the dashboard view.
func (d *DashboardView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch keybinding.KeyOperation(event, keybinding.KeyContextDashboard) {
	case keybinding.KeySwitchTab:
		d.CurrentPage(app.SwitchTab(false))

		client.Cancel()
		app.ShowInfo("", false)
		d.Load(d.CurrentPage())

	case keybinding.KeyClose:
		client.Cancel()
		CloseView()

	case keybinding.KeyDashboardReload:
		d.Load(d.CurrentPage(), struct{}{})
	}

	return event
}

// feedKeybindings defines keybindings for the feed page.
func (d *DashboardView) feedKeybindings(event *tcell.EventKey) *tcell.EventKey {
	d.Keybindings(event)

	switch keybinding.KeyOperation(event, keybinding.KeyContextComments) {
	case keybinding.KeyLoadMore:
		d.loadFeed(false, struct{}{})

	case keybinding.KeyAdd:
		d.ModifyHandler(true)

	case keybinding.KeyLink:
		popup.ShowLink()

	case keybinding.KeyComments:
		Comments.Show()
	}

	return event
}

// plKeybindings defines keybindings for the playlist page.
func (d *DashboardView) plKeybindings(event *tcell.EventKey) *tcell.EventKey {
	d.Keybindings(event)

	switch keybinding.KeyOperation(event, keybinding.KeyContextDashboard) {
	case keybinding.KeyPlaylist:
		Playlist.EventHandler(event.Modifiers() == tcell.ModAlt, true)

	case keybinding.KeyDashboardCreatePlaylist, keybinding.KeyDashboardEditPlaylist:
		d.PlaylistForm(event.Rune() == 'e')

	case keybinding.KeyRemove:
		d.ModifyHandler(false)

	case keybinding.KeyLink:
		popup.ShowLink()
	}

	return event
}

// subKeybindings defines keybindings for the subscription page.
func (d *DashboardView) subKeybindings(event *tcell.EventKey) *tcell.EventKey {
	d.Keybindings(event)

	switch keybinding.KeyOperation(event, keybinding.KeyContextComments) {
	case keybinding.KeyChannelVideos:
		Channel.EventHandler("video", event.Modifiers() == tcell.ModAlt)

	case keybinding.KeyChannelPlaylists:
		Channel.EventHandler("playlist", event.Modifiers() == tcell.ModAlt)

	case keybinding.KeyChannelReleases:
		Channel.EventHandler("releases", event.Modifiers() == tcell.ModAlt)

	case keybinding.KeyRemove:
		d.ModifyHandler(false)

	case keybinding.KeyComments:
		Comments.Show()
	}

	return event
}

// checkAuth checks if the user is authenticated
// before loading the dashboard.
func (d *DashboardView) checkAuth() {
	if !d.lock.TryAcquire(1) {
		return
	}
	defer d.lock.Release(1)

	app.ShowInfo("Loading dashboard", true)

	auth := client.IsAuthInstance() && client.CurrentTokenValid()

	app.UI.QueueUpdateDraw(func() {
		if auth {
			d.Load(d.CurrentPage(), struct{}{})
			app.ShowInfo("Dashboard loaded", false)

			return
		}

		d.AuthPage()
	})
}

// playlistFormHandler handles creating/editing the user playlist.
func (d *DashboardView) playlistFormHandler(
	form *tview.Form, modal *app.Modal, info inv.SearchData,
	mode string, edit bool,
) {
	var description string

	title := form.GetFormItem(0).(*tview.InputField).GetText()
	_, privacy := form.GetFormItem(1).(*tview.DropDown).GetCurrentOption()

	if title == "" || privacy == "" {
		app.ShowError(fmt.Errorf("View: Dashboard: Cannot submit empty form data"))
		return
	}

	app.UI.QueueUpdateDraw(func() {
		modal.Exit(false)
	})

	if !edit {
		mode = mode[:len(mode)-1]
	}

	app.ShowInfo(mode+"ing playlist "+info.Title, true)

	if edit {
		description = form.GetFormItem(2).(*tview.InputField).GetText()

		err := inv.EditPlaylist(info.PlaylistID, title, description, privacy)
		if err != nil {
			app.ShowError(err)
			return
		}

		newInfo := info
		newInfo.Title = title
		newInfo.Description = description
		title = theme.SetTextStyle(
			"title",
			tview.Escape(title),
			theme.ThemeContextDashboard,
			theme.ThemeVideo,
		)

		app.UI.QueueUpdateDraw(func() {
			if err := app.ModifyReference(title, true, info, newInfo); err != nil {
				app.ShowError(err)
			}
		})
	} else {
		if err := inv.CreatePlaylist(title, privacy); err != nil {
			app.ShowError(err)
			return
		}

		d.loadPlaylists(true)
	}

	app.ShowInfo(mode+"ed playlist "+info.Title, false)
}

// modifySubscription adds/removes a channel subscription.
func (d *DashboardView) modifySubscription(info inv.SearchData, add, focused bool) {

	if add && !focused {
		info.Author = tview.Escape(info.Author)
		app.ShowInfo("Subscribing to "+info.Author, true)

		if err := inv.AddSubscription(info.AuthorID); err != nil {
			app.ShowError(err)
			return
		}

		app.ShowInfo("Subscribed to "+info.Author, false)

		return
	}

	if !add && !focused {
		return
	}

	info.Author = tview.Escape(info.Author)
	app.ShowInfo("Unsubscribing from "+info.Author, true)

	if err := inv.RemoveSubscription(info.AuthorID); err != nil {
		app.ShowError(err)
		return
	}

	app.UI.QueueUpdateDraw(func() {
		if err := app.ModifyReference("", false, info); err != nil {
			app.ShowError(err)
		}
	})

	app.ShowInfo("Unsubscribed from "+info.Author, false)
}

// modifyPlaylist removes a user playlist.
func (d *DashboardView) modifyPlaylist(info inv.SearchData, add, focused bool) {
	if add || !focused {
		return
	}

	app.ShowInfo("Removing playlist "+info.Title, true)

	if err := inv.RemovePlaylist(info.PlaylistID); err != nil {
		app.ShowError(err)
		return
	}

	app.UI.QueueUpdateDraw(func() {
		if err := app.ModifyReference("", false, info); err != nil {
			app.ShowError(err)
		}
	})

	app.ShowInfo("Removed playlist "+info.Title, false)
}

// modifyVideoInPlaylist adds/removes videos in a playlist.
func (d *DashboardView) modifyVideoInPlaylist(info inv.SearchData, add bool, lock *semaphore.Weighted) {
	if !add {
		d.removeVideo(info)
		return
	}

	var modal *app.Modal

	app.ShowInfo("Retrieving playlists", true)

	playlists, err := inv.UserPlaylists()
	if err != nil {
		app.ShowError(err)
		return
	}
	if len(playlists) == 0 {
		app.ShowInfo("No user playlists found", false)
		return
	}

	app.ShowInfo("Retrieved playlists", false)

	property := d.property.
		SetItem(theme.ThemePopupBackground)

	table := theme.NewTable(property)
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextDashboard) {
		case keybinding.KeySelect:
			playlist, _ := app.FocusedTableReference()
			modal.Exit(false)

			go d.addVideo(info, playlist, lock)

		case keybinding.KeyClose:
			modal.Exit(false)
		}

		return event
	})

	for i, p := range playlists {
		ref := inv.SearchData{
			Type:       "playlist",
			Title:      p.Title,
			PlaylistID: p.PlaylistID,
			Author:     p.Author,
		}

		table.SetCell(i, 0, theme.NewTableCell(
			property.Context,
			theme.ThemeVideo,
			tview.Escape(p.Title),
		).
			SetExpansion(1).
			SetReference(ref),
		)

		table.SetCell(i, 1, theme.NewTableCell(
			property.Context,
			theme.ThemeTotalVideos,
			strconv.FormatInt(p.VideoCount, 10)+" videos",
		).
			SetSelectable(true).
			SetAlign(tview.AlignRight),
		)
	}

	modal = app.NewModal("user_playlists", "Add to playlist", table, 20, 60, property)

	app.UI.QueueUpdateDraw(func() {
		modal.Show(false)
	})
}

// removeVideo removes a video from a user playlist.
func (d *DashboardView) removeVideo(info inv.SearchData) {
	app.ShowInfo("Removing video from "+info.Title, true)

	if err := inv.RemoveVideoFromPlaylist(info.PlaylistID, info.IndexID); err != nil {
		app.ShowError(err)
		return
	}

	app.UI.QueueUpdateDraw(func() {
		if err := app.ModifyReference("", false, info); err != nil {
			app.ShowError(err)
		}
	})

	app.ShowInfo("Removed video from "+info.Title, false)
}

// addVideo adds a video to a user playlist.
func (d *DashboardView) addVideo(info, playlist inv.SearchData, lock *semaphore.Weighted) {
	if !lock.TryAcquire(1) {
		app.ShowError(fmt.Errorf("View: Dashboard: Cannot add video, operation in progress"))
		return
	}
	defer lock.Release(1)

	app.ShowInfo("Adding "+info.Title+" to "+playlist.Title, true)

	err := inv.AddVideoToPlaylist(playlist.PlaylistID, info.VideoID)
	if err != nil {
		app.ShowError(err)
		return
	}

	app.ShowInfo("Added "+info.Title+" to "+playlist.Title, false)
}

// loadFeed loads and renders the user feed.
func (d *DashboardView) loadFeed(reload bool, loadMore ...struct{}) {
	feedView := d.getTableMap()["Feed"]

	if loadMore != nil {
		feedView.page++
		goto LoadFeed
	} else {
		feedView.page = 1
	}

	if !reload && feedView.loaded {
		return
	}

LoadFeed:
	app.ShowInfo("Loading feed", true)

	feed, err := inv.Feed(feedView.page)
	if err != nil {
		app.ShowError(err)
		return
	}

	feedView.loaded = true

	app.UI.QueueUpdateDraw(func() {
		var skipped int

		if loadMore == nil {
			feedView.table.Clear()
		}

		_, _, width, _ := app.UI.Pages.GetRect()
		rows := feedView.table.GetRowCount()

		for i, video := range feed.Videos {
			if video.LengthSeconds == 0 {
				skipped++
				continue
			}

			sref := inv.SearchData{
				Type:     "video",
				Title:    video.Title,
				VideoID:  video.VideoID,
				AuthorID: video.AuthorID,
				Author:   video.Author,
			}

			feedView.table.SetCell((rows+i)-skipped, 0, theme.NewTableCell(
				theme.ThemeContextDashboard,
				theme.ThemeVideo,
				tview.Escape(video.Title),
			).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)),
			)

			feedView.table.SetCell((rows+i)-skipped, 1, theme.NewTableCell(
				theme.ThemeContextDashboard,
				theme.ThemeTotalDuration,
				utils.FormatDuration(video.LengthSeconds),
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}

		feedView.table.SetSelectable(true, false)
		app.SetTableSelector(feedView.table, rows)
	})

	app.ShowInfo("Feed loaded", false)
}

// loadPlaylists loads and renders the user playlists.
func (d *DashboardView) loadPlaylists(reload bool) {
	plView := d.getTableMap()["Playlists"]

	if !reload && plView.loaded {
		return
	}

	app.ShowInfo("Loading playlists", true)

	playlists, err := inv.UserPlaylists()
	if err != nil {
		app.ShowError(err)
		return
	}

	plView.loaded = true

	app.UI.QueueUpdateDraw(func() {
		_, _, width, _ := app.UI.Pages.GetRect()
		rows := plView.table.GetRowCount()

		plView.table.SetSelectable(false, false)

		for i, playlist := range playlists {
			sref := inv.SearchData{
				Type:       "playlist",
				Title:      playlist.Title,
				PlaylistID: playlist.PlaylistID,
				AuthorID:   playlist.AuthorID,
				Author:     playlist.Author,
			}

			plView.table.SetCell(i, 0, theme.NewTableCell(
				theme.ThemeContextDashboard,
				theme.ThemePlaylist,
				tview.Escape(playlist.Title),
			).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)),
			)

			plView.table.SetCell(i, 1, theme.NewTableCell(
				theme.ThemeContextDashboard,
				theme.ThemeTotalVideos,
				strconv.FormatInt(playlist.VideoCount, 10)+" videos",
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}

		plView.table.SetSelectable(true, false)
		app.SetTableSelector(plView.table, rows)
	})

	app.ShowInfo("Playlists loaded", false)
}

// loadSubscriptions loads and renders the user subscriptions.
func (d *DashboardView) loadSubscriptions(reload bool) {
	subView := d.getTableMap()["Subscriptions"]

	if !reload && subView.loaded {
		return
	}

	app.ShowInfo("Loading subscriptions", true)

	subscriptions, err := inv.Subscriptions()
	if err != nil {
		app.ShowError(err)
		return
	}

	subView.loaded = true

	app.UI.QueueUpdateDraw(func() {
		_, _, width, _ := app.UI.Pages.GetRect()
		rows := subView.table.GetRowCount()

		subView.table.SetSelectable(false, false)

		for i, subscription := range subscriptions {
			sref := inv.SearchData{
				Type:     "channel",
				Author:   subscription.Author,
				AuthorID: subscription.AuthorID,
			}

			subView.table.SetCell(i, 0, theme.NewTableCell(
				theme.ThemeContextDashboard,
				theme.ThemeChannel,
				tview.Escape(subscription.Author),
			).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)),
			)
		}

		subView.table.SetSelectable(true, false)
		app.SetTableSelector(subView.table, rows)
	})

	app.ShowInfo("Subscriptions loaded", false)
}

// validateToken validates the provided token
// in the authentication page.
func (d *DashboardView) validateToken() {
	app.ShowInfo("Checking token", true)

	if !client.IsTokenValid(d.token.GetText()) {
		app.ShowError(fmt.Errorf("View: Dashboard: Token is invalid"))
		app.UI.QueueUpdateDraw(func() {
			app.UI.SetFocus(d.token)
		})

		return
	}

	client.AddCurrentAuth(d.token.GetText())
	d.Load(d.CurrentPage())
}

// getTableMap gets a map of tables within the dashboard view.
func (d *DashboardView) getTableMap() map[string]*DashboardTable {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.tableMap
}

// queueWrite executes the given function thread-safely.
func (d *DashboardView) queueWrite(write func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	write()
}
