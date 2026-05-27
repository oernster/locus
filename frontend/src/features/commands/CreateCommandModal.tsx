import { useEffect, useMemo, useRef, useState } from "react";
import { CreateCommand } from "../../../wailsjs/go/main/App";
import type { StageId, Status } from "../../types/locus";
import { DEFAULT_STAGE_LABELS, STAGES, STATUSES } from "./constants";
import styles from "./CreateCommandModal.module.css";

export type CreateCommandModalProps = {
  initialStageId: StageId;
  stageLabels: Record<StageId, string>;
  onClose: () => void;
  onCreated: () => Promise<void>;
  setError: (msg: string) => void;
};

export function CreateCommandModal(props: CreateCommandModalProps) {
  const { initialStageId, stageLabels, onClose, onCreated, setError } = props;

  const [title, setTitle] = useState("");
  const [stage_id, setStageId] = useState<StageId>(initialStageId);
  const [_status, setStatus] = useState<Status>("Not Started");
  const [saving, setSaving] = useState(false);
  const [inlineNotice, setInlineNotice] = useState<string | null>(null);
  const titleRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    titleRef.current?.focus();
  }, []);

  useEffect(() => {
    if (!inlineNotice) return;
    const t = window.setTimeout(() => setInlineNotice(null), 3000);
    return () => window.clearTimeout(t);
  }, [inlineNotice]);

  // Status is set to default "Not Started" but we expose the selector for future use.
  void setStatus;

  const canSubmit = useMemo(() => title.trim().length > 0 && !saving, [title, saving]);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    if (!canSubmit) return;

    setError("");
    setInlineNotice(null);
    setSaving(true);
    try {
      await CreateCommand(title.trim(), stage_id);
      await onCreated();
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Could not create command";
      setError(msg);
      if (msg.toLowerCase().includes("already exists") || msg.toLowerCase().includes("duplicate")) {
        setInlineNotice("Duplicate task title - choose a different name.");
      }
    } finally {
      setSaving(false);
    }
  }

  function onBackdropMouseDown(e: React.MouseEvent<HTMLDivElement>): void {
    if (e.target === e.currentTarget) onClose();
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLDivElement>): void {
    if (e.key === "Escape") onClose();
  }

  return (
    <div
      className={styles.backdrop}
      role="dialog"
      aria-modal="true"
      aria-label="Create command"
      onMouseDown={onBackdropMouseDown}
      onKeyDown={onKeyDown}
      tabIndex={-1}
    >
      <form className={styles.modal} onSubmit={(e) => void onSubmit(e)}>
        <div className={styles.header}>
          <h3 className={styles.title}>Create Task</h3>
          <button type="button" className={styles.close} onClick={onClose}>
            Close
          </button>
        </div>

        <div className={styles.body}>
          {inlineNotice ? <div className={styles.inlineNotice}>{inlineNotice}</div> : null}

          <div className={styles.row}>
            <span className={styles.label}>Title</span>
            <input
              ref={titleRef}
              className={styles.input}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="What needs doing?"
              maxLength={200}
            />
          </div>

          <div className={styles.row}>
            <span className={styles.label}>Stage</span>
            <select
              className={styles.select}
              value={stage_id}
              onChange={(e) => setStageId(e.target.value as StageId)}
            >
              {STAGES.map((s) => (
                <option key={s} value={s}>
                  {stageLabels[s] ?? DEFAULT_STAGE_LABELS[s]}
                </option>
              ))}
            </select>
          </div>

          <div className={styles.row}>
            <span className={styles.label}>Status</span>
            <select
              className={styles.select}
              value={_status}
              onChange={(e) => setStatus(e.target.value as Status)}
            >
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className={styles.footer}>
          <button className={styles.primary} type="submit" disabled={!canSubmit}>
            {saving ? "Creating..." : "Create"}
          </button>
        </div>
      </form>
    </div>
  );
}
