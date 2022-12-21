package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jnovack/flag"
	"github.com/mitchellh/go-homedir"
)

var (
	sockPath   string
	configPath string

	videoResolution string
	mpvpath         string
	ytdlpath        string
	vidsearch       string
	plistsearch     string
	channelsearch   string
	playaudio       string
	playvideo       string
	connretries     int
	fcSocket        bool
	currInstance    bool
	instanceList    bool
	customInstance  string
	downloadFolder  string
	authToken       string
	genTokenLink    bool
)

// SetupFlags sets up the commandline flags
//
//gocyclo:ignore
func SetupFlags() error {
	var validres bool

	fs := flag.NewFlagSetWithEnvPrefix("invidtui", "INVIDTUI", flag.ExitOnError)

	fs.StringVar(
		&videoResolution,
		"video-res",
		"720p",
		"Set the default video resolution.",
	)

	fs.BoolVar(
		&fcSocket,
		"close-instances",
		false,
		"Close all currently running instances.",
	)

	fs.BoolVar(
		&currInstance,
		"use-current-instance",
		false,
		"Use the current invidious instance to retrieve media.",
	)

	fs.BoolVar(
		&instanceList,
		"show-instances",
		false,
		"Show a list of instances.",
	)

	fs.BoolVar(
		&genTokenLink,
		"token-link",
		false,
		"Display a link to the token generation page.",
	)

	fs.StringVar(
		&customInstance,
		"force-instance",
		"",
		"Force load media from specified invidious instance.",
	)

	fs.StringVar(
		&mpvpath,
		"mpv-path",
		"mpv",
		"Specify path to the mpv executable.",
	)

	fs.StringVar(
		&ytdlpath,
		"ytdl-path",
		"",
		"Specify path to youtube-dl executable or its forks (yt-dlp, yt-dtlp_x86)",
	)

	fs.StringVar(
		&vidsearch,
		"search-video",
		"",
		"Search for a video.",
	)

	fs.StringVar(
		&plistsearch,
		"search-playlist",
		"",
		"Search for a playlist.",
	)

	fs.StringVar(
		&channelsearch,
		"search-channel",
		"",
		"Search for a channel.",
	)

	fs.StringVar(
		&playaudio,
		"play-audio",
		"",
		"Specify video/playlist URL to play audio from.",
	)

	fs.StringVar(
		&playvideo,
		"play-video",
		"",
		"Specify video/playlist URL to play video from.",
	)

	fs.StringVar(
		&downloadFolder,
		"download-dir",
		"",
		"Specify directory to download media into.",
	)

	fs.StringVar(
		&authToken,
		"token",
		"",
		"Specify an authorization token. "+
			"This has to be used along with the --force-instance option.",
	)
	fs.IntVar(
		&connretries,
		"num-retries",
		100,
		"Set the number of retries for connecting to the socket.",
	)

	config, err := ConfigPath("config")
	if err != nil {
		return err
	}
	fs.ParseFile(config)

	fs.Usage = func() {
		fmt.Fprintf(
			fs.Output(),
			"invidtui [<flags>]\n\nConfig file is %s\n\nFlags:\n",
			config,
		)

		fs.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf("  --%s", f.Name)

			if len(s) <= 4 {
				s += "\t"
			} else {
				s += "\n    \t"
			}
			s += strings.ReplaceAll(f.Usage, "\n", "\n    \t")

			if f.Name == "ytdl-path" {
				s += fmt.Sprintf(" (default %q)", "youtube-dl")
			} else {
				for _, name := range []string{
					"token",
					"token-link",
					"search-video",
					"search-channel",
					"search-playlist",
					"show-instances",
					"play-audio",
					"play-video",
					"close-instances",
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
			}

		cmdOutPrint:
			fmt.Fprint(fs.Output(), s, "\n")
		})
	}

	fs.Parse(os.Args[1:])

	for _, q := range []string{
		"144p",
		"240p",
		"360p",
		"480p",
		"720p",
		"1080p",
		"1440p",
		"2160p",
	} {
		if q == videoResolution {
			validres = true
			break
		}
	}

	if !validres {
		return fmt.Errorf("%s is not a valid video resolution", videoResolution)
	}

	_, err = exec.LookPath(mpvpath)
	if err != nil {
		return fmt.Errorf("Could not find the mpv executable")
	}

	_, err = exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("Could not find the ffmpeg executable")
	}

	err = findYoutubeDL()
	if err != nil {
		return err
	}

	if downloadFolder != "" {
		if dir, err := os.Stat(downloadFolder); err != nil || !dir.IsDir() {
			return fmt.Errorf("Cannot access %s for downloads", downloadFolder)
		}
	}

	return nil
}

