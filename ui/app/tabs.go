package app

import "fmt"

// Tab describes the layout for a tab.
type Tab struct {
	Title, Selected string
	Info            []TabInfo
}

// TabInfo stores the tab information.
type TabInfo struct {
	ID, Title string
}

// SetTab sets the tab.
func SetTab(tabInfo Tab) {
	if tabInfo.Title == "" {
		UI.MenuTabs.Clear()
		return
	}

	tab := fmt.Sprintf("[::b]%s[-:-:-] ", tabInfo.Title)
	for _, info := range tabInfo.Info {
		tab += fmt.Sprintf("[\"%s\"][darkcyan]%s[\"\"] ", info.ID, info.Title)
	}

	UI.MenuTabs.SetText(tab)

	SelectTab(tabInfo.Selected)
}

// SelectTab selects a tab.
func SelectTab(tab string) {
	UI.MenuTabs.Highlight(tab)
}

// GetCurrentTab returns the currently selected tab.
func GetCurrentTab() string {
	tab := UI.MenuTabs.GetHighlights()
	if tab == nil {
		return ""
	}

	return tab[0]
}

// SwitchTab handles the tab selection.
// If reverse is set, the previous tab is selected and vice-versa.
func SwitchTab(reverse bool, tabs ...Tab) string {
	var currentView int
	var selected string
	var regions []string

	if tabs != nil {
		selected = tabs[0].Selected
		for _, region := range tabs[0].Info {
			regions = append(regions, region.ID)
		}

		goto Selected
	}

	regions = UI.MenuTabs.GetRegionIDs()
	if len(regions) == 0 {
		return ""
	}

	if highlights := UI.MenuTabs.GetHighlights(); highlights != nil {
		selected = highlights[0]
	} else {
		return ""
	}

Selected:
	for i, region := range regions {
		if region == selected {
			currentView = i
		}
	}

	if reverse {
		currentView--
	} else {
		currentView++
	}

	if currentView >= len(regions) {
		currentView = 0
	} else if currentView < 0 {
		currentView = len(regions) - 1
	}

	UI.MenuTabs.Highlight(regions[currentView])
	UI.MenuTabs.ScrollToHighlight()

	return regions[currentView]
}
