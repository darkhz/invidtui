package theme

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/darkhz/invidtui/cmd"
	"github.com/gdamore/tcell/v2"
	"github.com/knadh/koanf/parsers/hjson"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

// ThemeConfig describes the theme configuration settings.
type ThemeConfig struct {
	settings map[ThemeContext]map[ThemeItem]ThemeSetting

	sync.Mutex
}

// ThemeSetting describes a theme setting.
type ThemeSetting struct {
	Style tcell.Style
	Tag   string
}

var config ThemeConfig

// GetConfigHandler returns the theme configuration handler.
func GetConfigHandler() *ThemeConfig {
	return &config
}

// Parse parses the theme from the configuration.
func (c *ThemeConfig) Parse(k *koanf.Koanf, dir string) error {
	themeFile := k.String("theme")
	if themeFile == "" {
		return parseConfig(nil)
	}

	if filepath.Ext(themeFile) == "" {
		themeFile += ".theme"
	}

	conf := koanf.New(".")
	if err := conf.Load(file.Provider(filepath.Join(dir, "themes", themeFile)), hjson.Parser()); err != nil {
		return err
	}

	return parseConfig(conf)
}

// Generate generates the theme configuration.
func (c *ThemeConfig) Generate(k *koanf.Koanf) (interface{}, error) {
	themeFile := k.String("theme")
	if themeFile == "" {
		themeFile = "default"
	}

	themePath, err := cmd.GetPath("themes/default.theme")
	if err != nil {
		return themeFile, err
	}

	fd, err := os.OpenFile(themePath, os.O_WRONLY, os.ModePerm)
	if err != nil {
		return themeFile, err
	}

	info, err := fd.Stat()
	if err == nil && info.Size() == 0 {
		_, err = fd.WriteString(DefaultThemeConfig)
	}

	return themeFile, err
}

// GetThemeSetting returns a style and tag according to the ThemeProperty.
func GetThemeSetting(p ThemeProperty) (tcell.Style, string, bool) {
	config.Lock()
	defer config.Unlock()

	var setting ThemeSetting

	if config.settings == nil {
		goto Setting
	}

	for _, context := range []ThemeContext{
		p.Context,
		ThemeContextApp,
	} {
		settingMap, ok := config.settings[context]
		if !ok {
			continue
		}

		s, ok := settingMap[p.Item]
		if ok {
			setting = s
			break
		}
	}

Setting:
	return setting.Style, setting.Tag, setting != (ThemeSetting{})
}

// parseConfig parses the current configuration or the default.
func parseConfig(k *koanf.Koanf) error {
	if k == nil {
		k = koanf.New(".")
		if err := k.Load(rawbytes.Provider([]byte(DefaultThemeConfig)), hjson.Parser()); err != nil {
			return err
		}
	}

	keyErrors := bytes.Buffer{}
	fmt.Fprintf(&keyErrors, "Config: The following theme directives will conflict:\n")
	length := keyErrors.Len()

	settings := make(map[ThemeContext]map[ThemeItem]ThemeSetting)
	for _, key := range k.Keys() {
		split := strings.Split(key, ".")
		if len(split) != 2 {
			fmt.Fprintf(&keyErrors, "- Invalid theme context/item: %s\n", strings.ReplaceAll(key, ".", " -> "))
			continue
		}

		context, item := ThemeContext(split[0]), ThemeItem(split[1])
		if _, ok := ThemeScopes[context]; !ok {
			fmt.Fprintf(&keyErrors, "- Invalid theme context: %s\n", context)
			continue
		}
		if _, ok := ThemeScopes[context][item]; !ok {
			fmt.Fprintf(&keyErrors, "- Item '%s' is not in scope for context '%s'\n", item, context)
			continue
		}

		setting := k.String(key)
		if setting == "" {
			fmt.Fprintf(&keyErrors, "- Empty theme directive for '%s -> %s'\n", context, item)
			continue
		}

		style, tag, err := parseThemeSetting(setting)
		if err != nil {
			fmt.Fprintf(&keyErrors, "- Invalid theme directive for '%s -> %s' (%s)\n", context, item, err.Error())
			continue
		}
		if settings[context] == nil {
			settings[context] = make(map[ThemeItem]ThemeSetting)
		}

		settings[context][item] = ThemeSetting{
			Style: style,
			Tag:   tag,
		}
	}

	if keyErrors.Len() > length {
		length = keyErrors.Len()
		return errors.New(keyErrors.String()[:length-1])
	}

	config.Lock()
	config.settings = settings
	config.Unlock()

	return nil
}

// parseThemeSetting parses a theme setting and returns a style and a tag.
func parseThemeSetting(setting string) (tcell.Style, string, error) {
	var style tcell.Style
	var styleAttr tcell.AttrMask

	var tagAttr []rune
	var tagColor [2]string

	attrMap := map[string][2]byte{
		"bold":      {'b', byte(tcell.AttrBold)},
		"underline": {'u', byte(tcell.AttrUnderline)},
		"italic":    {'i', byte(tcell.AttrItalic)},
		"blink":     {'l', byte(tcell.AttrBlink)},
		"dim":       {'d', byte(tcell.AttrDim)},
	}

	split := strings.Split(setting, ";")
	if len(split) == 0 {
		return tcell.StyleDefault, "", fmt.Errorf("Empty parameter")
	}

	for _, s := range split {
		s = strings.TrimSpace(s)
		property := strings.Split(s, ":")
		if len(property) != 2 {
			return tcell.StyleDefault, "", fmt.Errorf("'%s' has invalid parameter length", s)
		}

		prop, value := property[0], property[1]
		values := strings.Split(value, ",")

		switch p := strings.TrimSpace(prop); p {
		case "attr":
			for _, v := range values {
				if a, ok := attrMap[strings.TrimSpace(v)]; ok {
					tagAttr = append(tagAttr, rune(a[0]))
					styleAttr |= tcell.AttrMask(a[1])
				}
			}

		case "fg", "bg":
			var color tcell.Color

			name := ""
			if len(values) > 0 {
				name = values[0]
			}

			switch name {
			case "black":
				color = tcell.Color16

			case "default":
				color = tcell.ColorDefault

			default:
				color = tcell.GetColor(name)
			}
			if color == 0 {
				return tcell.Style{}, "", fmt.Errorf("Invalid color '%s'", name)
			}

			switch p {
			case "fg":
				tagColor[0] = name
				style = style.Foreground(color)

			case "bg":
				tagColor[1] = name
				style = style.Background(color)
			}

		default:
			return tcell.Style{}, "", fmt.Errorf("Invalid option '%s'", p)
		}
	}

	style = style.Attributes(styleAttr)
	tag := fmt.Sprintf(
		"[%s:%s:%s]",
		tagColor[0], tagColor[1], string(tagAttr),
	)

	return style, tag, nil
}
