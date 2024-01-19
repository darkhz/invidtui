package keybinding

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/knadh/koanf/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ParseKeybindings parses the keybindings from the configuration.
func ParseKeybindings(k *koanf.Koanf, dir string) error {
	if !k.Exists("keybindings") {
		return nil
	}

	kbMap := k.StringMap("keybindings")
	if len(kbMap) == 0 {
		return nil
	}

	keyNames := make(map[string]tcell.Key)
	for key, names := range tcell.KeyNames {
		keyNames[names] = key
	}

	for keyType, key := range kbMap {
		if err := checkBindings(keyType, key, keyNames); err != nil {
			return err
		}
	}

	keyErrors := make(map[Keybinding]struct{})
	kbErrors := strings.Builder{}

	fmt.Fprintf(&kbErrors, "Config: The following keybindings will conflict:\n")

	for keyType, keydata := range OperationKeys {
		for existing, data := range OperationKeys {
			if data.Kb == keydata.Kb && data.Title != keydata.Title {
				if data.Context == keydata.Context || data.Global || keydata.Global {
					goto KeyError
				}

				continue

			KeyError:
				if _, ok := keyErrors[keydata.Kb]; !ok {
					keyErrors[keydata.Kb] = struct{}{}
					fmt.Fprintf(&kbErrors, "- %s will override %s (%s)\n", keyType, existing, KeyName(keydata.Kb))
				}
			}
		}
	}

	if len(keyErrors) > 0 {
		return fmt.Errorf(kbErrors.String()[:kbErrors.Len()-1])
	}

	return nil
}

// checkBindings validates the provided keybinding.
//
//gocyclo:ignore
func checkBindings(keyType, key string, keyNames map[string]tcell.Key) error {
	var runes []rune
	var keys []tcell.Key

	if _, ok := OperationKeys[Key(keyType)]; !ok {
		return fmt.Errorf("Config: Invalid key type %s", keyType)
	}

	keybinding := Keybinding{
		Key:  tcell.KeyRune,
		Rune: ' ',
		Mod:  tcell.ModNone,
	}

	tokens := strings.FieldsFunc(key, func(c rune) bool {
		return unicode.IsSpace(c) || c == '+'
	})

	for _, token := range tokens {
		if len(token) > 1 {
			token = cases.Title(language.Und).String(token)
		} else if len(token) == 1 {
			keybinding.Rune = rune(token[0])
			runes = append(runes, keybinding.Rune)

			continue
		}

		if translated, ok := translateKeys[token]; ok {
			token = translated
		}

		switch token {
		case "Ctrl":
			keybinding.Mod |= tcell.ModCtrl

		case "Alt":
			keybinding.Mod |= tcell.ModAlt

		case "Shift":
			keybinding.Mod |= tcell.ModShift

		case "Space", "Plus":
			keybinding.Rune = ' '
			if token == "Plus" {
				keybinding.Rune = '+'
			}

			runes = append(runes, keybinding.Rune)

		default:
			if key, ok := keyNames[token]; ok {
				keybinding.Key = key
				keybinding.Rune = ' '
				keys = append(keys, keybinding.Key)
			}
		}
	}

	if keys != nil && runes != nil || len(runes) > 1 || len(keys) > 1 {
		return fmt.Errorf("Config: More than one key entered for %s (%s)", keyType, key)
	}

	if keybinding.Mod&tcell.ModShift != 0 {
		keybinding.Rune = unicode.ToUpper(keybinding.Rune)

		if unicode.IsLetter(keybinding.Rune) {
			keybinding.Mod &^= tcell.ModShift
		}
	}

	if keybinding.Mod&tcell.ModCtrl != 0 {
		var modKey string

		switch {
		case len(keys) > 0:
			if key, ok := tcell.KeyNames[keybinding.Key]; ok {
				modKey = key
			}

		case len(runes) > 0:
			if keybinding.Rune == ' ' {
				modKey = "Space"
			} else {
				modKey = string(unicode.ToUpper(keybinding.Rune))
			}
		}

		if modKey != "" {
			modKey = "Ctrl-" + modKey
			if key, ok := keyNames[modKey]; ok {
				keybinding.Key = key
				keybinding.Rune = ' '
				keys = append(keys, keybinding.Key)
			}
		}
	}

	if keys == nil && runes == nil {
		return fmt.Errorf("Config: No key specified or invalid keybinding for %s (%s)", keyType, key)
	}

	OperationKeys[Key(keyType)].Kb = keybinding

	return nil
}
