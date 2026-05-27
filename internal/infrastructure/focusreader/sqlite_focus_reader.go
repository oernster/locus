package focusreader

import (
	"database/sql"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/oernster/locus/internal/application/dto"
	"github.com/oernster/locus/internal/application/service"
	"github.com/oernster/locus/internal/infrastructure/wininfo"
)

const (
	// idleThresholdSeconds is the gap between focus sessions that counts as idle
	// time (5 minutes).
	idleThresholdSeconds = int64(300)
	// maxAppsInReport is the maximum number of apps returned per stage.
	maxAppsInReport = 8
	// windowsSystemPathPrefix is filtered out of app results.
	windowsSystemPathPrefix = `C:\Windows\`
)

// SQLiteFocusReader reads the focus_sessions table in locus.db.
type SQLiteFocusReader struct {
	db *sql.DB
}

// NewSQLiteFocusReader creates a reader backed by the supplied locus DB.
func NewSQLiteFocusReader(db *sql.DB) *SQLiteFocusReader {
	return &SQLiteFocusReader{db: db}
}

// GetFocusDataForSessions implements service.FocusReader.
func (r *SQLiteFocusReader) GetFocusDataForSessions(sessions []service.FocusSessionWindow) dto.FocusDataDTO {
	if len(sessions) == 0 {
		return dto.FocusDataDTO{Available: true, Apps: []dto.AppFocusDTO{}}
	}

	type appKey = string
	appSeconds := make(map[appKey]int64)
	appSessions := make(map[appKey]int)
	var totalSeconds, idleSeconds int64

	now := time.Now().Unix()

	for _, win := range sessions {
		winDuration := win.EndedAt - win.StartedAt
		totalSeconds += winDuration

		// Query focus_sessions overlapping this locus session window.
		// Timestamps are Unix seconds.
		rows, err := r.db.Query(
			`SELECT exe_path, started_at, COALESCE(ended_at, ?) AS ended_at
			 FROM focus_sessions
			 WHERE started_at < ? AND (ended_at IS NULL OR ended_at > ?)
			 ORDER BY started_at`,
			now, win.EndedAt, win.StartedAt)
		if err != nil {
			log.Printf("focus reader: query error: %v", err)
			continue
		}

		type fSession struct {
			exePath string
			started int64
			ended   int64
		}
		var fSessions []fSession
		for rows.Next() {
			var exePath string
			var started, ended int64
			if err := rows.Scan(&exePath, &started, &ended); err != nil {
				continue
			}
			// Clamp to locus session window.
			if started < win.StartedAt {
				started = win.StartedAt
			}
			if ended > win.EndedAt {
				ended = win.EndedAt
			}
			fSessions = append(fSessions, fSession{exePath, started, ended})
		}
		rows.Close()

		// Sort by start time.
		sort.Slice(fSessions, func(i, j int) bool {
			return fSessions[i].started < fSessions[j].started
		})

		// Aggregate per-app durations and detect idle gaps.
		prevEnd := win.StartedAt
		for _, fs := range fSessions {
			// Idle gap before this session.
			gapSec := fs.started - prevEnd
			if gapSec > idleThresholdSeconds {
				idleSeconds += gapSec
			}

			// Skip system processes.
			if strings.HasPrefix(fs.exePath, windowsSystemPathPrefix) {
				prevEnd = fs.ended
				continue
			}

			durSec := fs.ended - fs.started
			if durSec < 0 {
				durSec = 0
			}
			appSeconds[fs.exePath] += durSec
			appSessions[fs.exePath]++
			prevEnd = fs.ended
		}

		// Idle tail gap.
		tailGap := win.EndedAt - prevEnd
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
