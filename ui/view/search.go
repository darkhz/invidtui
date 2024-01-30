package view

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/cmd"
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

// SearchView describes the layout for a search view.
type SearchView struct {
	init                              bool
	page, pos                         int
	currentType, savedText, file, tab string
	entries                           []string

	table *tview.Table

	suggestBox  *app.Modal
	suggestText string

	parametersBox  *app.Modal
	parametersForm *tview.Form
	parameters     map[string]string

	property theme.ThemeProperty

	lock *semaphore.Weighted
}

var (
	// Search stores the search view properties
	Search SearchView

	formParams = map[string]map[string][]string{
		"Date:": {"date": []string{
			"",
			"hour",
			"week",
			"year",
			"month",
			"today",
		}},
		"Sort By:": {"sort_by": []string{
			"",
			"rating",
			"relevance",
			"view_count",
			"upload_date",
		}},
		"Duration:": {"duration": []string{
			"",
			"long",
			"short",
		}},
		"Features:": {"features": []string{
			"4k",
			"hd",
			"3d",
			"360",
			"hdr",
			"live",
			"location",
			"purchased",
			"subtitles",
			"creative_commons",
		}},
		"Region:": {"region": []string{}},
	}
)

// Name returns the name of the search view.
func (s *SearchView) Name() string {
	return "Search"
}

// Init initializes the search view.
func (s *SearchView) Init() bool {
	if s.init {
		return true
	}

	s.property = s.ThemeProperty()

	s.currentType = "video"
	s.tab = s.currentType

	s.table = theme.NewTable(s.property)
	s.table.SetBorder(false)
	s.table.SetInputCapture(s.Keybindings)
	s.table.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextSearch, s.table)
	})

	s.suggestBox = app.NewModal(
		"suggestion", "Suggestions", nil, 0, 0,
		s.property.SetItem(theme.ThemePopupBackground),
	)
	s.suggestBox.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeyClose:
			s.suggestBox.Exit(true)
		}

		return event
	})
	s.suggestBox.Table.SetSelectionChangedFunc(func(row, column int) {
		text, ok := s.suggestBox.Table.GetCell(row, column).GetReference().(string)
		if ok {
			app.UI.Status.SetText(text)
		}
	})

	if s.parametersForm == nil {
		s.parametersForm = theme.NewForm(s.property.SetItem(theme.ThemePopupBackground))
	}

	s.parametersBox = app.NewModal(
		"parameters", "Set Search Parameters", s.parametersForm, 40, 60,
		s.property.SetItem(theme.ThemePopupBackground),
	)

	s.parameters = make(map[string]string)

	s.lock = semaphore.NewWeighted(1)

	s.setupHistory()

	s.init = true

	return true
}

// Exit closes the search view.
func (s *SearchView) Exit() bool {
	return true
}

// Tabs returns the tab layout for the search view.
func (s *SearchView) Tabs() app.Tab {
	return app.Tab{
		Title: "Search",
		Info: []app.TabInfo{
			{ID: "video", Title: "Videos"},
			{ID: "playlist", Title: "Playlists"},
			{ID: "channel", Title: "Channels"},
		},

		Selected: s.currentType,
	}
}

// Primitive returns the primitive for the search view.
func (s *SearchView) Primitive() tview.Primitive {
	return s.table
}

// ThemeProperty returns the search view's theme property.
func (d *SearchView) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextSearch,
		Item:    theme.ThemeBackground,
	}
}

// Start shows the search view and fetches results for
// the search query.
func (s *SearchView) Start(text string) {
	if text == "" {
		if !s.lock.TryAcquire(1) {
			app.ShowInfo("Still loading Search results", false)
			return
		}
		defer s.lock.Release(1)

		text = s.savedText
		goto StartSearch
	} else {
		client.Cancel()

		s.page = 0
		s.savedText = text
	}

	s.addToHistory(text)

	app.UI.QueueUpdateDraw(func() {
		s.table.Clear()
		s.table.SetSelectable(false, false)

		s.suggestBox.Exit(false)
		s.parametersBox.Exit(false)
		app.UI.Status.SwitchToPage("messages")

		app.SetPrimaryFocus()
	})

StartSearch:
	app.ShowInfo("Fetching results", true)

	results, page, err := inv.Search(s.currentType, text, s.parameters, s.page)
	if err != nil {
		app.ShowError(err)
		return
	}
	if results == nil {
		app.ShowError(fmt.Errorf("View: Search: No more results"))
		return
	}

	s.page = page
	app.UI.QueueUpdateDraw(func() {
		SetView(&Search)
		s.renderResults(results)
	})

	app.ShowInfo("Results fetched", false)
}

