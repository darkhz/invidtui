package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

var (
	// ResultsFlex contains the arranged page marker
	// and ResultsList table elements.
	ResultsFlex *tview.Flex

	// ResultsList is a table to display results.
	ResultsList    *tview.Table
	resultPageMark *tview.TextView

	suggestionList *tview.Table

	listWidth     int
	searchLock    *semaphore.Weighted
	stype         string
	searchString  string
	suggestChange string
)

const loadingText = "Search still in progress, please wait"

// SetupList sets up a table to display search results.
func SetupList() {
	ResultsList = tview.NewTable()
	ResultsList.SetBorder(false)
	ResultsList.SetSelectorWrap(true)
	ResultsList.SetBackgroundColor(tcell.ColorDefault)

	resultPageMark = tview.NewTextView()
	resultPageMark.SetWrap(false)
	resultPageMark.SetRegions(true)
	resultPageMark.SetDynamicColors(true)
	resultPageMark.SetBackgroundColor(tcell.ColorDefault)
	resultPageMark.SetText(
		`[::b]Search[-:-:-] ["video"][darkcyan]Videos[""] ["playlist"][darkcyan]Playlists[""] ["channel"][darkcyan]Channels[""]`,
	)

	box := tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault)

	ResultsFlex = tview.NewFlex().
		AddItem(resultPageMark, 1, 0, false).
		AddItem(box, 1, 0, false).
		AddItem(ResultsList, 0, 10, true).
		SetDirection(tview.FlexRow)

	ResultsFlex.SetBackgroundColor(tcell.ColorDefault)

	ResultsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		captureListEvents(event)
		capturePlayerEvent(event)

		return event
	})

	suggestionList = tview.NewTable()
	suggestionList.SetSelectorWrap(true)
	suggestionList.SetSelectable(true, false)
	suggestionList.SetBackgroundColor(tcell.ColorDefault)
	suggestionList.SetSelectionChangedFunc(func(row, column int) {
		cell := suggestionList.GetCell(row, 0)

		InputBox.SetText(cell.Text)
	})

	searchLock = semaphore.NewWeighted(1)

	toggleSearch()
}

// SearchAndList displays search results on the screen.
func SearchAndList(text string) {
	var getmore bool

	if !searchLock.TryAcquire(1) {
		InfoMessage(loadingText, false)
		return
	}
	defer searchLock.Release(1)

	if text == "" && searchString == "" {
		return
	}

	msg := "Fetching "
	if text != "" {
		getmore = false
		searchString = text
	} else {
		getmore = true
		msg += "more "
	}

	InfoMessage(msg+stype+" results for '"+tview.Escape(searchString)+"'", true)

	results, err := lib.GetClient().Search(stype, searchString, getmore)
	if err != nil {
		ErrorMessage(err)
		return
	}

	if results == nil {
		ErrorMessage(fmt.Errorf("No more results"))
		return
	}

	App.QueueUpdateDraw(func() {
		searchAndList(results)
	})

	InfoMessage("Results fetched", false)
}

