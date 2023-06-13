package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/utils"
	"github.com/jnovack/flag"
)

// Parameters describes the command-line parameter types.
type Parameters struct {
	stringValue map[string]map[string]*string
	boolValue   map[string]*bool

	err string
}

// Option describes a command-line option.
type Option struct {
	name, description string
	value, optionType string
}

var options = []Option{
	{
		name:        "token",
		description: "Specify an authorization token. Use with --force-instance.",
		value:       "",
		optionType:  "auth",
	},
	{
		name:        "mpv-path",
		description: "Specify path to the mpv executable.",
		value:       "mpv",
		optionType:  "path",
	},
	{
		name:        "ytdl-path",
		description: "Specify path to youtube-dl executable or its forks (yt-dlp, yt-dtlp_x86)",
		value:       "youtube-dl",
		optionType:  "path",
	},
	{
		name:        "ffmpeg-path",
		description: "Specify path to ffmpeg executable.",
		value:       "ffmpeg",
		optionType:  "path",
	},
	{
		name:        "download-dir",
		description: "Specify directory to download media into.",
		value:       "",
		optionType:  "path",
	},
	{
		name:        "search-video",
		description: "Search for a video.",
		value:       "",
		optionType:  "search",
	},
	{
		name:        "search-playlist",
		description: "Search for a playlist.",
		value:       "",
		optionType:  "search",
	},
	{
		name:        "search-channel",
		description: "Search for a channel.",
		value:       "",
		optionType:  "search",
	},
	{
		name:        "play-audio",
		description: "Specify video/playlist URL to play audio from.",
		value:       "",
		optionType:  "play",
	},
	{
		name:        "play-video",
		description: "Specify video/playlist URL to play video from.",
		value:       "",
		optionType:  "play",
	},
	{
		name:        "video-res",
		description: "Set the default video resolution.",
		value:       "720p",
		optionType:  "other",
	},
	{
		name:        "num-retries",
		description: "Set the number of retries for connecting to the socket.",
		value:       "100",
		optionType:  "other",
	},
	{
		name:        "force-instance",
		description: "Force load media from specified invidious instance.",
		value:       "",
		optionType:  "other",
	},
	{
		name:        "close-instances",
		description: "Close all currently running instances.",
		value:       "",
		optionType:  "bool",
	},
	{
		name:        "use-current-instance",
		description: "Use the current invidious instance to retrieve media.",
		value:       "",
		optionType:  "bool",
	},
	{
		name:        "show-instances",
		description: "Show a list of instances.",
		value:       "",
		optionType:  "bool",
	},
	{
		name:        "token-link",
		description: "Display a link to the token generation page.",
		value:       "",
		optionType:  "bool",
	},
	{
		name:        "version",
		description: "Print version information.",
		value:       "",
		optionType:  "bool",
	},
}

var parameters Parameters

// parse parses the command-line parameters.
func parse() {
	parameters.stringValue = make(map[string]map[string]*string)
	parameters.boolValue = make(map[string]*bool)

	fs := flag.NewFlagSetWithEnvPrefix("invidtui", "INVIDTUI", flag.ContinueOnError)
	fs.SetOutput(&parameters)

	for _, option := range options {
		var s string
		var b bool

		if option.optionType == "bool" {
			fs.BoolVar(
				&b,
				option.name,
				false,
				option.description,
			)

			parameters.boolValue[option.name] = &b

			continue
		}

		if v := parameters.stringValue[option.optionType]; v == nil {
			parameters.stringValue[option.optionType] = make(map[string]*string)
		}

		fs.StringVar(
			&s,
			option.name,
			option.value,
			option.description,
		)

		parameters.stringValue[option.optionType][option.name] = &s
	}

	configFile, err := GetPath("config")
	if err != nil {
		printer.Error(err.Error())
	}
	fs.ParseFile(configFile)

	fs.Usage = func() {
		var usage string

		usage += fmt.Sprintf(
			"invidtui [<flags>]\n\nConfig file is %s\n\nFlags:\n",
			configFile,
		)

		if parameters.err != "" {
			usage = fmt.Sprintf("%s\n", parameters.err) + usage
		}

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
				"use-current-instance",
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

		if parameters.err != "" {
			printer.Error(usage)
		}

		printer.Print(usage, 0)
	}

	fs.Parse(os.Args[1:])
}

