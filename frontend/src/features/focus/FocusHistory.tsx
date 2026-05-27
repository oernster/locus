import { useEffect, useState } from "react";
import { GetFocusDataForTimeRange } from "../../../wailsjs/go/main/App";
import type { FocusDataDTO } from "../../types/locus";
import styles from "./FocusHistory.module.css";

type Period = "today" | "yesterday" | "week" | "month";

const PERIOD_LABELS: Record<Period, string> = {
  today: "Today",
  yesterday: "Yesterday",
  week: "This Week",
  month: "This Month",
};

const PERIODS: Period[] = ["today", "yesterday", "week", "month"];

// pollIntervalMs is how often today's data refreshes (live tracking).
const pollIntervalMs = 2_000;

function getPeriodRange(period: Period): [number, number] {
  const nowMs = Date.now();
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  switch (period) {
    case "today":
      return [Math.floor(today.getTime() / 1000), Math.floor(nowMs / 1000)];

    case "yesterday": {
      const yStart = new Date(today);
      yStart.setDate(yStart.getDate() - 1);
      return [Math.floor(yStart.getTime() / 1000), Math.floor(today.getTime() / 1000)];
    }

    case "week": {
      const weekStart = new Date(today);
      const dow = weekStart.getDay();
      const diff = dow === 0 ? -6 : 1 - dow; // ISO week: Monday start
      weekStart.setDate(weekStart.getDate() + diff);
      return [Math.floor(weekStart.getTime() / 1000), Math.floor(nowMs / 1000)];
    }

    case "month": {
      const monthStart = new Date(today.getFullYear(), today.getMonth(), 1);
      return [Math.floor(monthStart.getTime() / 1000), Math.floor(nowMs / 1000)];
    }
  }
}

function formatDuration(seconds: number): string {
  const totalMin = Math.floor(seconds / 60);
  const h = Math.floor(totalMin / 60);
  const m = totalMin % 60;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function FocusHistory() {
  const [open, setOpen] = useState(false);
  const [period, setPeriod] = useState<Period>("today");
  const [data, setData] = useState<FocusDataDTO | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!open) return;

    let cancelled = false;

    function fetchData(): void {
      const [start, end] = getPeriodRange(period);
      GetFocusDataForTimeRange(start, end)
        .then((result) => { if (!cancelled) setData(result); })
        .catch(() => { if (!cancelled) setData(null); })
        .finally(() => { if (!cancelled) setLoading(false); });
    }

    setLoading(true);
    fetchData();

    const timer = period === "today"
      ? window.setInterval(fetchData, pollIntervalMs)
      : null;

    return () => {
      cancelled = true;
      if (timer !== null) window.clearInterval(timer);
    };
  }, [open, period]);

  const maxAppTime = data?.apps?.reduce((m, a) => Math.max(m, a.total_seconds), 1) ?? 1;
  const apps = data?.apps ?? [];

  return (
    <div className={styles.root}>
      <button
        type="button"
        className={styles.toggleRow}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <span className={styles.title}>Focus History</span>
        <span className={`${styles.chevron} ${open ? styles.chevronOpen : ""}`} aria-hidden="true">
          ▲
        </span>
      </button>

      {open ? (
        <div className={styles.body}>
          <div className={styles.periodPicker} role="group" aria-label="Time period">
            {PERIODS.map((p) => (
              <button
                key={p}
                type="button"
                className={`${styles.periodBtn} ${period === p ? styles.periodBtnActive : ""}`}
                onClick={() => setPeriod(p)}
                aria-pressed={period === p}
              >
                {PERIOD_LABELS[p]}
              </button>
            ))}
          </div>

          <div className={styles.content}>
            {loading ? (
              <span className={styles.hint}>Loading...</span>
            ) : apps.length === 0 ? (
              <span className={styles.hint}>No focus data for this period</span>
            ) : (
              <div className={styles.appGrid}>
                {apps.map((app) => {
                  const pct = Math.max(4, Math.round((app.total_seconds / maxAppTime) * 100));
                  return (
                    <div key={app.exe_path} className={styles.appRow}>
                      <div className={styles.appName} title={app.exe_path}>
                        {app.friendly_name}
                      </div>
                      <div className={styles.barTrack}>
                        <div
                          className={styles.barFill}
                          style={{ width: `${pct}%` }}
                          aria-hidden="true"
                        />
                      </div>
                      <div className={styles.appTime}>{formatDuration(app.total_seconds)}</div>
                    </div>
                  );
                })}
              </div>
            )}

            {data && data.deep_work_seconds > 0 && period === "today" ? (
              <div className={styles.deepWorkLine}>
                <span className={styles.deepWorkValue}>{formatDuration(data.deep_work_seconds)} deep work</span>
                {data.idle_seconds > 0 ? (
                  <span className={styles.idleNote}> ({formatDuration(data.idle_seconds)} idle subtracted)</span>
                ) : null}
              </div>
            ) : null}

            {data && data.total_seconds > 0 ? (
              <div className={styles.totalLine}>Total: {formatDuration(data.total_seconds)}</div>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}