// Query displays a prompt and search for the provided query.
func (s *SearchView) Query(switchMode ...struct{}) {
	s.Init()

	app.UI.Status.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextSearch, app.UI.Status.InputField)
	})

	label := "Search (" + s.tab + "):"
	app.UI.Status.SetInput(label, 0, switchMode == nil, Search.Start, Search.inputFunc)
}

// Suggestions shows search suggestions.
func (s *SearchView) Suggestions(text string) {
	if text == s.suggestText && s.suggestBox.Open {
		return
	}

	s.suggestText = text
	s.suggestBox.Exit(true)
	s.suggestBox.Table.Clear()

	suggestions, err := inv.SearchSuggestions(text)
	if err != nil {
		return
	}

	app.UI.QueueUpdateDraw(func() {
		defer app.UI.SetFocus(app.UI.Status.InputField)

		totalSuggestions := len(suggestions.Suggestions)
		if totalSuggestions == 0 {
			s.suggestBox.Exit(true)
			return
		}

		s.suggestBox.Height = totalSuggestions + 1

		for row, suggest := range suggestions.Suggestions {
			s.suggestBox.Table.SetCell(row, 0, theme.NewTableCell(
				theme.ThemeContextSearch,
				theme.ThemeText,
				suggest,
			).
				SetReference(suggest),
			)
		}

		s.suggestBox.Table.Select(0, 0)

		s.suggestBox.Show(true)
	})
}

// Parameters displays a popup to modify the search parameters.
func (s *SearchView) Parameters() {
	if !s.lock.TryAcquire(1) {
		app.ShowInfo("Cannot modify Search parameters", false)
		return
	}
	defer s.lock.Release(1)

	s.parametersForm = s.getParametersForm()

	s.parametersBox.Flex.RemoveItemIndex(2)
	s.parametersBox.Flex.AddItem(s.parametersForm, 0, 1, true)

	app.UI.QueueUpdateDraw(func() {
		s.parametersBox.Show(true)
	})
}

// ParseQuery parses the 'search-video', 'search-playlist'
// and 'search-channel' command-line parameters.
func (s *SearchView) ParseQuery() {
	s.Init()

	stype, query, err := cmd.GetQueryParams("search")
	if err != nil {
		return
	}

	s.currentType = stype
	s.addToHistory(query)

	go Search.Start(query)
}

// Keybindings describes the keybindings for the search view.
func (s *SearchView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch keybinding.KeyOperation(event, keybinding.KeyContextSearch, keybinding.KeyContextComments) {
	case keybinding.KeyLoadMore:
		go s.Start("")
		app.UI.Status.SetFocusFunc()

	case keybinding.KeyClose:
		CloseView()

	case keybinding.KeyQuery:
		s.Query()

	case keybinding.KeyPlaylist:
		Playlist.EventHandler(event.Modifiers() == tcell.ModAlt, false)

	case keybinding.KeyChannelVideos:
		Channel.EventHandler("video", event.Modifiers() == tcell.ModAlt)

	case keybinding.KeyChannelPlaylists:
		Channel.EventHandler("playlist", event.Modifiers() == tcell.ModAlt)

	case keybinding.KeyComments:
		Comments.Show()

	case keybinding.KeyAdd:
		Dashboard.ModifyHandler(true)

	case keybinding.KeyLink:
		popup.ShowLink()
	}

	return event
}

