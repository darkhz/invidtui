package main

import (
	"github.com/darkhz/invidtui/cmd"
	"github.com/darkhz/invidtui/ui"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
)

func main() {
	cmd.RegisterConfigHandler(theme.GetConfigHandler(), cmd.ConfigTheme)
	cmd.RegisterConfigHandler(keybinding.GetConfigHandler(), cmd.ConfigKeybindings)

	cmd.Init()

	ui.SetupUI()

	cmd.SaveSettings()
}
