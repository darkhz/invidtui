package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/client"
	mp "github.com/darkhz/invidtui/mediaplayer"
)

// Version stores the version information.
var Version string

// Init parses the command-line parameters and initializes the application.
func Init() {
	printer.setup()
	config.setup()

	parse()

	printVersion()
	generate()

	client.Init()
	printInstances()

	check()

	loadInstance()
	loadPlayer()

	printer.Stop()
}

// loadInstance selects an instance.
func loadInstance() {
	if IsOptionEnabled("instance-validated") {
		return
	}

	customInstance := GetOptionValue("force-instance")

	msg := "Selecting an instance"
	if customInstance != "" {
		msg = "Checking " + customInstance
	}

	printer.Print(msg)

	instance, err := client.GetBestInstance(customInstance)
	if err != nil {
		printer.Error(err.Error())
	}

	client.SetHost(instance)
}

// loadPlayer loads the media player.
func loadPlayer() {
	printer.Print("Starting player")

	socketpath, err := GetPath("socket")
	if err != nil {
		printer.Error(err.Error())
	}

	err = mp.Init(
		"mpv",
		GetOptionValue("mpv-path"),
		GetOptionValue("ytdl-path"),
		GetOptionValue("num-retries"),
		client.UserAgent,
		socketpath,
	)
	if err != nil {
		printer.Error(err.Error())
	}
}

// printVersion prints the version information.
func printVersion() {
	if !IsOptionEnabled("version") {
		return
	}

	text := "InvidTUI v%s"

	versionInfo := strings.Split(Version, "@")
	if len(versionInfo) < 2 {
		printer.Print(fmt.Sprintf(text, versionInfo), 0)
	}

	text += " (%s)"
	printer.Print(fmt.Sprintf(text, versionInfo[0], versionInfo[1]), 0)
}

// printInstances prints a list of instances.
func printInstances() {
	var list string

	if !IsOptionEnabled("show-instances") {
		return
	}

	printer.Print("Retrieving instances")

	instances, err := client.GetInstances()
	if err != nil {
		printer.Error(fmt.Sprintf("Error retrieving instances: %s", err.Error()))
	}

	list += "Instances list:\n"
	list += strings.Repeat("-", len(list)) + "\n"
	for i, instance := range instances {
		list += strconv.Itoa(i+1) + ": " + instance + "\n"
	}

	printer.Print(list, 0)
}

// generate generates the configuration.
func generate() {
	if !IsOptionEnabled("generate") {
		return
	}

	generateConfig()

	printer.Print("Configuration is generated", 0)
}
