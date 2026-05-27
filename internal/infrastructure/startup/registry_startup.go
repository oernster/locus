//go:build windows

package startup

import (
	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	appValueName = "locus"
)

// Install registers executablePath to run at user login via the HKCU Run key.
func Install(executablePath string) error {
	return InstallWithName(appValueName, executablePath)
}

// InstallWithName registers executablePath under a custom value name.
func InstallWithName(name, executablePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(name, executablePath)
}

// Uninstall removes the locus Run key entry.
func Uninstall() error {
	return UninstallWithName(appValueName)
}

// UninstallWithName removes a named Run key entry.
func UninstallWithName(name string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	err = k.DeleteValue(name)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

// IsInstalled reports whether the locus Run key entry exists.
func IsInstalled() (bool, error) {
	return IsInstalledWithName(appValueName)
}

// IsInstalledWithName checks for a named entry.
func IsInstalledWithName(name string) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, nil
	}
	defer k.Close()

	_, _, err = k.GetStringValue(name)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
