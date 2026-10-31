package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type application struct {
	Name string
	Exec string
	Path string
}

func applicationDirectories(home string) []string {
	return []string{
		"/usr/share/applications",
		filepath.Join(home, ".local", "share", "applications"),
		filepath.Join(home, "Desktop"),
	}
}

func discoverApplications(home string) ([]application, error) {
	applications := make(map[string]application)
	var errs []error

	for _, directory := range applicationDirectories(home) {
		entries, err := os.ReadDir(directory)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			errs = append(errs, fmt.Errorf("read %s: %w", directory, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.EqualFold(filepath.Ext(entry.Name()), ".desktop") {
				continue
			}

			path := filepath.Join(directory, entry.Name())
			app, err := parseDesktopEntry(path)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if app.Exec == "" {
				continue
			}
			applications[app.Name] = app
		}
	}

	result := make([]application, 0, len(applications))
	for _, app := range applications {
		result = append(result, app)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, errors.Join(errs...)
}

func filterApplications(applications []application, query string) []application {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return applications
	}

	filtered := make([]application, 0, len(applications))
	for _, app := range applications {
		if strings.Contains(strings.ToLower(app.Name), query) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

func parseDesktopEntry(path string) (application, error) {
	file, err := os.Open(path)
	if err != nil {
		return application{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	var (
		inDesktopEntry bool
		name           string
		exec           string
		hidden         bool
		noDisplay      bool
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inDesktopEntry = line == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntry {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch key {
		case "Name":
			name = value
		case "Exec":
			exec = value
		case "Hidden":
			hidden = value == "true"
		case "NoDisplay":
			noDisplay = value == "true"
		}
	}
	if err := scanner.Err(); err != nil {
		return application{}, fmt.Errorf("read %s: %w", path, err)
	}
	if hidden || noDisplay || name == "" || exec == "" {
		return application{}, nil
	}

	return application{
		Name: name,
		Exec: exec,
		Path: path,
	}, nil
}

func expandExec(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quoted bool
	var escaped bool
	var argumentStarted bool

	flush := func() {
		if argumentStarted {
			args = append(args, current.String())
			current.Reset()
			argumentStarted = false
		}
	}

	chars := []rune(command)
	for index := 0; index < len(chars); index++ {
		char := chars[index]
		if escaped {
			current.WriteRune(char)
			escaped = false
			argumentStarted = true
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '"' {
			quoted = !quoted
			argumentStarted = true
			continue
		}
		if !quoted && unicode.IsSpace(char) {
			flush()
			continue
		}
		if char == '%' {
			if index+1 == len(chars) {
				return nil, errors.New("invalid trailing field code")
			}
			index++
			if chars[index] == '%' {
				current.WriteRune('%')
				argumentStarted = true
			}
			continue
		}
		current.WriteRune(char)
		argumentStarted = true
	}
	if escaped {
		return nil, errors.New("invalid trailing escape")
	}
	if quoted {
		return nil, errors.New("unterminated quote")
	}
	flush()
	if len(args) == 0 {
		return nil, errors.New("empty command")
	}
	return args, nil
}
