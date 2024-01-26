https://github.com/darkhz/invidtui/assets/44058754/a73a689c-98fc-46d7-862b-e6b3e4e72f40


## Setup
To setup the theme configuration:
- Generate the theme configuration first
```
invidtui --generate
```

- Then, either use the `theme` command-line option:
```
invidtui --theme=default
```

Or, set the theme file in the configuration:
```
{
    theme: default
}
```

Note that the default configuration is inbuilt, so these steps aren't necessary unless you plan to use more themes.

## Usage
To apply a theme (assuming the default keybinding is set), press <kbd>T</kbd>, navigate to your preferred theme file,
and press <kbd>Enter</kbd> to apply.

## Documentation
A brief documentation is provided in the default theme file.
Extensive documentation will be updated shortly in the main documentation site.

All theme files must be present in the **themes** directory in your configuration directory.
Every theme file preferably must have the `.theme` extension.

## Contribution
When submitting a theme file, ensure that the theme configuration is valid.
The theme file must be present in the `themes` directory.

Then, within the THEMES.md file, add:
- A heading with the name of the theme, and
- A screenshot/gif of the applied theme.
