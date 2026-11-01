package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type appSettings struct {
	ApplicationDirectories []string     `json:"application_directories"`
	Palette                colorPalette `json:"palette"`
}

type colorPalette struct {
	Background         string `json:"background"`
	Foreground         string `json:"foreground"`
	Border             string `json:"border"`
	Title              string `json:"title"`
	SelectedBackground string `json:"selected_background"`
	SelectedForeground string `json:"selected_foreground"`
}

type resolvedPalette struct {
	background         tcell.Color
	foreground         tcell.Color
	border             tcell.Color
	title              tcell.Color
	selectedBackground tcell.Color
	selectedForeground tcell.Color
}

func defaultSettings() appSettings {
	return appSettings{
		ApplicationDirectories: defaultApplicationDirectories(),
		Palette: colorPalette{
			Background:         "default",
			Foreground:         "default",
			Border:             "default",
			Title:              "default",
			SelectedBackground: "white",
			SelectedForeground: "black",
		},
	}
}

func settingsPath() (string, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}
	return filepath.Join(configDirectory, "launchy", "config.json"), nil
}

func loadSettings(path string) (appSettings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultSettings(), nil
	}
	if err != nil {
		return appSettings{}, fmt.Errorf("read settings %s: %w", path, err)
	}
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return appSettings{}, errors.New("settings must be a JSON object")
	}

	settings := defaultSettings()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		return appSettings{}, fmt.Errorf("parse settings %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return appSettings{}, fmt.Errorf("parse settings %s: multiple JSON values", path)
		}
		return appSettings{}, fmt.Errorf("parse settings %s: %w", path, err)
	}
	if len(settings.ApplicationDirectories) == 0 {
		return appSettings{}, errors.New("application_directories must contain at least one path")
	}
	for index, directory := range settings.ApplicationDirectories {
		if strings.TrimSpace(directory) == "" {
			return appSettings{}, fmt.Errorf("application_directories[%d] must not be empty", index)
		}
	}
	if _, err := resolvePalette(settings.Palette); err != nil {
		return appSettings{}, fmt.Errorf("validate settings %s: %w", path, err)
	}
	return settings, nil
}

func resolvePalette(palette colorPalette) (resolvedPalette, error) {
	colors := resolvedPalette{}
	fields := []struct {
		name  string
		value string
		color *tcell.Color
	}{
		{"background", palette.Background, &colors.background},
		{"foreground", palette.Foreground, &colors.foreground},
		{"border", palette.Border, &colors.border},
		{"title", palette.Title, &colors.title},
		{"selected_background", palette.SelectedBackground, &colors.selectedBackground},
		{"selected_foreground", palette.SelectedForeground, &colors.selectedForeground},
	}
	for _, field := range fields {
		color, err := parseColor(field.value)
		if err != nil {
			return resolvedPalette{}, fmt.Errorf("palette.%s: %w", field.name, err)
		}
		*field.color = color
	}
	return colors, nil
}

func parseColor(value string) (tcell.Color, error) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "default") {
		return tcell.ColorDefault, nil
	}
	color := tcell.GetColor(strings.ToLower(value))
	if color == tcell.ColorDefault {
		return tcell.ColorDefault, fmt.Errorf("unknown color %q; use a W3C color name, #RRGGBB, or default", value)
	}
	return color, nil
}

func ensureSettingsFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	data, err := json.MarshalIndent(defaultSettings(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode default settings: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return addDefaultApplicationDirectories(path)
	}
	if err != nil {
		return fmt.Errorf("create settings %s: %w", path, err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return fmt.Errorf("write settings %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close settings %s: %w", path, err)
	}
	return nil
}

func addDefaultApplicationDirectories(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read settings %s: %w", path, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("parse settings %s: %w", path, err)
	}
	if fields == nil {
		return errors.New("settings must be a JSON object")
	}
	if _, exists := fields["application_directories"]; exists {
		return nil
	}

	settings, err := loadSettings(path)
	if err != nil {
		return err
	}
	settings.ApplicationDirectories = defaultApplicationDirectories()
	updated, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode migrated settings: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect settings %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to migrate settings symlink %s", path)
	}

	file, err := os.CreateTemp(filepath.Dir(path), ".launchy-config-*")
	if err != nil {
		return fmt.Errorf("create temporary settings file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		file.Close()
		return fmt.Errorf("set migrated settings permissions: %w", err)
	}
	if _, err := file.Write(append(updated, '\n')); err != nil {
		file.Close()
		return fmt.Errorf("write migrated settings: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync migrated settings: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close migrated settings: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace settings %s: %w", path, err)
	}
	return nil
}

func parseEditorCommand(editor string) ([]string, error) {
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}

	var (
		args    []string
		current strings.Builder
		quote   rune
		escaped bool
		started bool
	)
	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}

	for _, char := range editor {
		if escaped {
			current.WriteRune(char)
			escaped = false
			started = true
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if (char == '"' || char == '\'') && (quote == 0 || quote == char) {
			if quote == 0 {
				quote = char
			} else {
				quote = 0
			}
			started = true
			continue
		}
		if quote == 0 && unicode.IsSpace(char) {
			flush()
			continue
		}
		current.WriteRune(char)
		started = true
	}
	if escaped {
		return nil, errors.New("editor command ends with an escape")
	}
	if quote != 0 {
		return nil, errors.New("editor command has an unterminated quote")
	}
	flush()
	if len(args) == 0 || args[0] == "" {
		return nil, errors.New("editor command is empty")
	}
	return args, nil
}
