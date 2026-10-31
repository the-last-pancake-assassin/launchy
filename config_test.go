package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestLoadSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"palette":{"background":"#112233","selected_background":"blue"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	colors, err := resolvePalette(settings.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if colors.background != tcell.GetColor("#112233") {
		t.Fatalf("got background color %v", colors.background)
	}
	if colors.selectedBackground != tcell.GetColor("blue") {
		t.Fatalf("got selected background color %v", colors.selectedBackground)
	}
	if settings.Palette.Foreground != "default" {
		t.Fatalf("omitted foreground should use default, got %q", settings.Palette.Foreground)
	}
}

func TestLoadSettingsRejectsUnknownColorsAndFields(t *testing.T) {
	for _, content := range []string{
		`{"palette":{"background":"not-a-color"}}`,
		`{"palette":{"unknown":"red"}}`,
	} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadSettings(path); err == nil {
			t.Fatalf("expected invalid settings error for %s", content)
		}
	}
}

func TestParseEditorCommand(t *testing.T) {
	args, err := parseEditorCommand(`nvim --cmd "set number" 'file with spaces'`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"nvim", "--cmd", "set number", "file with spaces"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("got %#v, want %#v", args, want)
	}
}

func TestParseEditorCommandRejectsUnterminatedQuote(t *testing.T) {
	_, err := parseEditorCommand(`nvim "unterminated`)
	if err == nil || !strings.Contains(err.Error(), "unterminated quote") {
		t.Fatalf("expected unterminated quote error, got %v", err)
	}
}

func TestSettingsPathUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(configHome, "launchy", "config.json")
	if path != want {
		t.Fatalf("got %q, want %q", path, want)
	}
}

func TestEnsureSettingsFileDoesNotOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launchy", "config.json")
	if err := ensureSettingsFile(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("user settings"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureSettingsFile(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user settings" {
		t.Fatalf("existing settings were overwritten: %q", data)
	}
}
