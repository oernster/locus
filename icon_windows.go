//go:build windows

package main

import (
	"log"
	"syscall"

	"github.com/lxn/win"
)

// wailsIconResourceID is the RT_GROUP_ICON resource ID that Wails (via go-winres)
// embeds in the Windows executable. Verified by extracting resources from the
// built exe: only RT_GROUP_ICON #3 is present.
const wailsIconResourceID = 3

// Window class index constants for SetClassLongPtr (not in lxn/win).
const (
	gclpHIcon   = -14
	gclpHIconSm = -34
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procSetClassLongPtr = user32.NewProc("SetClassLongPtrW")
)

func setClassLongPtr(hwnd win.HWND, index int32, value uintptr) uintptr {
	r, _, _ := procSetClassLongPtr.Call(uintptr(hwnd), uintptr(index), value)
	return r
}

// setTaskbarIcon loads the embedded icon and applies it to both the window
// message slots (WM_SETICON) and the window class (SetClassLongPtr). The
// taskbar button reads the window class icon; WM_SETICON alone does not
// update it reliably.
func setTaskbarIcon() {
	hmod := win.GetModuleHandle(nil)
	idPtr := win.MAKEINTRESOURCE(wailsIconResourceID)

	// Load big icon (32x32 at 96 DPI) for all slots.
	hicoBig := win.LoadImage(
		hmod,
		idPtr,
		win.IMAGE_ICON,
		win.GetSystemMetrics(win.SM_CXICON),
		win.GetSystemMetrics(win.SM_CYICON),
		win.LR_DEFAULTCOLOR,
	)
	if hicoBig == 0 {
		log.Printf("setTaskbarIcon: LoadImage failed for resource ID %d", wailsIconResourceID)
		return
	}

	// Find the Wails host window by its title.
	hwnd := win.FindWindow(nil, syscall.StringToUTF16Ptr("Locus"))
	if hwnd == 0 {
		log.Printf("setTaskbarIcon: window 'Locus' not found")
		return
	}

	// WM_SETICON: title bar (ICON_SMALL=0) and alt-tab (ICON_BIG=1).
	win.SendMessage(hwnd, win.WM_SETICON, 0, uintptr(hicoBig))
	win.SendMessage(hwnd, win.WM_SETICON, 1, uintptr(hicoBig))

	// SetClassLongPtr: taskbar button reads the window class icon, not WM_SETICON.
	setClassLongPtr(hwnd, gclpHIcon, uintptr(hicoBig))
	setClassLongPtr(hwnd, gclpHIconSm, uintptr(hicoBig))
}
