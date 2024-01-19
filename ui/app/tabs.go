package app

import (
	"github.com/darkhz/invidtui/ui/theme"
)

// Tab describes the layout for a tab.
type Tab struct {
	Title, Selected string
	Info            []TabInfo

	Context theme.ThemeContext
}

// TabInfo stores the tab information.
type TabInfo struct {
	ID, Title string
}

var currentTab = &Tab{}

// SetTab sets the tab.
func SetTab(tabInfo Tab, context theme.ThemeContext) {
	if tabInfo.Title == "" {
		UI.Tabs.Clear()
		return
	}

	tabInfo.Context = context
	if context == "" {
		tabInfo.Context = theme.ThemeContextApp
	}
	currentTab = &tabInfo

	builder := theme.NewTextBuilder(tabInfo.Context)
	for _, info := range tabInfo.Info {
		builder.Append(theme.ThemeTabs, info.ID, info.Title)
		builder.AppendText(" ")
	}
	UI.Tabs.SetText(builder.Get())

	SelectTab(tabInfo.Selected)
}

// SelectTab selects a tab.
func SelectTab(tab string) {
	if currentTab.Info == nil {
		return
	}

	UI.Tabs.Highlight(
		theme.FormatRegion(tab, currentTab.Context, theme.ThemeTabs),
	)
}

// GetCurrentTab returns the currently selected tab.
func GetCurrentTab() string {
	tab := UI.Tabs.GetHighlights()
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

	if tabs != nil && tabs[0].Info != nil {
		selected = theme.FormatRegion(tabs[0].Selected, currentTab.Context, theme.ThemeTabs)
		for _, region := range tabs[0].Info {
			regions = append(
				regions,
				theme.FormatRegion(region.ID, currentTab.Context, theme.ThemeTabs),
			)
		}

		goto Selected
	}

	regions = UI.Tabs.GetRegionIDs()
	if len(regions) == 0 {
		return selected
	}

	if highlights := UI.Tabs.GetHighlights(); highlights != nil {
		selected = highlights[0]
	} else {
		return selected
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

	UI.Tabs.Highlight(regions[currentView])
	UI.Tabs.ScrollToHighlight()

	region, _, _ := theme.GetThemeRegion(regions[currentView])
	return region
}
