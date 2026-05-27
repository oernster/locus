//go:build windows

package tray

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"runtime"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// ShowWindowFunc is a callback to show the main Wails window.
type ShowWindowFunc func()

// QuitFunc is a callback to quit the Wails application.
type QuitFunc func()

// Run starts the walk tray in the current goroutine (which must be OS-thread-
// locked). It blocks until the walk message loop exits.
//
// onShow is called when the user clicks Open.
// onQuit is called when the user clicks Exit.
// ready is closed once the tray is set up so the caller can proceed.
// iconPNG is the raw PNG bytes of the tray icon (embedded from main).
func Run(onShow ShowWindowFunc, onQuit QuitFunc, ready chan<- struct{}, iconPNG []byte) {
	// This goroutine must be pinned to its OS thread for walk to work correctly.
	runtime.LockOSThread()

	mw, err := walk.NewMainWindow()
	if err != nil {
		log.Printf("tray: NewMainWindow: %v", err)
		close(ready)
		return
	}

	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		log.Printf("tray: NewNotifyIcon: %v", err)
		close(ready)
		return
	}
	defer ni.Dispose()

	// Load tray icon from embedded PNG bytes.
	if len(iconPNG) > 0 {
		img, _, decErr := image.Decode(bytes.NewReader(iconPNG))
		if decErr == nil {
			dpi := int(win.GetDpiForWindow(mw.Handle()))
			if dpi == 0 {
				dpi = 96
			}
			ico, icoErr := walk.NewIconFromImageForDPI(img, dpi)
			if icoErr != nil {
				ico, icoErr = walk.NewIconFromImage(img)
			}
			if icoErr == nil {
				_ = ni.SetIcon(ico)
			} else {
				log.Printf("tray: set icon: %v", icoErr)
			}
		} else {
			log.Printf("tray: decode icon PNG: %v", decErr)
		}
	}

	if err := ni.SetToolTip("Locus: task board + focus insights"); err != nil {
		log.Printf("tray: SetToolTip: %v", err)
	}

	// Left-click opens the main window.
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			onShow()
		}
	})

	// Context menu: Open / separator / Exit.
	openAction := walk.NewAction()
	if err := openAction.SetText("Open"); err == nil {
		openAction.Triggered().Attach(func() { onShow() })
		_ = ni.ContextMenu().Actions().Add(openAction)
	}

	_ = ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	if err := exitAction.SetText("Exit"); err == nil {
		exitAction.Triggered().Attach(func() {
			ni.Dispose()
			onQuit()
		})
		_ = ni.ContextMenu().Actions().Add(exitAction)
	}

	if err := ni.SetVisible(true); err != nil {
		log.Printf("tray: SetVisible: %v", err)
	}

	// Signal that the tray is ready.
	close(ready)

	// Hide the main window (it's a message-only host for the tray icon).
	mw.Hide()

	// Block until the walk message loop exits.
	mw.Run()
}
