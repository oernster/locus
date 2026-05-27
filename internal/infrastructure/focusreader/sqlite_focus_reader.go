package focusreader

import (
	"database/sql"
	"log"
	"sort"
	"strings"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/application/service"
	"github.com/oernster/locus/internal/infrastructure/wininfo"

	_ "modernc.org/sqlite"
)

const (
	// idleThresholdSeconds is the gap between focus-reader sessions that counts
	// as idle time (5 minutes).
	idleThresholdSeconds = int64(300)
	// maxAppsInReport is the maximum number of apps returned per stage.
	maxAppsInReport = 8
	// msPerSecond converts focus-reader millisecond timestamps to seconds.
	msPerSecond = int64(1000)
	// windowsSystemPathPrefix is filtered out of app results.
	windowsSystemPathPrefix = `C:\Windows\`
)

// SQLiteFocusReader reads the focus-reader SQLite database read-only.
type SQLiteFocusReader struct {
	dbPath string
}

// NewSQLiteFocusReader creates a reader for the given focus-reader DB path.
func NewSQLiteFocusReader(dbPath string) *SQLiteFocusReader {
	return &SQLiteFocusReader{dbPath: dbPath}
}

// GetFocusDataForSessions implements service.FocusReader.
func (r *SQLiteFocusReader) GetFocusDataForSessions(sessions []service.FocusSessionWindow) dto.FocusDataDTO {
	if len(sessions) == 0 {
		return dto.FocusDataDTO{Available: true, Apps: []dto.AppFocusDTO{}}
	}

	db, err := sql.Open("sqlite", "file:"+r.dbPath+"?mode=ro")
	if err != nil {
		log.Printf("focus-reader DB open error: %v", err)
		return dto.FocusDataDTO{Available: false}
	}
	defer db.Close()

	// Verify connectivity.
	if err := db.Ping(); err != nil {
		return dto.FocusDataDTO{Available: false}
	}

	type appKey = string
	appSeconds := make(map[appKey]int64)
	appSessions := make(map[appKey]int)
	var totalSeconds, idleSeconds int64

	for _, win := range sessions {
		winStartMS := win.StartedAt * msPerSecond
		winEndMS := win.EndedAt * msPerSecond
		winDuration := win.EndedAt - win.StartedAt
		totalSeconds += winDuration

		// Query focus-reader sessions overlapping this locus session window.
		rows, err := db.Query(
			`SELECT exe_path, started_at, COALESCE(ended_at, ?) AS ended_at
			 FROM sessions
			 WHERE started_at < ? AND (ended_at IS NULL OR ended_at > ?)
			 ORDER BY started_at`,
			winEndMS, winEndMS, winStartMS)
		if err != nil {
			log.Printf("focus-reader query error: %v", err)
			continue
		}

		type frSession struct {
			exePath   string
			startedMS int64
			endedMS   int64
		}
		var frSessions []frSession
		for rows.Next() {
			var exePath string
			var startedMS, endedMS int64
			if err := rows.Scan(&exePath, &startedMS, &endedMS); err != nil {
				continue
			}
			// Clamp to locus session window.
			if startedMS < winStartMS {
				startedMS = winStartMS
			}
			if endedMS > winEndMS {
				endedMS = winEndMS
			}
			frSessions = append(frSessions, frSession{exePath, startedMS, endedMS})
		}
		rows.Close()

		// Sort by start time.
		sort.Slice(frSessions, func(i, j int) bool {
			return frSessions[i].startedMS < frSessions[j].startedMS
		})

		// Aggregate per-app durations and detect idle gaps.
		prevEnd := winStartMS
		for _, fs := range frSessions {
			// Idle gap before this session.
			gapSec := (fs.startedMS - prevEnd) / msPerSecond
			if gapSec > idleThresholdSeconds {
				idleSeconds += gapSec
			}

			// Skip system processes.
			if strings.HasPrefix(fs.exePath, windowsSystemPathPrefix) {
				prevEnd = fs.endedMS
				continue
			}

			durSec := (fs.endedMS - fs.startedMS) / msPerSecond
			if durSec < 0 {
				durSec = 0
			}
			appSeconds[fs.exePath] += durSec
			appSessions[fs.exePath]++
			prevEnd = fs.endedMS
		}

		// Idle tail gap.
		tailGap := (winEndMS - prevEnd) / msPerSecond
		if tailGap > idleThresholdSeconds {
			idleSeconds += tailGap
		}
	}

	// Build app list sorted by descending total time.
	type appEntry struct {
		exePath string
		seconds int64
		count   int
	}
	var appList []appEntry
	for path, secs := range appSeconds {
		appList = append(appList, appEntry{path, secs, appSessions[path]})
	}
	sort.Slice(appList, func(i, j int) bool {
		return appList[i].seconds > appList[j].seconds
	})
	if len(appList) > maxAppsInReport {
		appList = appList[:maxAppsInReport]
	}

	apps := make([]dto.AppFocusDTO, 0, len(appList))
	for _, a := range appList {
		info := wininfo.GetAppInfo(a.exePath)
		apps = append(apps, dto.AppFocusDTO{
			ExePath:      a.exePath,
			FriendlyName: info.FriendlyName,
			TotalSeconds: a.seconds,
			SessionCount: a.count,
		})
	}

	deepWork := totalSeconds - idleSeconds
	if deepWork < 0 {
		deepWork = 0
	}

	return dto.FocusDataDTO{
		Available:       true,
		TotalSeconds:    totalSeconds,
		IdleSeconds:     idleSeconds,
		DeepWorkSeconds: deepWork,
		Apps:            apps,
	}
}
