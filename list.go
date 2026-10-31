package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func normalizeTabClick(action tview.MouseAction) tview.MouseAction {
	if action == tview.MouseLeftDoubleClick {
		return tview.MouseLeftClick
	}
	return action
}

func runList() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determine home directory: %w", err)
	}
	applications, discoveryErr := discoverApplications(home)
	if len(applications) == 0 {
		if discoveryErr != nil {
			return discoveryErr
		}
		return errors.New("no applications found")
	}
	configFile, err := settingsPath()
	if err != nil {
		return err
	}
	settings, settingsErr := loadSettings(configFile)
	if settingsErr != nil {
		settings = defaultSettings()
	}
	paletteColors, err := resolvePalette(settings.Palette)
	if err != nil {
		return fmt.Errorf("resolve default palette: %w", err)
	}

	ui := tview.NewApplication().EnableMouse(true)
	var (
		root             *tview.Flex
		pages            *tview.Pages
		list             *tview.List
		filter           *tview.InputField
		settingsView     *tview.TextView
		editorButton     *tview.Button
		searchTab        *tview.Button
		settingsTab      *tview.Button
		activeTab        = "search"
		settingsMessage  string
		activeTabStyle   tcell.Style
		inactiveTabStyle tcell.Style
	)
	if settingsErr != nil {
		settingsMessage = settingsErr.Error()
	}

	updateSettingsView := func() {
		editorName := strings.TrimSpace(os.Getenv("EDITOR"))
		if editorName == "" {
			editorName = "vi"
		}
		data, err := os.ReadFile(configFile)
		message := settingsMessage
		if errors.Is(err, os.ErrNotExist) {
			data, _ = json.MarshalIndent(settings, "", "  ")
			if message != "" {
				message += "\n"
			}
			message += "Settings file will be created when you edit it."
		} else if err != nil {
			if message != "" {
				message += "\n"
			}
			message += fmt.Sprintf("Unable to read settings file: %v", err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "Config file: %s\n", configFile)
		if message != "" {
			fmt.Fprintf(&content, "\n%s\n", message)
		}
		fmt.Fprintf(&content, "\n%s\n\nEnter to edit with %s. F1: Search, F2: Settings, Esc: Search.",
			strings.TrimSpace(string(data)), editorName)
		settingsView.SetText(content.String())
	}

	var buildUI func()
	var showSearch func()
	var showSettings func()
	var openEditor func()

	buildUI = func() {
		styles := tview.Styles
		styles.PrimitiveBackgroundColor = paletteColors.background
		styles.ContrastBackgroundColor = paletteColors.background
		styles.MoreContrastBackgroundColor = paletteColors.background
		styles.PrimaryTextColor = paletteColors.foreground
		styles.SecondaryTextColor = paletteColors.foreground
		styles.TertiaryTextColor = paletteColors.foreground
		styles.BorderColor = paletteColors.border
		styles.TitleColor = paletteColors.title
		styles.GraphicsColor = paletteColors.foreground
		styles.InverseTextColor = paletteColors.selectedForeground
		styles.ContrastSecondaryTextColor = paletteColors.selectedForeground
		tview.Styles = styles

		list = tview.NewList().
			ShowSecondaryText(false).
			SetWrapAround(true).
			SetSelectedFocusOnly(false).
			SetSelectedStyle(tcell.StyleDefault.
				Foreground(paletteColors.selectedForeground).
				Background(paletteColors.selectedBackground))
		list.SetBorder(true).SetTitle(" launchy ")
		filter = tview.NewInputField().SetLabel("Filter: ").SetPlaceholder("Type to filter applications")
		searchPage := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(list, 0, 1, true).
			AddItem(filter, 1, 0, false)

		var filtered []application
		launchApplication := func(app application) {
			if err := launch(app); err != nil {
				list.SetTitle(fmt.Sprintf(" launchy — %v ", err))
				ui.SetFocus(filter)
				return
			}
			ui.Stop()
		}
		updateList := func(query string) {
			filtered = filterApplications(applications, query)
			list.Clear()
			for _, app := range filtered {
				list.AddItem(app.Name, "", 0, func() {
					launchApplication(app)
				})
			}
			if len(filtered) == 0 {
				list.AddItem("No matching applications", "", 0, nil)
				return
			}
			list.SetCurrentItem(0)
		}
		filter.SetChangedFunc(updateList)
		updateList(filter.GetText())
		filter.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEnter:
				index := list.GetCurrentItem()
				if index >= 0 && index < len(filtered) {
					launchApplication(filtered[index])
				}
			case tcell.KeyEscape:
				ui.Stop()
			case tcell.KeyTab, tcell.KeyBacktab:
				ui.SetFocus(list)
			}
		})
		filter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyDown, tcell.KeyUp:
				list.InputHandler()(event, func(primitive tview.Primitive) {
					ui.SetFocus(primitive)
				})
				return nil
			}
			return event
		})
		list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyTab, tcell.KeyBacktab:
				ui.SetFocus(filter)
				return nil
			}
			return event
		})
		list.SetDoneFunc(func() {
			ui.Stop()
		})

		settingsView = tview.NewTextView().
			SetDynamicColors(false).
			SetScrollable(true).
			SetTextStyle(tcell.StyleDefault.
				Foreground(paletteColors.foreground).
				Background(paletteColors.background))
		settingsView.SetBorder(true).SetTitle(" Settings ")
		editorName := strings.TrimSpace(os.Getenv("EDITOR"))
		if editorName == "" {
			editorName = "vi"
		}
		editorButton = tview.NewButton("Edit with " + editorName)
		editorButton.SetSelectedFunc(func() {
			openEditor()
		})
		settingsPage := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(settingsView, 0, 1, true).
			AddItem(editorButton, 1, 0, true)
		updateSettingsView()
		settingsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEnter:
				openEditor()
				return nil
			case tcell.KeyEscape:
				showSearch()
				return nil
			case tcell.KeyTab, tcell.KeyBacktab:
				ui.SetFocus(editorButton)
				return nil
			}
			return event
		})

		searchTab = tview.NewButton("Search")
		settingsTab = tview.NewButton("Settings")
		selectTabOnDoubleClick := func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			return normalizeTabClick(action), event
		}
		searchTab.SetMouseCapture(selectTabOnDoubleClick)
		settingsTab.SetMouseCapture(selectTabOnDoubleClick)
		inactiveTabStyle = tcell.StyleDefault.
			Foreground(paletteColors.foreground).
			Background(paletteColors.background)
		activeTabStyle = tcell.StyleDefault.
			Foreground(paletteColors.selectedForeground).
			Background(paletteColors.selectedBackground)
		searchTab.SetStyle(inactiveTabStyle).SetActivatedStyle(activeTabStyle)
		settingsTab.SetStyle(inactiveTabStyle).SetActivatedStyle(activeTabStyle)
		if activeTab == "search" {
			searchTab.SetStyle(activeTabStyle)
		} else {
			settingsTab.SetStyle(activeTabStyle)
		}
		searchTab.SetSelectedFunc(func() {
			showSearch()
		})
		settingsTab.SetSelectedFunc(func() {
			showSettings()
		})
		tabBar := tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(searchTab, 12, 0, true).
			AddItem(settingsTab, 12, 0, true).
			AddItem(tview.NewBox(), 0, 1, false)
		pages = tview.NewPages().
			AddPage("search", searchPage, true, activeTab == "search").
			AddPage("settings", settingsPage, true, activeTab == "settings")
		root = tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(tabBar, 1, 0, true).
			AddItem(pages, 0, 1, true)
		ui.SetRoot(root, true)
		if activeTab == "settings" {
			ui.SetFocus(settingsView)
		} else {
			ui.SetFocus(filter)
		}
	}

	showSearch = func() {
		activeTab = "search"
		pages.SwitchToPage("search")
		ui.SetFocus(filter)
		searchTab.SetStyle(activeTabStyle)
		settingsTab.SetStyle(inactiveTabStyle)
	}
	showSettings = func() {
		activeTab = "settings"
		pages.SwitchToPage("settings")
		ui.SetFocus(settingsView)
		searchTab.SetStyle(inactiveTabStyle)
		settingsTab.SetStyle(activeTabStyle)
	}
	openEditor = func() {
		if err := ensureSettingsFile(configFile); err != nil {
			settingsMessage = err.Error()
			updateSettingsView()
			return
		}
		editorArgs, err := parseEditorCommand(os.Getenv("EDITOR"))
		if err != nil {
			settingsMessage = err.Error()
			updateSettingsView()
			return
		}

		command := exec.Command(editorArgs[0], append(editorArgs[1:], configFile)...)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		var editorErr error
		if !ui.Suspend(func() {
			editorErr = command.Run()
		}) {
			settingsMessage = "Unable to suspend the TUI to start the editor."
			updateSettingsView()
			return
		}

		updatedSettings, err := loadSettings(configFile)
		if err != nil {
			settingsMessage = err.Error()
			if editorErr != nil {
				settingsMessage += "\nEditor error: " + editorErr.Error()
			}
			updateSettingsView()
			return
		}
		updatedPalette, err := resolvePalette(updatedSettings.Palette)
		if err != nil {
			settingsMessage = err.Error()
			updateSettingsView()
			return
		}
		settings = updatedSettings
		paletteColors = updatedPalette
		settingsMessage = ""
		if editorErr != nil {
			settingsMessage = "Editor error: " + editorErr.Error()
		}
		activeTab = "settings"
		buildUI()
	}

	ui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			showSearch()
			return nil
		case tcell.KeyF2:
			showSettings()
			return nil
		}
		return event
	})
	buildUI()
	if err := ui.Run(); err != nil {
		return fmt.Errorf("run interface: %w", err)
	}
	return discoveryErr
}

func launch(app application) error {
	args, err := expandExec(app.Exec)
	if err != nil {
		return fmt.Errorf("parse %s: %w", app.Path, err)
	}

	command := exec.Command(args[0], args[1:]...)
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	command.Stdin = null
	command.Stdout = null
	command.Stderr = null
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := command.Start(); err != nil {
		null.Close()
		return fmt.Errorf("start %q: %w", app.Name, err)
	}
	return null.Close()
}
