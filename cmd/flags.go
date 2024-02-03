package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/platform"
	"github.com/darkhz/invidtui/utils"
	"github.com/knadh/koanf/parsers/hjson"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
)

// Option describes a command-line option.
type Option struct {
	Name, Description string
	Value, Type       string
}

var options = []Option{
	{
		Name:        "token",
		Description: "Specify an authorization token. Use with --force-instance.",
		Value:       "",
		Type:        "auth",
	},
	{
		Name:        "mpv-path",
		Description: "Specify path to the mpv executable.",
		Value:       "mpv",
		Type:        "path",
	},
	{
		Name:        "ytdl-path",
		Description: "Specify path to youtube-dl executable or its forks (yt-dlp, yt-dlp_x86)",
		Value:       "",
		Type:        "path",
	},
	{
		Name:        "ffmpeg-path",
		Description: "Specify path to ffmpeg executable.",
		Value:       "ffmpeg",
		Type:        "path",
	},
	{
		Name:        "download-dir",
		Description: "Specify directory to download media into.",
		Value:       "",
		Type:        "path",
	},
	{
		Name:        "search-video",
		Description: "Search for a video.",
		Value:       "",
		Type:        "search",
	},
	{
		Name:        "search-playlist",
		Description: "Search for a playlist.",
		Value:       "",
		Type:        "search",
	},
	{
		Name:        "search-channel",
		Description: "Search for a channel.",
		Value:       "",
		Type:        "search",
	},
	{
		Name:        "play-audio",
		Description: "Specify video/playlist URL to play audio from.",
		Value:       "",
		Type:        "play",
	},
	{
		Name:        "play-video",
		Description: "Specify video/playlist URL to play video from.",
		Value:       "",
		Type:        "play",
	},
	{
		Name:        "video-res",
		Description: "Set the default video resolution.",
		Value:       "720p",
		Type:        "other",
	},
	{
		Name:        "num-retries",
		Description: "Set the number of retries for connecting to the socket.",
		Value:       "100",
		Type:        "other",
	},
	{
		Name:        "force-instance",
		Description: "Force load media from specified invidious instance.",
		Value:       "",
		Type:        "other",
	},
	{
		Name:        "theme",
		Description: "Specify theme file to apply on startup.",
		Value:       "",
		Type:        "other",
	},
	{
		Name:        "close-instances",
		Description: "Close all currently running instances.",
		Value:       "",
		Type:        "bool",
	},
	{
		Name:        "show-instances",
		Description: "Show a list of instances.",
		Value:       "",
		Type:        "bool",
	},
	{
		Name:        "token-link",
		Description: "Display a link to the token generation page.",
		Value:       "",
		Type:        "bool",
	},
	{
		Name:        "version",
		Description: "Print version information.",
		Value:       "",
		Type:        "bool",
	},
	{
		Name:        "generate",
		Description: "Generate configuration",
		Value:       "",
		Type:        "bool",
	},
}

// parse parses the command-line parameters.
func parse() {
	configFile, err := GetPath("invidtui.conf")
	if err != nil {
		printer.Error(err.Error())
	}

	fs := flag.NewFlagSet("invidtui", flag.ContinueOnError)
	fs.Usage = func() {
		var usage string

		usage += fmt.Sprintf(
			"invidtui [<flags>]\n\nConfig file is %s\n\nFlags:\n",
			configFile,
		)

		fs.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf("  --%s", f.Name)

			if len(s) <= 4 {
				s += "\t"
			} else {
				s += "\n    \t"
			}
			s += strings.ReplaceAll(f.Usage, "\n", "\n    \t")

			for _, name := range []string{
				"token",
				"token-link",
				"search-video",
				"search-channel",
				"search-playlist",
				"show-instances",
				"play-audio",
				"play-video",
				"force-instance",
				"close-instances",
				"version",
				"download-dir",
			} {
				if f.Name == name {
					goto cmdOutPrint
				}
			}

			if f.Name != "num-retries" {
				s += fmt.Sprintf(" (default %q)", f.DefValue)
			} else {
				s += fmt.Sprintf(" (default %v)", f.DefValue)
			}

		cmdOutPrint:
			usage += fmt.Sprintf(s + "\n")
		})

		printer.Print(usage, 0)
	}

	for _, option := range options {
		switch option.Type {
		case "bool":
			fs.Bool(option.Name, false, option.Description)

		default:
			fs.String(option.Name, option.Value, option.Description)
		}
	}

	if err = fs.Parse(os.Args[1:]); err != nil {
		printer.Error(err.Error())
	}

	if err := config.Load(file.Provider(configFile), hjson.Parser()); err != nil {
		printer.Error(err.Error())
	}

	if err := config.Load(posflag.Provider(fs, ".", config.Koanf), nil); err != nil {
		printer.Error(err.Error())
	}
}