// SetupConfig checks for the config directory, and creates one if it
// doesn't exist.
func SetupConfig() error {
	var dotConfigExists bool

	configDirs := []string{".config/invidtui", ".invidtui"}

	fullpath, err := homedir.Expand("~")
	if err != nil {
		return fmt.Errorf("Cannot get home directory")
	}

	for i, cd := range configDirs {
		p := filepath.Join(fullpath, cd)
		configDirs[i] = p

		if _, err := os.Stat(p); err == nil {
			configPath = p
			return nil
		}

		if i == 0 {
			if _, err := os.Stat(
				filepath.Clean(filepath.Dir(p)),
			); err == nil {
				dotConfigExists = true
			}
		}
	}

	if configPath == "" {
		if dotConfigExists {

			err := os.Mkdir(configDirs[0], 0700)
			if err != nil {
				return fmt.Errorf("Cannot create %s", configDirs[0])
			}

			configPath = configDirs[0]

		} else {

			err := os.Mkdir(configDirs[1], 0700)
			if err != nil {
				return fmt.Errorf("Cannot create %s", configDirs[1])
			}

			configPath = configDirs[1]
		}
	}

	return nil
}

// ConfigPath returns the absolute path for the given filetype:
// socket, history and config, and performs actions related to it.
func ConfigPath(ftype string) (string, error) {
	var cfpath string

	switch ftype {
	case "socket":
		sockPath = filepath.Join(configPath, "socket")
		cfpath = getSocket(sockPath)

		if _, err := os.Stat(sockPath); err != nil {
			fd, err := os.Create(sockPath)
			fd.Close()
			if err != nil {
				return "", fmt.Errorf("Cannot create socket file at %s", sockPath)
			}

		} else {
			if !fcSocket {
				return "", fmt.Errorf("Socket exists at %s, is another instance running?", sockPath)
			}

			CloseInstances(cfpath)
		}

	default:
		cfpath = filepath.Join(configPath, ftype)

		if _, err := os.Stat(cfpath); err != nil {
			fd, err := os.Create(cfpath)
			fd.Close()
			if err != nil {
				return "", fmt.Errorf("Cannot create %s file at %s", ftype, cfpath)
			}
		}
	}

	return cfpath, nil
}

// GetSearchQuery returns the search type and query from
// the command-line options.
func GetSearchQuery() (string, string, error) {
	if vidsearch != "" {
		return "video", vidsearch, nil
	}

	if plistsearch != "" {
		return "playlist", plistsearch, nil
	}

	if channelsearch != "" {
		return "channel", channelsearch, nil
	}

	return "", "", fmt.Errorf("No search query specified")
}

// GetPlayParams returns the video URL and media type to play.
func GetPlayParams() (string, bool, error) {
	if playaudio != "" {
		return playaudio, true, nil
	}

	if playvideo != "" {
		return playvideo, false, nil
	}

	return "", false, fmt.Errorf("No player parameters specified")
}

// findYoutubeDL searches for the youtube-dl or yt-dlp executables.
func findYoutubeDL() error {
	if ytdlpath != "" {
		_, err := exec.LookPath(ytdlpath)
		if err != nil {
			return fmt.Errorf("Could not find the " + ytdlpath + " executable")
		}

		return nil
	}

	for _, ytdl := range []string{
		"youtube-dl",
		"yt-dlp",
		"yt-dtlp_x86",
	} {
		_, err := exec.LookPath(ytdl)
		if err == nil {
			ytdlpath = ytdl
			return nil
		}
	}

	return fmt.Errorf("Could not find the youtube-dl/yt-dlp/yt-dtlp_x86 executables")
}

// CheckAuthConfig checks and loads the token and instance.
func CheckAuthConfig() (string, error) {
	if (genTokenLink || authToken != "") && customInstance == "" {
		return "", fmt.Errorf("Instance is not specified")
	}

	instanceName := GetHostname(customInstance)

	if genTokenLink {
		return GetAuthLink(instanceName), nil
	}

	err := LoadAuth()
	if err != nil {
		return "", err
	}

	if authToken != "" {
		AddAuth(instanceName, authToken)
		if err := UpdateClient(); err != nil {
			return "", err
		}
		if !AuthTokenValid() {
			return "", fmt.Errorf("Invalid token")
		}
	}

	return "", nil
}

// ListInstances prints a list of instances.
func ListInstances() (string, error) {
	var list string

	if !instanceList {
		return "", nil
	}

	instances, err := GetInstanceList()
	if err != nil {
		return "", err
	}

	list += "Instances list:\n"
	list += strings.Repeat("-", len(list)) + "\n"
	for i, instance := range instances {
		list += strconv.Itoa(i+1) + ": " + instance + "\n"
	}

	return list, nil
}
