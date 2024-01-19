package cmd

import "github.com/knadh/koanf/v2"

// ConfigHandler describes a configuration handler.
type ConfigHandler interface {
	Parse(k *koanf.Koanf, dir string) error
	Generate(k *koanf.Koanf) (interface{}, error)
}

// ConfigSettings stores the configuration handlers.
type ConfigSettings struct {
	settings map[ConfigType]ConfigHandler
}

// ConfigType describes the configuration type.
type ConfigType string

// The different types of configuration handlers.
const (
	ConfigTheme       ConfigType = "theme"
	ConfigKeybindings ConfigType = "keybindings"
)

var handler ConfigSettings

// RegisterConfigHandler registers a configuration handler.
func RegisterConfigHandler(h ConfigHandler, i ConfigType) {
	if handler.settings == nil {
		handler.settings = make(map[ConfigType]ConfigHandler)
	}

	handler.settings[i] = h
}

// RunAllParsers runs all the stored handler's parsers.
func RunAllParsers() {
	for _, h := range handler.settings {
		if err := h.Parse(config.Koanf, config.path); err != nil {
			PrintError(err.Error())
		}
	}
}

// RunAllGenerators runs all the stored handler's generators.
func RunAllGenerators(genMap map[string]interface{}) {
	for i, h := range handler.settings {
		value, err := h.Generate(config.Koanf)
		if err != nil {
			PrintError(err.Error())
		}

		genMap[string(i)] = value
	}
}
