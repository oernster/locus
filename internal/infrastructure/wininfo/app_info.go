//go:build windows

package wininfo

import (
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// AppInfo holds resolved metadata for an executable.
type AppInfo struct {
	FriendlyName string
	ExePath      string
}

var (
	cache   = make(map[string]AppInfo)
	cacheMu sync.RWMutex
)

// GetAppInfo resolves a friendly display name for an executable path.
// It tries FileDescription, then ProductName, then the filename stem.
// Results are cached in-memory for the lifetime of the process.
func GetAppInfo(exePath string) AppInfo {
	cacheMu.RLock()
	if info, ok := cache[exePath]; ok {
		cacheMu.RUnlock()
		return info
	}
	cacheMu.RUnlock()

	info := AppInfo{ExePath: exePath, FriendlyName: friendlyFromPath(exePath)}
	if resolved := resolveVersionInfo(exePath); resolved != "" {
		info.FriendlyName = resolved
	}

	cacheMu.Lock()
	cache[exePath] = info
	cacheMu.Unlock()
	return info
}

// friendlyFromPath extracts the filename without extension as a last resort.
func friendlyFromPath(exePath string) string {
	base := filepath.Base(exePath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base
}

// resolveVersionInfo reads PE version info and returns FileDescription or
// ProductName if available. Returns empty string on any failure.
func resolveVersionInfo(exePath string) string {
	exePtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return ""
	}

	size, err := getFileVersionInfoSize(exePtr)
	if err != nil || size == 0 {
		return ""
	}

	buf := make([]byte, size)
	if err := getFileVersionInfo(exePtr, size, buf); err != nil {
		return ""
	}

	// Try English (0x0409) and Unicode (0x04B0) code page first, then neutral.
	langCPs := []string{"040904B0", "040904E4", "04090000"}
	for _, langCP := range langCPs {
		for _, field := range []string{"FileDescription", "ProductName"} {
			sub := `\StringFileInfo\` + langCP + `\` + field
			subPtr, err := windows.UTF16PtrFromString(sub)
			if err != nil {
				continue
			}
			val := queryStringValue(buf, subPtr)
			if val != "" {
				return strings.TrimSpace(val)
			}
		}
	}
	return ""
}

// getFileVersionInfoSize wraps the Win32 GetFileVersionInfoSizeW call.
func getFileVersionInfoSize(filename *uint16) (uint32, error) {
	modVersion := windows.NewLazySystemDLL("version.dll")
	procSize := modVersion.NewProc("GetFileVersionInfoSizeW")
	var handle uint32
	r1, _, err := procSize.Call(
		uintptr(unsafe.Pointer(filename)),
		uintptr(unsafe.Pointer(&handle)),
	)
	if r1 == 0 {
		return 0, err
	}
	return uint32(r1), nil
}

// getFileVersionInfo fills buf with the version resource.
func getFileVersionInfo(filename *uint16, size uint32, buf []byte) error {
	modVersion := windows.NewLazySystemDLL("version.dll")
	procInfo := modVersion.NewProc("GetFileVersionInfoW")
	r1, _, err := procInfo.Call(
		uintptr(unsafe.Pointer(filename)),
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

// queryStringValue calls VerQueryValueW for a string sub-block.
func queryStringValue(buf []byte, sub *uint16) string {
	modVersion := windows.NewLazySystemDLL("version.dll")
	procQuery := modVersion.NewProc("VerQueryValueW")
	var pVal uintptr
	var vLen uint32
	r1, _, _ := procQuery.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(sub)),
		uintptr(unsafe.Pointer(&pVal)),
		uintptr(unsafe.Pointer(&vLen)),
	)
	if r1 == 0 || vLen == 0 || pVal == 0 {
		return ""
	}
	// pVal is a pointer into buf (same allocation). Convert via uintptr arithmetic.
	// We calculate the offset from buf base and slice accordingly.
	bufBase := uintptr(unsafe.Pointer(&buf[0]))
	if pVal < bufBase || pVal >= bufBase+uintptr(len(buf)) {
		return ""
	}
	offset := (pVal - bufBase) / 2 // convert byte offset to uint16 offset
	buf16 := (*[1 << 20]uint16)(unsafe.Pointer(&buf[0]))
	end := offset + uintptr(vLen)
	if end > uintptr(len(buf)/2) {
		end = uintptr(len(buf) / 2)
	}
	raw := buf16[offset:end]
	// Remove trailing null.
	for len(raw) > 0 && raw[len(raw)-1] == 0 {
		raw = raw[:len(raw)-1]
	}
	return windows.UTF16ToString(raw)
}
