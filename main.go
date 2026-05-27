package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/oernster/locus/internal/application/service"
	"github.com/oernster/locus/internal/infrastructure/focusreader"
	"github.com/oernster/locus/internal/infrastructure/persistence"
	"github.com/oernster/locus/internal/infrastructure/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Resolve DB path: %APPDATA%\locus\locus.db
	appData := os.Getenv("APPDATA")
	if appData == "" {
		log.Fatal("APPDATA environment variable not set")
	}
	dbDir := filepath.Join(appData, "locus")
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		log.Fatalf("create DB dir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "locus.db")

	// Open locus SQLite DB.
	db, err := persistence.Open(dbPath)
	if err != nil {
		log.Fatalf("open DB: %v", err)
	}
	defer db.Close()

	// Wire repositories.
	commandRepo := persistence.NewSQLiteCommandRepository(db)
	sessionRepo := persistence.NewSQLiteSessionRepository(db)
	outcomeRepo := persistence.NewSQLiteOutcomeRepository(db)
	boardRepo := persistence.NewSQLiteBoardRepository(db)
	snapshotRepo := persistence.NewSQLiteSnapshotRepository(db)

	// Wire services.
	commandSvc := service.NewCommandService(commandRepo)
	sessionSvc := service.NewSessionService(sessionRepo, commandRepo)
	outcomeSvc := service.NewOutcomeService(outcomeRepo)
	boardSvc := service.NewBoardService(boardRepo, commandRepo)
	snapshotSvc := service.NewSnapshotService(snapshotRepo, commandRepo, outcomeRepo, boardRepo)

	// Wire focus-reader integration.
	focusReaderDBPath := filepath.Join(appData, "focus-reader", "sessions.db")
	focusReader := focusreader.NewSQLiteFocusReader(focusReaderDBPath)
	focusSvc := service.NewFocusService(sessionRepo, focusReader)

	// Create the Wails App.
	app := NewApp(commandSvc, sessionSvc, outcomeSvc, boardSvc, snapshotSvc, focusSvc)

	// Start walk tray in a background goroutine.
	ready := make(chan struct{})
	go tray.Run(
		func() {
			if app.ctx != nil {
				runtime.WindowShow(app.ctx)
			}
		},
		func() {
			if app.ctx != nil {
				runtime.Quit(app.ctx)
			}
		},
		ready,
	)
	// Wait for tray to initialise before Wails takes over the main goroutine.
	<-ready

	// Run Wails on the main goroutine (blocks until quit).
	if err := wails.Run(&options.App{
		Title:  "Locus",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		log.Fatalf("wails.Run: %v", err)
	}
}