// searchAndList renders the search results list.
func searchAndList(results []lib.SearchResult) {
	var skipped int

	pos := -1
	rows := ResultsList.GetRowCount()
	_, _, width, _ := VPage.GetRect()

	for i, result := range results {
		select {
		case <-lib.SearchCtx().Done():
			ResultsList.Clear()
			return

		default:
		}
		var lentext string

		if result.Type == "category" {
			skipped++
			continue
		}

		if pos < 0 {
			pos = (rows + i) - skipped
		}

		if result.Title == "" {
			result.Title = result.Author
			result.Author = ""
		}

		if result.LiveNow {
			lentext = "Live"
		} else {
			lentext = lib.FormatDuration(result.LengthSeconds)
		}

		actualRow := (rows + i) - skipped

		ResultsList.SetCell(actualRow, 0, tview.NewTableCell("[blue::b]"+tview.Escape(result.Title)).
			SetExpansion(1).
			SetReference(result).
			SetMaxWidth((width / 4)).
			SetSelectedStyle(mainStyle),
		)

		ResultsList.SetCell(actualRow, 1, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		ResultsList.SetCell(actualRow, 2, tview.NewTableCell("[purple::b]"+result.Author).
			SetSelectable(true).
			SetMaxWidth((width / 4)).
			SetAlign(tview.AlignLeft).
			SetSelectedStyle(auxStyle),
		)

		ResultsList.SetCell(actualRow, 3, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "playlist" || result.Type == "channel" {
			ResultsList.SetCell(actualRow, 4, tview.NewTableCell("[pink]"+strconv.Itoa(result.VideoCount)+" videos").
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)

			if result.Type == "playlist" {
				continue
			}
		} else {
			ResultsList.SetCell(actualRow, 4, tview.NewTableCell("[pink]"+lentext).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}

		ResultsList.SetCell(actualRow, 5, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "channel" {
			ResultsList.SetCell(actualRow, 6, tview.NewTableCell("[pink]"+lib.FormatNumber(result.SubCount)+" subs").
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		} else {
			ResultsList.SetCell(actualRow, 6, tview.NewTableCell("[pink]"+lib.FormatPublished(result.PublishedText)).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}
	}

	ResultsList.Select(pos, 0)
	ResultsList.ScrollToEnd()

	ResultsList.SetSelectable(true, false)

	if bannerShown && len(results) > 0 {
		bannerShown = false
		VPage.SwitchToPage("search")
	}
}

// showLinkPopup shows a popup with links.
func showLinkPopup() {
	info, err := getListReference()
	if err != nil {
		ErrorMessage(err)
		return
	}

	invlink, ytlink := lib.GetLinks(info)
	linkText := "[::u]Invidious link[-:-:-]\n[::b]" + invlink +
		"\n\n[::u]Youtube link[-:-:-]\n[::b]" + ytlink

	linkTitle := tview.NewTextView()
	linkTitle.SetDynamicColors(true)
	linkTitle.SetTextAlign(tview.AlignCenter)
	linkTitle.SetText("[white::bu]Copy link")
	linkTitle.SetBackgroundColor(tcell.ColorDefault)

	linkPopup := tview.NewTextView()
	linkPopup.SetText(linkText)
	linkPopup.SetDynamicColors(true)
	linkPopup.SetBackgroundColor(tcell.ColorDefault)
	linkPopup.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		captureSendPlayerEvent(event)

		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyEscape:
			exitFocus()
		}

		return event
	})

	linkFlex := tview.NewFlex().
		AddItem(linkTitle, 1, 0, false).
		AddItem(linkPopup, 10, 10, false).
		SetDirection(tview.FlexRow)

	MPage.AddAndSwitchToPage(
		"linkpage",
		statusmodal(linkFlex, linkPopup),
		true,
	).ShowPage("ui")

	App.SetFocus(linkPopup)
}

// searchSuggestions shows a popup with search recommendations.
func searchSuggestions(text string) {
	if text == suggestChange {
		return
	}

	suggestChange = text

	suggestion, err := lib.GetClient().Suggestions(text)
	if err != nil {
		return
	}

	App.QueueUpdateDraw(func() {
		if len(suggestion.Suggestions) == 0 {
			MPage.HidePage("suggestion")
			App.SetFocus(InputBox)

			return
		}

		suggestionList.Clear()

		for row, suggest := range suggestion.Suggestions {
			suggestionList.SetCell(row, 0, tview.NewTableCell(suggest).
				SetSelectedStyle(auxStyle),
			)
		}

		suggestionList.Select(0, 0)

		suggestFlex := tview.NewFlex().
			AddItem(tview.NewBox().SetBackgroundColor(tcell.ColorDefault), 1, 0, false).
			AddItem(suggestionList, 0, 1, false).
			SetDirection(tview.FlexRow)

		if pg, _ := MPage.GetFrontPage(); pg != "suggestion" {
			MPage.AddAndSwitchToPage(
				"suggestion",
				statusmodal(suggestFlex, suggestionList),
				true,
			).ShowPage("ui")
		}

		resizemodal()

		App.SetFocus(InputBox)
	})
}

// captureListEvents binds keys to ResultsList's InputCapture.
func captureListEvents(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyEnter:
		loadMoreResults()
	}

	switch event.Rune() {
	case '/':
		searchText(event.Modifiers() == tcell.ModAlt)

	case 'i':
		ViewPlaylist(true, event.Modifiers() == tcell.ModAlt)

	case 'u':
		ViewChannel("video", true, event.Modifiers() == tcell.ModAlt)

	case 'U':
		ViewChannel("playlist", true, event.Modifiers() == tcell.ModAlt)

	case 'C':
		ShowComments()

	case '+':
		go Modify(true)

	case ';':
		showLinkPopup()
	}
}

// resizeListEntries detects if the screen is resized, and resizes
// each TableCell's text to an appropriate width.
func resizeListEntries(width int) {
	if listWidth == width {
		return
	}

	go func() {
		if !searchLock.TryAcquire(1) {
			return
		}
		defer searchLock.Release(1)

		App.QueueUpdateDraw(func() {
			table := getListTable()
			if table == nil {
				return
			}

			for i := 0; i < table.GetRowCount(); i++ {
				for j := 0; j < table.GetColumnCount(); j++ {
					cell := table.GetCell(i, j)
					if cell == nil || cell.NotSelectable {
						continue
					}

					cell.SetMaxWidth((width / 3))
				}
			}

			pos, _ := table.GetSelection()
			ResultsList.Select(pos, 0)
		})
	}()

	listWidth = width
}

// searchText takes the search string from user input,
// clears ResultsList, and displays new search results.
//
//gocyclo:ignore
func searchText(channel bool) {
	srchfn := func(text string) {
		if channel {
			SearchChannel(text)
		} else {
			SearchAndList(text)
		}
	}

	srchfocus := func() *tview.Table {
		var table *tview.Table

		if channel {
			table = chSearchTable
		} else {
			table = ResultsList
		}

		Status.SwitchToPage("messages")
		MPage.RemovePage("suggestion")
		MPage.RemovePage("searchparam")

		App.SetFocus(table)

		return table
	}

	sfunc := func(text string) {
		table := srchfocus()

		if text != "" {
			lib.AddToHistory(text)
			table.Clear()
			table.SetSelectable(false, false)
			resultPageMark.Highlight(stype)
			lib.SearchCancel()
		} else {
			return
		}

		go srchfn(text)
	}

	ifunc := func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:
			sfunc(InputBox.GetText())

		case tcell.KeyTab:
			go searchSuggestions(InputBox.GetText())

		case tcell.KeyEscape:
			srchfocus()
			lib.HistoryReset()

		case tcell.KeyUp:
			if e.Modifiers() == tcell.ModCtrl {
				suggestionList.InputHandler()(tcell.NewEventKey(tcell.KeyUp, ' ', tcell.ModNone), nil)
				return e
			}

			t := lib.HistoryReverse()
			if t != "" {
				InputBox.SetText(t)
			}

		case tcell.KeyDown:
			if e.Modifiers() == tcell.ModCtrl {
				suggestionList.InputHandler()(tcell.NewEventKey(tcell.KeyDown, ' ', tcell.ModNone), nil)
				return e
			}

			t := lib.HistoryForward()
			if t != "" {
				InputBox.SetText(t)
			}

		case tcell.KeyCtrlE:
			if channel {
				return nil
			}

			InputBox.SetLabel(toggleSearch())
		}

		switch e.Rune() {
		case 'e':
			if e.Modifiers() == tcell.ModAlt {
				go searchParamPopup()
			}
		}

		return e
	}

	label := "Search"

	if channel {
		label += " channel:"
		if ResultsList.HasFocus() {
			err := ViewChannel("search", true, false)
			if err != nil {
				ErrorMessage(err)
				return
			}
		}
	} else {
		label += " (" + stype + "):"
	}

	SetInput(label, 0, sfunc, ifunc)
}

