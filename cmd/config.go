package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/darkhz/invidtui/platform"
)

// Config describes the configuration for the app.
type Config struct {
	path string

	values  map[string]string
	enabled map[string]struct{}

	mutex sync.Mutex
}

var config Config

// setup sets up the configuration.
func (c *Config) setup() {
	var configExists bool

	printer.Print("Loading configuration")

	c.values = make(map[string]string)
	c.enabled = make(map[string]struct{})

	homedir, err := os.UserHomeDir()
	if err != nil {
		printer.Error(fmt.Sprintf("Cannot get home directory: %s\n", err.Error()))
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
			printer.Error(fmt.Sprintf("Cannot create %s", dirs[pos]))
		}

		c.path = dirs[pos]
	}
}

// GetPath returns the full config path for the provided file type.
func GetPath(ftype string) (string, error) {
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

		goto ReturnPath
	}

	cfpath = filepath.Join(config.path, ftype)

	if fd, err := os.OpenFile(cfpath, os.O_CREATE, os.ModePerm); err != nil {
		return "", fmt.Errorf("Config: Cannot create %s file at %s", ftype, cfpath)
	} else {
		fd.Close()
	}

ReturnPath:
	return cfpath, nil
}

// GetQueryParams returns the parameters for the search and play option types.
func GetQueryParams(queryType string) (string, string, error) {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	for key, value := range config.values {
		t := strings.Split(key, "-")
		if len(t) != 2 {
			return "", "", fmt.Errorf("Config: Invalid query type")
		}

		if t[0] != queryType {
			continue
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

	return config.values[key]
}

// SetOptionValue sets a value for an option
// in the configuration store.
func SetOptionValue(key, value string) {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	config.values[key] = value
}

// IsOptionEnabled returns if an option is enabled.
func IsOptionEnabled(key string) bool {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	_, ok := config.enabled[key]

	return ok
}

// EnableOption enables an option.
func EnableOption(key string) {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	config.enabled[key] = struct{}{}
}
