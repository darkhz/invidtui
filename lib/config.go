package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	configPath string

	videoResolution *string
	fcSocket        *bool
)

// SetupFlags sets up the commandline flags
func SetupFlags() error {
	var validres bool

	videoResolution = kingpin.Flag(
		"video-res", "Set the default video resolution.").
		Default("720p").String()

	fcSocket = kingpin.Flag(
		"close-instances", "Close all currently running instances.").
		Default("false").Bool()

	kingpin.Parse()

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
		if q == *videoResolution {
			validres = true
			break
		}
	}

	if !validres {
		return fmt.Errorf("%s is not a valid video resolution", *videoResolution)
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
		spath := filepath.Join(configPath, "socket")

		if _, err := os.Stat(spath); err != nil {
			_, err = os.Create(spath)
			if err != nil {
				return "", fmt.Errorf("Cannot create socket file at %s", spath)
			}
		} else {
			if !*fcSocket {
				return "", fmt.Errorf("Socket exists at %s, is another instance running?", spath)
			}

			CloseInstances(spath)
		}

		return spath, nil

	case "history":
		hpath := filepath.Join(configPath, "history")

		if _, err := os.Stat(hpath); err != nil {
			_, err = os.Create(hpath)
			if err != nil {
				return "", fmt.Errorf("Cannot create history file at %s", hpath)
			}
		}

		return hpath, nil

	// TODO: Implement config.yaml
	case "config":
		cpath := filepath.Join(configPath, "config.yaml")

		if _, err := os.Stat(cpath); err != nil {
			_, err = os.Create(cpath)
			if err != nil {
				return "", fmt.Errorf("Cannot create config file at %s", cpath)
			}
		}

		return cpath, nil
	}

	return configPath, nil
}
