package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/utils"
)

// SettingsData describes the format to store the application settings.
type SettingsData struct {
	Credentials []client.Credential `json:"credentials"`

	SearchHistory []string              `json:"searchHistory"`
	PlayHistory   []PlayHistorySettings `json:"playHistory"`

	PlayerStates []string `json:"playerStates"`
}

// PlayHistorySettings describes the format to store the play history.
type PlayHistorySettings struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	VideoID    string `json:"videoId"`
	PlaylistID string `json:"playlistId"`
	AuthorID   string `json:"authorId"`
}

// Settings stores the application settings.
var Settings SettingsData

// SaveSettings saves the application settings.
func SaveSettings() {
	Settings.Credentials = client.GetAuthCredentials()

	Settings.SearchHistory = utils.Deduplicate(Settings.SearchHistory)

	data, err := utils.JSON().MarshalIndent(Settings, "", " ")
	if err != nil {
		printer.Error(fmt.Sprintf("Settings: Cannot encode data: %s", err))
	}

	file, err := GetPath("settings.json")
	if err != nil {
		printer.Error("Settings: Cannot get store path")
	}

	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC|os.O_SYNC, os.ModePerm)
	if err != nil {
		printer.Error(fmt.Sprintf("Settings: Cannot open file: %s", err))
	}
	defer fd.Close()

	_, err = fd.Write(data)
	if err != nil {
		printer.Error(fmt.Sprintf("Settings: Cannot save data: %s", err))
	}
}

// getSettings retrives the settings from the settings file.
func getSettings() {
	getOldSettings()

	file, err := GetPath("settings.json")
	if err != nil {
		printer.Error("Settings: Cannot create/get store path")
	}

	fd, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		printer.Error("Settings: Cannot open file")
	}
	defer fd.Close()

	err = utils.JSON().NewDecoder(fd).Decode(&Settings)
	if err != nil && err != io.EOF {
		printer.Error("Settings: Cannot parse values")
	}

	client.SetAuthCredentials(Settings.Credentials)
}

// getOldSettings retreives the settings stored in various files
// and merges them according to the settings format.
func getOldSettings() {
	for _, files := range []struct {
		Type, File string
	}{
		{"Auth", "auth.json"},
		{"Config", "config"},
		{"SearchHistory", "history"},
		{"State", "state"},
		{"PlayHistory", "playhistory.json"},
	} {
		file, err := GetPath(files.File, struct{}{})
		if err != nil {
			continue
		}

		fd, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
		if err != nil {
			continue
		}

		decoder := utils.JSON().NewDecoder(fd)

		switch files.Type {
		case "Auth":
			err = decoder.Decode(&Settings.Credentials)

		case "PlayHistory":
			err = decoder.Decode(&Settings.PlayHistory)

		case "Config", "State", "SearchHistory":
			scanner := bufio.NewScanner(fd)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				switch files.Type {
				case "Config":
					values := strings.Split(line, "=")
					if len(values) != 2 {
						continue
					}

					SetOptionValue(values[0], values[1])

				case "State":
					Settings.PlayerStates = strings.Split(line, ",")

				case "SearchHistory":
					Settings.SearchHistory = append(Settings.SearchHistory, line)
				}
			}

			err = scanner.Err()
		}

		fd.Close()

		if err != nil && err != io.EOF {
			printer.Error(fmt.Sprintf("Settings: Could not parse %s", files.File))
		}
	}
}