// toggleSearch toggles the search type.
func toggleSearch() string {
	switch stype {
	case "video":
		stype = "playlist"

	case "playlist":
		stype = "channel"

	case "channel":
		fallthrough

	default:
		stype = "video"
	}

	return "[::b]Search (" + stype + "): "
}

// loadMoreResults appends more search results to ResultsList
func loadMoreResults() {
	go SearchAndList("")
}

// searchParamPopup displays a popup with modifiable search parameters.
//
//gocyclo:ignore
func searchParamPopup() {
	App.QueueUpdateDraw(func() {
		var paramForm *tview.Form
		var savedFeatures []string

		if !searchLock.TryAcquire(1) {
			InfoMessage(loadingText, false)
			return
		}
		defer searchLock.Release(1)

		params := lib.GetSearchParams()
		if params == nil {
			params = make(map[string]string)
		}

		if f, ok := params["features"]; ok {
			savedFeatures = strings.Split(f, ",")
		}

		selparams := map[string]map[string][]string{
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

		exit := func() {
			exitFocus()
			App.SetFocus(InputBox)
		}

		setparams := func() {
			var features []string

			for i := 0; i < paramForm.GetFormItemCount(); i++ {
				var curropt string

				item := paramForm.GetFormItem(i)
				label := item.GetLabel()
				optMap := selparams[label]

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

				for p := range optMap {
					params[p] = curropt
				}
			}

			params["features"] = strings.Join(features, ",")

			lib.SetSearchParams(params)

			exit()
		}

		paramForm = tview.NewForm()
		paramForm.SetItemPadding(2)
		paramForm.SetHorizontal(true)
		paramForm.SetBackgroundColor(tcell.ColorDefault)
		paramForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape:
				exit()
			}

			switch event.Rune() {
			case 'e':
				if event.Modifiers() == tcell.ModAlt {
					setparams()
				}
			}

			return event
		})

		for splabel, spvalue := range selparams {
			var options []string
			var savedOption string

			for sp, opts := range spvalue {
				savedOption = params[sp]
				options = opts
			}

			switch splabel {
			case "Region:":
				paramForm.AddInputField(splabel, savedOption, 2, nil, nil)
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

					defer paramForm.AddCheckbox(o, checked, nil)
				}

			default:
				selOptIndex := -1

				for i, o := range options {
					if savedOption == "" {
						break
					}

					if o == savedOption {
						selOptIndex = i
					}
				}

				paramForm.AddDropDown(splabel, options, selOptIndex, nil)
			}
		}

		paramForm.AddButton("Set parameters", setparams)

		paramForm.AddButton("Cancel", func() {
			exit()
		})

		paramTitle := tview.NewTextView()
		paramTitle.SetDynamicColors(true)
		paramTitle.SetText("[::bu]Search Parameters")
		paramTitle.SetTextAlign(tview.AlignCenter)
		paramTitle.SetBackgroundColor(tcell.ColorDefault)

		paramFlex := tview.NewFlex().
			AddItem(paramTitle, 1, 0, false).
			AddItem(paramForm, 10, 10, true).
			SetDirection(tview.FlexRow)

		MPage.AddAndSwitchToPage(
			"searchparam",
			statusmodal(paramFlex, paramForm),
			true,
		).ShowPage("ui")

		App.SetFocus(paramFlex)
	})
}