// Write writes parameter parsing errors to the screen.
func (p *Parameters) Write(b []byte) (int, error) {
	p.err = string(b)

	return 0, nil
}

// checkAuth parses and checks the 'token' and 'token-link' command-line parameters.
// If token-link is set, it will print a link to generate an authentication token.
func checkAuth() {
	var instance string

	token := *parameters.stringValue["auth"]["token"]
	generateLink := IsOptionEnabled("token-link")
	customInstance := GetOptionValue("force-instance")

	if (generateLink || token != "") && customInstance == "" {
		printer.Error("Instance is not specified")
	}

	instance = utils.GetHostname(customInstance)
	if generateLink {
		printer.Print(client.AuthLink(instance), 0)
	}

	authFile, err := GetPath("auth.json")
	if err != nil {
		printer.Error(err.Error())
	}
	err = client.LoadAuthFile(authFile)
	if err != nil {
		printer.Error(err.Error())
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

	EnableOption("instance-validated")
}

// checkQuery parses and checks the command-line parameters
// related to the 'search' and 'play' option types.
func checkQuery() {
	for _, valueType := range []string{"search", "play"} {
		for queryType, query := range parameters.stringValue[valueType] {
			if *query != "" {
				SetOptionValue(queryType, *query)
				break
			}
		}
	}
}

// checkExecutablePaths checks the mpv, youtube-dl and ffmpeg
// application paths and the download directory.
func checkExecutablePaths() {
CheckPath:
	for pathType, path := range parameters.stringValue["path"] {
		if pathType == "download-dir" && *path != "" {
			if dir, err := os.Stat(*path); err != nil || !dir.IsDir() {
				printer.Error(fmt.Sprintf("Cannot access %s for downloads\n", *path))
			}

			continue
		}

		if pathType == "ytdl-path" && *path == "" {
			for _, ytdl := range []string{
				"youtube-dl",
				"yt-dlp",
				"yt-dtlp_x86",
			} {
				_, err := exec.LookPath(ytdl)
				if err == nil {
					parameters.stringValue["path"]["ytdl-path"] = &ytdl
					continue CheckPath
				}
			}
		}

		if *path == "" {
			continue
		}

		if _, err := exec.LookPath(*path); err != nil {
			printer.Error(fmt.Sprintf("%s: Could not find %s", pathType, *path))
		}
	}

	if *parameters.stringValue["path"]["ytdl-path"] == "" {
		printer.Error("Could not find the youtube-dl/yt-dlp/yt-dtlp_x86 executables")
	}

	for p, path := range parameters.stringValue["path"] {
		SetOptionValue(p, *path)
	}
}

// handleBoolOptions parses and checks the command-line parameters
// related to the 'bool' option type.
func handleBoolOptions() {
	for boolType, enabled := range parameters.boolValue {
		if !*enabled {
			continue
		}

		EnableOption(boolType)
	}
}

// handleOtherOptions parses and checks the command-line parameters
// related to the 'other' option type.
func handleOtherOptions() {
	var resValid bool

	for otherType, other := range parameters.stringValue["other"] {
		if otherType == "force-instance" && *other != "" {
			if _, err := utils.IsValidURL(*other); err != nil {
				printer.Error("Invalid instance URL")
			}
		}

		if otherType == "num-retries" {
			if _, err := strconv.Atoi(*other); err != nil {
				printer.Error("Invalid value for num-retries")
			}
		}

		if otherType == "video-res" {
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
				if res == *other {
					resValid = true
					continue
				}
			}
		}

		SetOptionValue(otherType, *other)
	}

	if !resValid {
		printer.Error("Invalid video resolution")
	}
}
