package main

import (
	"fmt"
	"os"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/invidtui/ui"
)

func errMessage(err string) {
	fmt.Fprintf(os.Stderr, "\rError: %s\n", err)
}

func infoMessage(info string) {
	fmt.Printf("\r%s", info)
}

func main() {
	var err error

	err = lib.SetupConfig()
	if err != nil {
		errMessage(err.Error())
		return
	}

	err = lib.SetupFlags()
	if err != nil {
		errMessage(err.Error())
		return
	}

	infoMessage("Starting MPV instance...")
	err = lib.MPVStart()
	if err != nil {
		errMessage(err.Error())
		return
	}

	infoMessage("Querying invidious instances...")
	err = lib.UpdateClient()
	if err != nil {
		lib.GetMPV().MPVStop(true)
		errMessage(err.Error())
		return
	}

	infoMessage("")

	lib.SetupHistory()

	ui.SetupUI()

	lib.SaveHistory()
}