// inputFunc describes the keybindings for the search input box.
func (s *SearchView) inputFunc(e *tcell.EventKey) *tcell.EventKey {
	operation := keybinding.KeyOperation(e, keybinding.KeyContextSearch)

	switch operation {
	case keybinding.KeySearchStart:
		s.currentType = s.tab

		text := app.UI.Status.GetText()
		if text != "" {
			go s.Start(text)
			app.UI.Status.SetFocusFunc()
		}

	case keybinding.KeyClose:
		if s.suggestBox.Open {
			s.suggestBox.Exit(false)
			goto Event
		}

		s.historyReset()

		s.tab = s.currentType
		app.SelectTab(s.currentType)

		app.UI.Status.SetFocusFunc()
		app.UI.Status.SwitchToPage("messages")
		app.SetPrimaryFocus()

	case keybinding.KeySearchSuggestions:
		go s.Suggestions(app.UI.Status.GetText())

	case keybinding.KeySearchSwitchMode:
		var tab app.Tab
		if GetCurrentView() != (&Search) {
			tab = s.Tabs()
			tab.Selected = s.tab
		}

		s.tab = app.SwitchTab(false, tab)
		s.Query(struct{}{})

	case keybinding.KeySearchParameters:
		go s.Parameters()

	case keybinding.KeySearchSuggestionReverse, keybinding.KeySearchSuggestionForward:
		s.suggestionHandler(operation)

	case keybinding.KeySearchHistoryReverse, keybinding.KeySearchHistoryForward:
		if t := s.historyEntry(operation); t != "" {
			app.UI.Status.SetText(t)
		}

	default:
		return e
	}

Event:
	return nil
}

// setupHistory reads the history file and loads the search history.
func (s *SearchView) setupHistory() {
	s.entries = cmd.Settings.SearchHistory
	s.pos = len(s.entries)
}

// addToHistory adds text to the history entries buffer.
func (s *SearchView) addToHistory(text string) {
	if text == "" {
		return
	}

	if len(s.entries) == 0 {
		s.entries = append(s.entries, text)
	} else if text != s.entries[len(s.entries)-1] {
		s.entries = append(s.entries, text)
	}

	s.pos = len(s.entries)
	cmd.Settings.SearchHistory = s.entries
}

// historyEntry returns the search history entry.
func (s *SearchView) historyEntry(key keybinding.Key) string {
	switch key {
	case keybinding.KeySearchHistoryReverse:
		if s.pos-1 < 0 || s.pos-1 >= len(s.entries) {
			var entry string

			if s.entries != nil {
				entry = s.entries[0]
			}

			return entry
		}

		s.pos--

	case keybinding.KeySearchHistoryForward:
		if s.pos+1 >= len(s.entries) {
			var entry string

			if s.entries != nil {
				entry = s.entries[len(s.entries)-1]

			}

			return entry
		}

		s.pos++
	}

	return s.entries[s.pos]
}

// historyReset resets the position in the s.entries buffer.
func (s *SearchView) historyReset() {
	s.pos = len(s.entries)
}

// suggestionHandler handles suggestion popup key events.
func (s *SearchView) suggestionHandler(key keybinding.Key) {
	s.suggestBox.Table.InputHandler()(keybinding.KeyEvent(key), nil)
}

// getParametersForm renders and returns a form to
// modify the search parameters.
//
//gocyclo:ignore
func (s *SearchView) getParametersForm() *tview.Form {
	var form *tview.Form
	var savedFeatures []string

	if f, ok := s.parameters["features"]; ok {
		savedFeatures = strings.Split(f, ",")
	}

	if s.parametersForm.GetFormItemCount() > 0 {
		form = s.parametersForm.Clear(false)
		goto SetContent
	}

	form = theme.NewForm(s.property.SetItem(theme.ThemePopupBackground))
	form.SetItemPadding(2)
	form.SetHorizontal(true)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextSearch) {
		case keybinding.KeySearchParameters:
			s.setParameters()

		case keybinding.KeyClose:
			s.parametersBox.Exit(true)
		}

		return event
	})
	form.AddButton("Set", s.setParameters)
	form.AddButton("Cancel", func() {
		s.parametersBox.Exit(true)
	})

SetContent:
	for label, value := range formParams {
		var options []string
		var savedOption string

		for sp, opts := range value {
			savedOption = s.parameters[sp]
			options = opts
		}

		switch label {
		case "Region:":
			form.AddInputField(label, savedOption, 2, nil, nil)
			continue

		case "Features:":
			for _, o := range options {
				var checked bool

				for _, f := range savedFeatures {
					if f == o {
						checked = true
						break
					}
				}

				defer form.AddCheckbox(o, checked, nil)
			}

		default:
			selected := -1

			for i, o := range options {
				if savedOption == "" {
					break
				}

				if o == savedOption {
					selected = i
				}
			}

			form.AddDropDown(label, options, selected, nil)
		}
	}

	return form
}

