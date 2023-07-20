package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/darkhz/invidtui/platform"
	"github.com/hjson/hjson-go/v4"
	"github.com/knadh/koanf/v2"
)

// Config describes the configuration for the app.
type Config struct {
	path string

	mutex sync.Mutex

	*koanf.Koanf
}

var config Config

// Init sets up the configuration.
func (c *Config) setup() {
	var configExists bool

	c.Koanf = koanf.New(".")

	homedir, err := os.UserHomeDir()
	if err != nil {
		printer.Error(err.Error())
	}

	dirs := []string{".config/invidtui", ".invidtui"}
	for i, dir := range dirs {
		p := filepath.Join(homedir, dir)
		dirs[i] = p

		if _, err := os.Stat(p); err == nil {
			c.path = p
			return
		}

		if i > 0 {
			continue
		}

		if _, err := os.Stat(filepath.Clean(filepath.Dir(p))); err == nil {
			configExists = true
		}
	}

	if c.path == "" {
		var pos int
		var err error

		if configExists {
			err = os.Mkdir(dirs[0], 0700)
		} else {
			pos = 1
			err = os.Mkdir(dirs[1], 0700)
		}

		if err != nil {
			printer.Error(err.Error())
		}

		c.path = dirs[pos]
	}
}

// GetPath returns the full config path for the provided file type.
func GetPath(ftype string, nocreate ...struct{}) (string, error) {
	var cfpath string

	if ftype == "socket" {
		socket := filepath.Join(config.path, "socket")
		cfpath = platform.Socket(socket)

		if _, err := os.Stat(socket); err == nil {
			if !IsOptionEnabled("close-instances") {
				return "", fmt.Errorf("Config: Socket exists at %s, is another instance running?", socket)
			}

			if err := os.Remove(socket); err != nil {
				return "", fmt.Errorf("Config: Cannot remove %s", socket)
			}
		}

		fd, err := os.OpenFile(socket, os.O_CREATE, os.ModeSocket)
		if err != nil {
			return "", fmt.Errorf("Config: Cannot create socket file at %s", socket)
		}
		fd.Close()

		return cfpath, nil
	}

	cfpath = filepath.Join(config.path, ftype)

	if nocreate != nil {
		_, err := os.Stat(cfpath)
		return cfpath, err
	}

	fd, err := os.OpenFile(cfpath, os.O_CREATE, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("Config: Cannot create %s file at %s", ftype, cfpath)
	}
	fd.Close()

	return cfpath, nil
}

// GetQueryParams returns the parameters for the search and play option types.
func GetQueryParams(queryType string) (string, string, error) {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	for _, option := range options {
		if option.Type != queryType {
			continue
		}

		value := config.String(option.Name)
		if value == "" {
			continue
		}

		t := strings.Split(option.Name, "-")
		if len(t) != 2 {
			return "", "", fmt.Errorf("Config: Invalid query type")
		}

		return t[1], value, nil
	}

	return "", "", fmt.Errorf("Config: Query type not found")
}

// GetOptionValue returns a value for an option
// from the configuration store.
func GetOptionValue(key string) string {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	return config.String(key)
}

// SetOptionValue sets a value for an option
// in the configuration store.
func SetOptionValue(key string, value interface{}) {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	config.Set(key, value)
}

// IsOptionEnabled returns if an option is enabled.
func IsOptionEnabled(key string) bool {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	return config.Bool(key)
}

// generateConfig generates and updates the configuration.
// Any existing values are appended to it.
func generateConfig() {
	genMap := make(map[string]interface{})

	for _, option := range options {
		for _, name := range []string{
			"force-instance",
			"download-dir",
			"num-retries",
			"video-res",
		} {
			if option.Type == "path" || option.Name == name {
				genMap[option.Name] = config.Get(option.Name)
			}
		}
	}

	keys := config.Get("keybindings")
	if keys == nil {
		keys = make(map[string]interface{})
	}
	genMap["keybindings"] = keys

	data, err := hjson.Marshal(genMap)
	if err != nil {
		printer.Error(err.Error())
	}

	conf, err := GetPath("invidtui.conf")
	if err != nil {
		printer.Error(err.Error())
	}

	file, err := os.OpenFile(conf, os.O_WRONLY, os.ModePerm)
	if err != nil {
		printer.Error(err.Error())
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		printer.Error(err.Error())
		return
	}

	if err := file.Sync(); err != nil {
		printer.Error(err.Error())
		return
	}
}
