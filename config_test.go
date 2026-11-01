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
	content := `{"application_directories":["/opt/applications"],"palette":{"background":"#112233","selected_background":"blue"}}`
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
	if !reflect.DeepEqual(settings.ApplicationDirectories, []string{"/opt/applications"}) {
		t.Fatalf("got application directories %#v", settings.ApplicationDirectories)
	}
}

func TestExampleSettingsLoads(t *testing.T) {
	settings, err := loadSettings("config.example.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.ApplicationDirectories) != 3 {
		t.Fatalf("got %d application directories, want 3", len(settings.ApplicationDirectories))
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

func TestExpandApplicationDirectories(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	extra := filepath.Join(t.TempDir(), "extra-apps")
	t.Setenv("LAUNCHY_TEST_APPS", extra)

	directories, err := expandApplicationDirectories([]string{
		"$HOME/.local/share/applications",
		"~/Desktop",
		"$LAUNCHY_TEST_APPS",
	}, home)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(home, ".local", "share", "applications"),
		filepath.Join(home, "Desktop"),
		extra,
	}
	if !reflect.DeepEqual(directories, want) {
		t.Fatalf("got %#v, want %#v", directories, want)
	}
}

func TestExpandApplicationDirectoriesRejectsUnsetVariables(t *testing.T) {
	t.Setenv("LAUNCHY_MISSING_APPS", "")
	_, err := expandApplicationDirectories([]string{"$LAUNCHY_MISSING_APPS/applications"}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unset environment variable") {
		t.Fatalf("expected unset variable error, got %v", err)
	}
}

func TestEnsureSettingsFileDoesNotOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launchy", "config.json")
	content := `{"application_directories":["/opt/apps"],"palette":{"border":"green"}}`
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureSettingsFile(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("existing settings were overwritten: %q", data)
	}
}

func TestEnsureSettingsFileMigratesMissingDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "launchy", "config.json")
	content := `{"palette":{"border":"green","selected_background":"purple"}}`
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureSettingsFile(path); err != nil {
		t.Fatal(err)
	}
	settings, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(settings.ApplicationDirectories, defaultApplicationDirectories()) {
		t.Fatalf("unexpected migrated application directories: %#v", settings.ApplicationDirectories)
	}
	if settings.Palette.Border != "green" || settings.Palette.SelectedBackground != "purple" {
		t.Fatalf("migration changed existing palette: %#v", settings.Palette)
	}
}
