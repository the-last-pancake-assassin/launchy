package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseDesktopEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.desktop")
	content := "[Desktop Entry]\nName=Sample App\nExec=sample --flag %U\nType=Application\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	app, err := parseDesktopEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if app.Name != "Sample App" || app.Exec != "sample --flag %U" {
		t.Fatalf("unexpected application: %#v", app)
	}
}

func TestParseDesktopEntrySkipsHiddenApplications(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hidden.desktop")
	content := "[Desktop Entry]\nName=Hidden\nExec=hidden\nHidden=true\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	app, err := parseDesktopEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if app != (application{}) {
		t.Fatalf("expected hidden app to be skipped, got %#v", app)
	}
}

func TestExpandExec(t *testing.T) {
	args, err := expandExec(`program --title "Hello world" %U`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"program", "--title", "Hello world"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("got %#v, want %#v", args, want)
	}
}

func TestExpandExecPreservesEscapedPercent(t *testing.T) {
	args, err := expandExec(`program --percent=%%`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"program", "--percent=%"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("got %#v, want %#v", args, want)
	}
}

func TestFilterApplications(t *testing.T) {
	applications := []application{
		{Name: "Calculator"},
		{Name: "Text Editor"},
		{Name: "Steam"},
	}

	filtered := filterApplications(applications, "  CALC ")
	if len(filtered) != 1 || filtered[0].Name != "Calculator" {
		t.Fatalf("unexpected filtered applications: %#v", filtered)
	}

	all := filterApplications(applications, "")
	if len(all) != len(applications) {
		t.Fatalf("empty query returned %d applications, want %d", len(all), len(applications))
	}
}
