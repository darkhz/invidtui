package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	connretries     int
	fcSocket        bool
)

// SetupFlags sets up the commandline flags
func SetupFlags() error {
	var validres bool

	flag.StringVar(
		&videoResolution,
		"video-res",
		"720p",
		"Set the default video resolution.",
	)

	flag.BoolVar(
		&fcSocket,
		"close-instances",
		false,
		"Close all currently running instances.",
	)

	flag.StringVar(
		&mpvpath,
		"mpv-path",
		"mpv",
		"Specify path to the mpv executable.",
	)

	flag.StringVar(
		&ytdlpath,
		"ytdl-path",
		"youtube-dl",
		"Specify path to youtube-dl executable or its forks (yt-dlp, yt-dtlp_x86)",
	)

	flag.IntVar(
		&connretries,
		"num-retries",
		100,
		"Set the number of retries for connecting to the socket.",
	)

	config, err := ConfigPath("config")
	if err != nil {
		return err
	}
	flag.CommandLine.ParseFile(config)

	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"invidtui [<flags>]\n\nConfig file is %s\n\nFlags:\n",
			config,
		)

		flag.CommandLine.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf("  --%s", f.Name)

			if len(s) <= 4 {
				s += "\t"
			} else {
				s += "\n    \t"
			}
			s += strings.ReplaceAll(f.Usage, "\n", "\n    \t")

			if f.Name != "close-instances" {
				if f.Name != "num-retries" {
					s += fmt.Sprintf(" (default %q)", f.DefValue)
				} else {
					s += fmt.Sprintf(" (default %v)", f.DefValue)
				}
			}

			fmt.Fprint(flag.CommandLine.Output(), s, "\n")
		})
	}

	flag.Parse()

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

	_, err = exec.LookPath(ytdlpath)
	if err != nil {
		return fmt.Errorf("Could not find the youtube-dl executable")
	}

	_, err = exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("Could not find the ffmpeg executable")
	}

	return nil
}

// SetupConfig checks for the config directory, and creates one if it
// doesn't exist.
func SetupConfig() error {
	var tpath string
	var dotConfigExists bool

	configDirs := []string{".config/invidtui", ".invidtui"}

	fullpath, err := homedir.Expand("~")
	if err != nil {
		return fmt.Errorf("Error: Cannot get home directory")
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
				return fmt.Errorf("Error: Cannot create %s", configDirs[0])
			}

		} else {

			err := os.Mkdir(configDirs[1], 0700)
			if err != nil {
				return fmt.Errorf("Error: Cannot create %s", configDirs[1])
			}
		}

		configPath = tpath
	}

	return nil
}

// ConfigPath returns the absolute path for the given filetype:
// socket, history and config, and performs actions related to it.
func ConfigPath(ftype string) (string, error) {
	switch ftype {
	case "socket":
		sockPath = filepath.Join(configPath, "socket")
		socket := getSocket(sockPath)

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

			CloseInstances(socket)
		}

		return socket, nil

	case "history":
		hpath := filepath.Join(configPath, "history")

		if _, err := os.Stat(hpath); err != nil {
			fd, err := os.Create(hpath)
			fd.Close()
			if err != nil {
				return "", fmt.Errorf("Cannot create history file at %s", hpath)
			}
		}

		return hpath, nil

	case "config":
		cpath := filepath.Join(configPath, "config")

		if _, err := os.Stat(cpath); err != nil {
			fd, err := os.Create(cpath)
			fd.Close()
			if err != nil {
				return "", fmt.Errorf("Cannot create config file at %s", cpath)
			}
		}

		return cpath, nil
	}

	return configPath, nil
}