// parseSearchCmd parses the search type and query from
// the command-line options.
func parseSearchCmd() {
	searchtype, searchquery, err := lib.GetSearchQuery()
	if err != nil {
		return
	}

	VPage.SwitchToPage("search")

	stype = searchtype
	resultPageMark.Highlight(stype)

	lib.AddToHistory(searchquery)
	go SearchAndList(searchquery)
}

// getListTable gets the Table in current focus.
func getListTable() *tview.Table {
	item := App.GetFocus()

	if item, ok := item.(*tview.Table); ok {
		return item
	}

	return nil
}

// getListReference gets a saved reference from a selected TableCell.
func getListReference() (lib.SearchResult, error) {
	var table *tview.Table

	err := fmt.Errorf("Cannot select this entry")

	table = getListTable()
	if table == nil {
		return lib.SearchResult{}, err
	}

	row, _ := table.GetSelection()

	for col := 0; col <= 1; col++ {
		cell := table.GetCell(row, col)
		if cell == nil {
			return lib.SearchResult{}, err
		}

		info, ok := cell.GetReference().(lib.SearchResult)
		if ok {
			return info, nil
		}
	}

	return lib.SearchResult{}, err
}

// modifyListReference modifies a TableCell containing the specified reference.
func modifyListReference(title string, add bool, info ...lib.SearchResult) error {
	err := fmt.Errorf("Cannot modify list entry")

	table := getListTable()
	if table == nil {
		return err
	}

	for i := 0; i < table.GetRowCount(); i++ {
		cell := table.GetCell(i, 0)
		if cell == nil {
			continue
		}

		ref := cell.GetReference()
		if ref == nil {
			continue
		}

		if info[0] == ref.(lib.SearchResult) {
			if add {
				cell.SetText(title)
				cell.SetReference(info[1])
			} else {
				table.RemoveRow(i)
			}

			break
		}
	}

	return nil
}