// setParameters sets the search parameters.
func (s *SearchView) setParameters() {
	var features []string

	for i := 0; i < s.parametersForm.GetFormItemCount(); i++ {
		var curropt string

		item := s.parametersForm.GetFormItem(i)
		label := item.GetLabel()
		options := formParams[label]

		if list, ok := item.(*tview.DropDown); ok {
			_, curropt = list.GetCurrentOption()
		} else if input, ok := item.(*tview.InputField); ok {
			curropt = input.GetText()
		} else if chkbox, ok := item.(*tview.Checkbox); ok {
			if chkbox.IsChecked() {
				features = append(features, label)
			}

			continue
		}

		for p := range options {
			s.parameters[p] = curropt
		}
	}

	s.parameters["features"] = strings.Join(features, ",")

	s.parametersBox.Exit(true)
	s.parametersForm.Clear(true)
}

// renderResults renders the search view.
func (s *SearchView) renderResults(results []inv.SearchData) {
	var skipped int

	pos := -1
	rows := s.table.GetRowCount()
	_, _, width, _ := app.UI.Pages.GetRect()

	for i, result := range results {
		var author, lentext string

		select {
		case <-client.Ctx().Done():
			s.table.Clear()
			return

		default:
		}

		if result.Type == "category" {
			skipped++
			continue
		}

		if pos < 0 {
			pos = (rows + i) - skipped
		}

		author = result.Author
		if result.Title == "" {
			result.Title = result.Author
			author = ""
		}

		if result.LiveNow {
			lentext = "Live"
		} else {
			lentext = utils.FormatDuration(result.LengthSeconds)
		}

		item := theme.ThemeVideo
		switch result.Type {
		case "playlist":
			item = theme.ThemePlaylist

		case "channel":
			item = theme.ThemeChannel
		}

		actualRow := (rows + i) - skipped

		s.table.SetCell(actualRow, 0, theme.NewTableCell(
			theme.ThemeContextSearch,
			item,
			tview.Escape(result.Title),
		).
			SetExpansion(1).
			SetReference(result).
			SetMaxWidth((width / 4)),
		)

		s.table.SetCell(actualRow, 1, theme.NewTableCell(
			theme.ThemeContextSearch,
			theme.ThemeBackground,
			" ",
		).
			SetSelectable(true).
			SetAlign(tview.AlignRight),
		)

		s.table.SetCell(actualRow, 2, theme.NewTableCell(
			theme.ThemeContextSearch,
			theme.ThemeAuthor,
			tview.Escape(author),
		).
			SetSelectable(true).
			SetMaxWidth((width / 4)).
			SetAlign(tview.AlignLeft),
		)

		s.table.SetCell(actualRow, 3, theme.NewTableCell(
			theme.ThemeContextSearch,
			theme.ThemeBackground,
			" ",
		).
			SetSelectable(true).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "playlist" || result.Type == "channel" {
			s.table.SetCell(actualRow, 4, theme.NewTableCell(
				theme.ThemeContextSearch,
				theme.ThemeTotalVideos,
				strconv.FormatInt(result.VideoCount, 10)+" videos",
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)

			if result.Type == "playlist" {
				continue
			}
		} else {
			s.table.SetCell(actualRow, 4, theme.NewTableCell(
				theme.ThemeContextSearch,
				theme.ThemeTotalDuration,
				lentext,
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}

		s.table.SetCell(actualRow, 5, theme.NewTableCell(
			theme.ThemeContextSearch,
			theme.ThemeBackground,
			" ",
		).
			SetSelectable(true).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "channel" {
			s.table.SetCell(actualRow, 6, theme.NewTableCell(
				theme.ThemeContextSearch,
				theme.ThemeSubscribers,
				utils.FormatNumber(result.SubCount)+" subs",
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		} else {
			s.table.SetCell(actualRow, 6, theme.NewTableCell(
				theme.ThemeContextSearch,
				theme.ThemePublished,
				utils.FormatPublished(result.PublishedText),
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}
	}

	s.table.Select(pos, 0)
	s.table.ScrollToEnd()

	s.table.SetSelectable(true, false)

	if Banner.shown && len(results) > 0 {
		app.UI.Pages.SwitchToPage(Search.Name())
	}
}