// check validates all the command-line and configuration values.
func check() {
	RunAllParsers()
	getSettings()

	checkSocket()
	checkAuth()

	for _, option := range options {
		switch option.Type {
		case "path":
			checkExecutablePaths(option.Name, GetOptionValue(option.Name))

		case "other":
			checkOtherOptions(option.Name, GetOptionValue(option.Name))
		}
	}
}

// checkSocket checks for and creates the socket. It parses the 'close-instances' command-line parameter.
// If close-instances is not set and the socket exists, a message will be shown.
// Otherwise, the socket is created/reused, and any pending connections to the socket are closed.
func checkSocket() error {
	socket := filepath.Join(config.path, "socket")
	cfpath := platform.Socket(socket)

	_, err := os.Stat(socket)
	if err == nil {
		if !IsOptionEnabled("close-instances") {
			printer.Error(fmt.Sprintf("Socket exists at %s, is another instance running?", socket))
		}
	}

	fd, err := os.OpenFile(socket, os.O_CREATE, os.ModeSocket|os.ModePerm)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		printer.Error(fmt.Sprintf("Cannot create socket file at %s", socket))
	}
	fd.Close()

	config.mutex.Lock()
	config.socket = cfpath
	config.mutex.Unlock()

	return nil
}

// checkAuth parses and checks the 'token' and 'token-link' command-line parameters.
// If token-link is set, it will print a link to generate an authentication token.
func checkAuth() {
	var instance string

	token := GetOptionValue("token")
	generateLink := IsOptionEnabled("token-link")
	customInstance := GetOptionValue("force-instance")

	if (generateLink || token != "") && customInstance == "" {
		printer.Error("Instance is not specified")
	}

	instance = utils.GetHostname(customInstance)
	if generateLink {
		printer.Print(client.AuthLink(instance), 0)
	}

	if token == "" {
		return
	}

	printer.Print("Authenticating")

	client.SetHost(instance)
	if !client.IsTokenValid(token) {
		printer.Error("Invalid token or authentication timeout")
	}

	client.AddAuth(instance, token)

	SetOptionValue("instance-validated", true)
}

// checkExecutablePaths checks the mpv, youtube-dl and ffmpeg
// application paths and the download directory.
func checkExecutablePaths(pathType, path string) {
	if pathType != "ytdl-path" && path == "" {
		return
	}

	switch pathType {
	case "download-dir":
		if dir, err := os.Stat(path); err != nil || !dir.IsDir() {
			printer.Error(fmt.Sprintf("Cannot access %s for downloads\n", path))
		}

	case "ytdl-path":
		for _, ytdl := range []string{
			path,
			"youtube-dl",
			"yt-dlp",
			"yt-dlp_x86",
		} {
			if _, err := exec.LookPath(ytdl); err == nil {
				SetOptionValue("ytdl-path", ytdl)
				return
			}
		}

		if GetOptionValue("ytdl-path") == "" {
			printer.Error("Could not find the youtube-dl/yt-dlp/yt-dlp_x86 executables")
		}

	default:
		if _, err := exec.LookPath(path); err != nil {
			printer.Error(fmt.Sprintf("%s: Could not find %s", pathType, path))
		}
	}
}

// checkOtherOptions parses and checks the command-line parameters
// related to the 'other' option type.
func checkOtherOptions(otherType, other string) {
	var resValid bool

	if other == "" {
		return
	}

	switch otherType {
	case "force-instance":
		if _, err := utils.IsValidURL(other); err != nil {
			printer.Error("Invalid instance URL")
		}

	case "num-retries":
		if _, err := strconv.Atoi(other); err != nil {
			printer.Error("Invalid value for num-retries")
		}

	case "video-res":
		for _, res := range []string{
			"144p",
			"240p",
			"360p",
			"480p",
			"720p",
			"1080p",
			"1440p",
			"2160p",
		} {
			if res == other {
				resValid = true
			}
		}
	}

	if otherType == "video-res" && !resValid {
		printer.Error("Invalid video resolution")
	}
}
