.PHONY: generate build install uninstall dev clean test

# Generate Wails bindings (wailsjs/) without full build
generate:
	wails generate module

# Production build -> build/bin/locus.exe
build:
	wails build

# Build + deploy to %LOCALAPPDATA%\locus\ + set Run key + launch
install:
	powershell -ExecutionPolicy Bypass -File install.ps1

# Stop process, remove Run key, remove install dir
uninstall:
	powershell -ExecutionPolicy Bypass -File uninstall.ps1

# Dev server with hot-reload (Wails + Vite)
dev:
	wails dev

# Run Go unit/structural tests
test:
	go test ./...

# Remove build artifacts
clean:
	rm -rf build/bin frontend/dist
