package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

	styles := tview.Styles
	styles.PrimitiveBackgroundColor = tcell.ColorDefault
	styles.ContrastBackgroundColor = tcell.ColorDefault
	styles.MoreContrastBackgroundColor = tcell.ColorDefault
	tview.Styles = styles

	list := tview.NewList().
		ShowSecondaryText(false).
		SetWrapAround(true).
		SetSelectedFocusOnly(false).
		SetSelectedStyle(tcell.StyleDefault.Reverse(true))
	list.SetBorder(true).SetTitle(" launchy ")
	filter := tview.NewInputField().SetLabel("Filter: ").SetPlaceholder("Type to filter applications")
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true).
		AddItem(filter, 1, 0, false)

	var ui *tview.Application
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
	ui = tview.NewApplication().SetRoot(layout, true).EnableMouse(true)
	ui.SetFocus(filter)
	list.SetDoneFunc(func() {
		ui.Stop()
	})
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
