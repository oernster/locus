import { useEffect, useState } from "react";
import { GetFocusData } from "../../../wailsjs/go/main/App";
import type { FocusDataDTO, StageId } from "../../types/locus";
import { BOTTOM_PANEL_LABELS } from "../commands/constants";
import styles from "./FocusPanel.module.css";

export type FocusPanelProps = {
  stageId: StageId;
  refreshNonce?: number;
};

function formatDurationHM(seconds: number): string {
  const totalMin = Math.floor(seconds / 60);
  const h = Math.floor(totalMin / 60);
  const m = totalMin % 60;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

// pollIntervalMs is how often the focus panel refreshes its data automatically.
const pollIntervalMs = 2_000;

export function FocusPanel({ stageId, refreshNonce }: FocusPanelProps) {
  const [data, setData] = useState<FocusDataDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const headerLabel = BOTTOM_PANEL_LABELS[stageId];

  useEffect(() => {
    let cancelled = false;

    function fetchData(): void {
      GetFocusData(stageId)
        .then((result) => { if (!cancelled) setData(result); })
        .catch(() => {
          if (!cancelled) setData({ available: false, stage_id: stageId, total_seconds: 0, idle_seconds: 0, deep_work_seconds: 0, apps: [] });
        })
        .finally(() => { if (!cancelled) setLoading(false); });
    }

    setLoading(true);
    fetchData();
    const timer = window.setInterval(fetchData, pollIntervalMs);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [stageId, refreshNonce]);

  const maxAppTime = data?.apps.reduce((m, a) => Math.max(m, a.total_seconds), 1) ?? 1;

  return (
    <div className={styles.panel}>
      <div className={styles.panelHeader}>{headerLabel}</div>

      {loading ? (
        <div className={styles.hint}>Loading...</div>
      ) : !data || !data.available ? (
        <div className={styles.hint}>Focus tracking unavailable</div>
      ) : (
        <>
          {stageId === "EXECUTE" && data.deep_work_seconds > 0 ? (
            <div className={styles.deepWorkLine}>
              <span className={styles.deepWorkValue}>{formatDurationHM(data.deep_work_seconds)} deep work</span>
              {data.idle_seconds > 0 ? (
                <span className={styles.idleNote}> ({formatDurationHM(data.idle_seconds)} idle subtracted)</span>
              ) : null}
            </div>
          ) : null}

          {stageId === "DONE" ? (
            <div className={styles.retroLine}>
              <span>Total time: {formatDurationHM(data.total_seconds)}</span>
            </div>
          ) : null}

          {data.apps.length === 0 ? (
            <div className={styles.hint}>No focus data recorded yet</div>
          ) : (
            <div className={styles.appList}>
              {data.apps.map((app) => {
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
                    <div className={styles.appTime}>{formatDurationHM(app.total_seconds)}</div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}
    </div>
  );
}
