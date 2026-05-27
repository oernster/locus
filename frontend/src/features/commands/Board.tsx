import { useEffect, useMemo, useRef, useState } from "react";

import locusLogo from "../../assets/locus.png";

import type { CommandDTO, OutcomeDTO, StageId, Status } from "../../types/locus";
import type { BoardDTO, SessionDTO, SnapshotDTO } from "../../types/locus";

import {
  ListCommands,
  UpdateCommand,
  DeleteCommand,
  ReorderCommands,
  GetBoard,
  UpdateBoard,
  UpdateStageLabels,
  ResetBoard,
  GetActiveSession,
  StartSession,
  StopSession,
  GetLatestSessionsByStageId,
  ListSnapshots,
  SaveSnapshot,
  LoadSnapshot,
  DeleteSnapshot,
  RenameSnapshot,
  ListOutcomes,
  CreateOutcome,
  DeleteOutcome,
} from "../../../wailsjs/go/main/App";

import { DEFAULT_STAGE_LABELS, STAGES, STATUSES, BOTTOM_PANEL_LABELS } from "./constants";
import { CreateCommandModal } from "./CreateCommandModal";
import { CommandDrawer } from "./CommandDrawer";
import { DestructiveGuardModal } from "../../components/DestructiveGuardModal";
import { ConfirmDangerModal } from "../../components/ConfirmDangerModal";
import { FocusPanel } from "../focus/FocusPanel";

import styles from "./Board.module.css";

// Type alias for latest sessions map from backend.
type LatestByStageId = Record<string, SessionDTO>;

function statusClass(status: Status): string {
  switch (status) {
    case "Not Started":
      return styles.statusNotStarted;
    case "In Progress":
      return styles.statusInProgress;
    case "Blocked":
      return styles.statusBlocked;
    case "Complete":
      return styles.statusComplete;
  }
}

export function Board() {
  const [commands, setCommands] = useState<CommandDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [outcomesByCommandId, setOutcomesByCommandId] = useState<Record<number, OutcomeDTO[]>>({});

  const [draggingId, setDraggingId] = useState<number | null>(null);
  const [dropTarget, setDropTarget] = useState<
    | { stage_id: StageId; beforeId: number | null; afterId: number | null }
    | null
  >(null);

  const [activeSession, setActiveSession] = useState<SessionDTO>({ active: false });
  const [latestByStageId, setLatestByStageId] = useState<LatestByStageId>({});
  const [nowMs, setNowMs] = useState(() => Date.now());

  const [startMode, setStartMode] = useState(false);

  const [board, setBoard] = useState<BoardDTO | null>(null);
  const [boardNameDraft, setBoardNameDraft] = useState<string>("");
  const nameInputRef = useRef<HTMLInputElement | null>(null);

  const [editingStageId, setEditingStageId] = useState<StageId | null>(null);
  const [stageLabelDraft, setStageLabelDraft] = useState<string>("");

  const [snapshots, setSnapshots] = useState<SnapshotDTO[]>([]);
  const [snapshotsOpen, setSnapshotsOpen] = useState(false);

  type PendingDestructiveAction =
    | { type: "reset" }
    | { type: "loadSnapshot"; snapshotId: number; snapshotName?: string };

  const [pendingAction, setPendingAction] = useState<PendingDestructiveAction | null>(null);
  const [destructiveModalOpen, setDestructiveModalOpen] = useState(false);
  const [destructiveBusy, setDestructiveBusy] = useState(false);

  const [renamingSnapshotId, setRenamingSnapshotId] = useState<number | null>(null);
  const [snapshotNameDraft, setSnapshotNameDraft] = useState<string>("");
  const snapshotRenameInputRef = useRef<HTMLInputElement | null>(null);

  const [deleteSnapshotModalOpen, setDeleteSnapshotModalOpen] = useState(false);
  const [deleteSnapshotBusy, setDeleteSnapshotBusy] = useState(false);
  const [snapshotToDelete, setSnapshotToDelete] = useState<SnapshotDTO | null>(null);

  const [focusedStageId, setFocusedStageId] = useState<StageId>("PLAN");
  const [createFor, setCreateFor] = useState<StageId | null>(null);
  const [selected, setSelected] = useState<CommandDTO | null>(null);

  const [deleteCommandModalOpen, setDeleteCommandModalOpen] = useState(false);
  const [deleteCommandBusy, setDeleteCommandBusy] = useState(false);
  const [commandToDelete, setCommandToDelete] = useState<CommandDTO | null>(null);

  const [deleteOutcomeModalOpen, setDeleteOutcomeModalOpen] = useState(false);
  const [outcomeToDelete, setOutcomeToDelete] = useState<{ commandId: number; outcome: OutcomeDTO } | null>(null);

  const [drawerOutcomesNonce, setDrawerOutcomesNonce] = useState(0);

  const [outcomeDraftByCommandId, setOutcomeDraftByCommandId] = useState<Record<number, string>>({});
  const [outcomeEditBaseByCommandId, setOutcomeEditBaseByCommandId] = useState<Record<number, string>>({});
  const [outcomeComposerOpenByCommandId, setOutcomeComposerOpenByCommandId] = useState<Record<number, boolean>>({});
  const [savingOutcomeByCommandId, setSavingOutcomeByCommandId] = useState<Record<number, boolean>>({});

  const outcomeTextareaByCommandId = useRef<Record<number, HTMLTextAreaElement | null>>({});
  const suppressNextCardClickByCommandId = useRef<Record<number, boolean>>({});

  const [titleDraftByCommandId, setTitleDraftByCommandId] = useState<Record<number, string>>({});
  const [titleEditorOpenByCommandId, setTitleEditorOpenByCommandId] = useState<Record<number, boolean>>({});
  const [savingTitleByCommandId, setSavingTitleByCommandId] = useState<Record<number, boolean>>({});
  const titleInputByCommandId = useRef<Record<number, HTMLInputElement | null>>({});

  const stageLabels = useMemo((): Record<StageId, string> => {
    const overrides = board?.stage_labels ?? null;
    return {
      ...DEFAULT_STAGE_LABELS,
      ...(overrides ?? {}),
    } as Record<StageId, string>;
  }, [board?.stage_labels]);

  const commandsByStageId = useMemo(() => {
    const map = new Map<StageId, CommandDTO[]>();
    for (const s of STAGES) map.set(s, []);
    for (const cmd of commands) {
      map.get(cmd.stage_id as StageId)?.push(cmd);
    }
    return map;
  }, [commands]);

  function computeInsertIndex(list: CommandDTO[], beforeId: number | null, afterId: number | null): number {
    if (beforeId !== null) {
      const i = list.findIndex((c) => c.id === beforeId);
      return i >= 0 ? i : list.length;
    }
    if (afterId !== null) {
      const i = list.findIndex((c) => c.id === afterId);
      return i >= 0 ? i + 1 : list.length;
    }
    return list.length;
  }

  function buildReorderPayload(next: CommandDTO[]): Record<string, number[]> {
    const by: Record<string, number[]> = { PLAN: [], EXECUTE: [], CHECK: [], DONE: [] };
    for (const c of next) by[c.stage_id].push(c.id);
    return by;
  }

  async function commitReorder(next: CommandDTO[]): Promise<void> {
    const byStageId = buildReorderPayload(next);
    await ReorderCommands(byStageId);
  }

  async function refresh(): Promise<void> {
    setError(null);
    setLoading(true);
    try {
      const b = await GetBoard();
      setBoard(b);
      setBoardNameDraft(b.name);

      const items = await ListCommands("");
      setCommands((items ?? []) as unknown as CommandDTO[]);

      // Fetch outcomes for all commands.
      const byId: Record<number, OutcomeDTO[]> = {};
      for (const c of (items ?? [])) {
        const oo = await ListOutcomes(c.id);
        byId[c.id] = oo ?? [];
      }
      setOutcomesByCommandId(byId);

      const s = await GetActiveSession();
      setActiveSession((s ?? { active: false }) as unknown as SessionDTO);

      const latest = await GetLatestSessionsByStageId();
      setLatestByStageId((latest ?? {}) as unknown as LatestByStageId);

      const snaps = await ListSnapshots();
      setSnapshots(snaps ?? []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Failed to load";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  function openOutcomeComposer(commandId: number): void {
    openOutcomeComposerWithDraft(commandId, "");
  }

  function openOutcomeComposerWithDraft(commandId: number, draft: string): void {
    setOutcomeDraftByCommandId((prev) => ({ ...prev, [commandId]: draft }));
    setOutcomeEditBaseByCommandId((prev) => ({ ...prev, [commandId]: draft }));
    setOutcomeComposerOpenByCommandId((prev) => ({ ...prev, [commandId]: true }));
    window.setTimeout(() => {
      const el = outcomeTextareaByCommandId.current[commandId];
      el?.focus();
      if (draft.trim().length > 0) el?.select();
    }, 0);
  }

  function beginEditLatestOutcome(commandId: number): void {
    const latest = outcomesByCommandId[commandId]?.[0]?.note ?? "";
    openOutcomeComposerWithDraft(commandId, latest);
  }

  function collapseOutcomeComposer(commandId: number): void {
    setOutcomeComposerOpenByCommandId((prev) => ({ ...prev, [commandId]: false }));
    setOutcomeDraftByCommandId((prev) => ({ ...prev, [commandId]: "" }));
    setOutcomeEditBaseByCommandId((prev) => ({ ...prev, [commandId]: "" }));
  }

  async function commitInlineOutcome(commandId: number): Promise<void> {
    const draft = outcomeDraftByCommandId[commandId] ?? "";
    const trimmed = draft.trim();
    if (!trimmed) {
      collapseOutcomeComposer(commandId);
      return;
    }
    const base = (outcomeEditBaseByCommandId[commandId] ?? "").trim();
    if (trimmed === base) {
      collapseOutcomeComposer(commandId);
      return;
    }

    suppressNextCardClickByCommandId.current[commandId] = true;
    window.setTimeout(() => { suppressNextCardClickByCommandId.current[commandId] = false; }, 0);

    setSavingOutcomeByCommandId((prev) => ({ ...prev, [commandId]: true }));
    setError(null);
    try {
      const created = await CreateOutcome(commandId, trimmed);
      setOutcomesByCommandId((prev) => {
        const next = { ...prev };
        const existing = next[commandId] ?? [];
        next[commandId] = [created, ...existing];
        return next;
      });
      setDrawerOutcomesNonce((n) => n + 1);
      collapseOutcomeComposer(commandId);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not save outcome";
      setError(msg);
    } finally {
      setSavingOutcomeByCommandId((prev) => ({ ...prev, [commandId]: false }));
    }
  }

  async function onDeleteInlineOutcome(commandId: number, outcomeId: number): Promise<boolean> {
    suppressNextCardClickByCommandId.current[commandId] = true;
    window.setTimeout(() => { suppressNextCardClickByCommandId.current[commandId] = false; }, 0);

    setSavingOutcomeByCommandId((prev) => ({ ...prev, [commandId]: true }));
    setError(null);
    try {
      await DeleteOutcome(outcomeId);
      setOutcomesByCommandId((prev) => {
        const next = { ...prev };
        next[commandId] = (next[commandId] ?? []).filter((o) => o.id !== outcomeId);
        return next;
      });
      setDrawerOutcomesNonce((n) => n + 1);
      return true;
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not delete outcome";
      setError(msg);
      return false;
    } finally {
      setSavingOutcomeByCommandId((prev) => ({ ...prev, [commandId]: false }));
    }
  }

  function beginDeleteInlineOutcome(commandId: number, outcome: OutcomeDTO): void {
    suppressNextCardClickByCommandId.current[commandId] = true;
    window.setTimeout(() => { suppressNextCardClickByCommandId.current[commandId] = false; }, 0);
    setOutcomeToDelete({ commandId, outcome });
    setDeleteOutcomeModalOpen(true);
  }

  function closeDeleteOutcomeModal(): void {
    setDeleteOutcomeModalOpen(false);
    setOutcomeToDelete(null);
  }

  async function confirmDeleteInlineOutcome(): Promise<void> {
    if (!outcomeToDelete) return;
    const ok = await onDeleteInlineOutcome(outcomeToDelete.commandId, outcomeToDelete.outcome.id);
    if (ok) closeDeleteOutcomeModal();
  }

  useEffect(() => { void refresh(); }, []);

  useEffect(() => {
    if (renamingSnapshotId === null) return;
    const t = window.setTimeout(() => {
      snapshotRenameInputRef.current?.focus();
      snapshotRenameInputRef.current?.select();
    }, 0);
    return () => window.clearTimeout(t);
  }, [renamingSnapshotId]);

  function pencilSvg() {
    return (
      <svg className={styles.icon} viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path fill="currentColor" d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25Zm18.71-11.04a1.003 1.003 0 0 0 0-1.42l-2.5-2.5a1.003 1.003 0 0 0-1.42 0l-1.83 1.83 3.75 3.75 1.99-1.66Z" />
      </svg>
    );
  }

  function plusSvg() {
    return (
      <svg className={styles.icon} viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path fill="currentColor" d="M19 11H13V5a1 1 0 1 0-2 0v6H5a1 1 0 1 0 0 2h6v6a1 1 0 1 0 2 0v-6h6a1 1 0 1 0 0-2Z" />
      </svg>
    );
  }

  function gripSvg() {
    return (
      <svg className={styles.gripIcon} viewBox="0 0 16 16" aria-hidden="true" focusable="false">
        <circle cx="5" cy="4" r="1.2" />
        <circle cx="11" cy="4" r="1.2" />
        <circle cx="5" cy="8" r="1.2" />
        <circle cx="11" cy="8" r="1.2" />
        <circle cx="5" cy="12" r="1.2" />
        <circle cx="11" cy="12" r="1.2" />
      </svg>
    );
  }

  useEffect(() => {
    if (!board?.is_new_unnamed) return;
    const t = window.setTimeout(() => {
      nameInputRef.current?.focus();
      nameInputRef.current?.select();
    }, 0);
    return () => window.clearTimeout(t);
  }, [board?.is_new_unnamed]);

  useEffect(() => {
    function onDocMouseDown(e: MouseEvent): void {
      const target = e.target as HTMLElement | null;
      if (!target) return;
      if (target.closest(`.${styles.snapshotsMenu}`)) return;
      setSnapshotsOpen(false);
    }
    if (!snapshotsOpen) return;
    document.addEventListener("mousedown", onDocMouseDown);
    return () => document.removeEventListener("mousedown", onDocMouseDown);
  }, [snapshotsOpen]);

  useEffect(() => {
    const t = window.setInterval(() => setNowMs(Date.now()), 1000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    if (!startMode) return;
    function onKeyDown(e: KeyboardEvent): void {
      if (e.key === "Escape") setStartMode(false);
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [startMode]);

  function openCreate(_stage_id: StageId): void {
    setStartMode(false);
    setCreateFor("PLAN");
  }

  async function onChangeStatus(id: number, status: Status): Promise<void> {
    setError(null);
    try {
      await UpdateCommand(id, "", status, "");
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not update command";
      setError(msg);
    }
  }

  function openTitleEditor(commandId: number, currentTitle: string): void {
    suppressNextCardClickByCommandId.current[commandId] = true;
    window.setTimeout(() => { suppressNextCardClickByCommandId.current[commandId] = false; }, 0);
    setTitleDraftByCommandId((prev) => ({ ...prev, [commandId]: currentTitle }));
    setTitleEditorOpenByCommandId((prev) => ({ ...prev, [commandId]: true }));
    window.setTimeout(() => {
      const el = titleInputByCommandId.current[commandId];
      el?.focus();
      el?.select();
    }, 0);
  }

  function cancelTitleEditor(commandId: number): void {
    setTitleEditorOpenByCommandId((prev) => ({ ...prev, [commandId]: false }));
    setTitleDraftByCommandId((prev) => ({ ...prev, [commandId]: "" }));
  }

  async function commitTitleEditor(commandId: number): Promise<void> {
    const raw = titleDraftByCommandId[commandId] ?? "";
    const cleaned = raw.trim();
    if (!cleaned) { cancelTitleEditor(commandId); return; }
    const current = commands.find((c) => c.id === commandId)?.title ?? "";
    if (cleaned === current.trim()) { cancelTitleEditor(commandId); return; }

    setSavingTitleByCommandId((prev) => ({ ...prev, [commandId]: true }));
    setError(null);
    try {
      await UpdateCommand(commandId, cleaned, "", "");
      setCommands((prev) => prev.map((c) => (c.id === commandId ? { ...c, title: cleaned } : c)));
      cancelTitleEditor(commandId);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not update title";
      setError(msg);
    } finally {
      setSavingTitleByCommandId((prev) => ({ ...prev, [commandId]: false }));
    }
  }

  async function onDelete(id: number): Promise<boolean> {
    setError(null);
    setDeleteCommandBusy(true);
    try {
      await DeleteCommand(id);
      await refresh();
      setSelected((prev) => (prev?.id === id ? null : prev));
      return true;
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not delete command";
      setError(msg);
      return false;
    } finally {
      setDeleteCommandBusy(false);
    }
  }

  function beginDeleteCommand(cmd: CommandDTO): void {
    suppressNextCardClickByCommandId.current[cmd.id] = true;
    window.setTimeout(() => { suppressNextCardClickByCommandId.current[cmd.id] = false; }, 0);
    setCommandToDelete(cmd);
    setDeleteCommandModalOpen(true);
    setDeleteCommandBusy(false);
  }

  function closeDeleteCommandModal(): void {
    setDeleteCommandModalOpen(false);
    setDeleteCommandBusy(false);
    setCommandToDelete(null);
  }

  async function confirmDeleteCommand(): Promise<void> {
    if (!commandToDelete) return;
    const ok = await onDelete(commandToDelete.id);
    if (ok) closeDeleteCommandModal();
  }

  function onGripDragStart(e: React.DragEvent, cmd: CommandDTO): void {
    e.stopPropagation();
    setDraggingId(cmd.id);
    setDropTarget(null);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", String(cmd.id));
  }

  function onGripDragEnd(): void {
    setDraggingId(null);
    setDropTarget(null);
  }

  function onCardDragOver(e: React.DragEvent, stage_id: StageId, cmd: CommandDTO): void {
    if (draggingId === null) return;
    if (cmd.id === draggingId) { setDropTarget(null); return; }
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = "move";
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const before = e.clientY < rect.top + rect.height / 2;
    setDropTarget({ stage_id, beforeId: before ? cmd.id : null, afterId: before ? null : cmd.id });
  }

  function onColumnDragOver(e: React.DragEvent, stage_id: StageId): void {
    if (draggingId === null) return;
    if (e.target !== e.currentTarget) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDropTarget({ stage_id, beforeId: null, afterId: null });
  }

  async function onDrop(e: React.DragEvent): Promise<void> {
    if (draggingId === null || dropTarget === null) return;
    e.preventDefault();
    const moving = commands.find((c) => c.id === draggingId);
    if (!moving) return;

    const prev = commands;
    const without = prev.filter((c) => c.id !== draggingId);
    const targetList = without.filter((c) => c.stage_id === dropTarget.stage_id);
    const insertIndex = computeInsertIndex(targetList, dropTarget.beforeId, dropTarget.afterId);
    const moved: CommandDTO = { ...moving, stage_id: dropTarget.stage_id };
    const nextTarget = [...targetList.slice(0, insertIndex), moved, ...targetList.slice(insertIndex)];

    const next: CommandDTO[] = [];
    for (const s of STAGES) {
      if (s === dropTarget.stage_id) next.push(...nextTarget);
      else next.push(...without.filter((c) => c.stage_id === s));
    }

    setCommands(next);
    setDraggingId(null);
    setDropTarget(null);

    try {
      await commitReorder(next);
      await refresh();
    } catch (err) {
      setCommands(prev);
      const msg = err instanceof Error ? err.message : "Could not reorder commands";
      setError(msg);
    }
  }

  async function onStartSession(command_id: number): Promise<void> {
    setError(null);
    try {
      await StartSession(command_id);
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Session could not be started";
      setError(msg);
    }
  }

  async function onStopSession(): Promise<void> {
    setError(null);
    try {
      await StopSession();
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Session could not be stopped";
      setError(msg);
    }
  }

  async function onSaveSnapshot(): Promise<void> {
    setError(null);
    try {
      await SaveSnapshot("");
      const snaps = await ListSnapshots();
      setSnapshots(snaps ?? []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not save snapshot";
      setError(msg);
    }
  }

  function formatGuardSnapshotName(prefix: string): string {
    const now = new Date();
    const pad = (n: number) => n.toString().padStart(2, "0");
    const yyyy = now.getFullYear();
    const mm = pad(now.getMonth() + 1);
    const dd = pad(now.getDate());
    const hh = pad(now.getHours());
    const mi = pad(now.getMinutes());
    return `${prefix} - ${yyyy}-${mm}-${dd} ${hh}:${mi}`;
  }

  async function executePendingAction(action: PendingDestructiveAction): Promise<void> {
    if (action.type === "reset") {
      await ResetBoard();
      return;
    }
    await LoadSnapshot(action.snapshotId);
  }

  function closeDestructiveModal(): void {
    setDestructiveModalOpen(false);
    setPendingAction(null);
    setDestructiveBusy(false);
  }

  async function startGuardedAction(action: PendingDestructiveAction): Promise<void> {
    if (board?.is_empty) {
      setError(null);
      setSnapshotsOpen(false);
      try {
        await executePendingAction(action);
        await refresh();
      } catch (e) {
        const msg = e instanceof Error ? e.message :
          action.type === "reset" ? "Could not reset board" : "Could not load snapshot";
        setError(msg);
      }
      return;
    }
    setPendingAction(action);
    setSnapshotsOpen(false);
    setDestructiveModalOpen(true);
  }

  async function onModalSaveAndContinue(): Promise<void> {
    if (!pendingAction) return;
    setError(null);
    setDestructiveBusy(true);
    try {
      const namePrefix = pendingAction.type === "reset" ? "Before reset" : "Before loading snapshot";
      await SaveSnapshot(formatGuardSnapshotName(namePrefix));
      await executePendingAction(pendingAction);
      closeDestructiveModal();
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Operation failed";
      setError(msg);
      setDestructiveBusy(false);
    }
  }

  async function onModalContinueWithoutSaving(): Promise<void> {
    if (!pendingAction) return;
    setError(null);
    setDestructiveBusy(true);
    try {
      await executePendingAction(pendingAction);
      closeDestructiveModal();
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Operation failed";
      setError(msg);
      setDestructiveBusy(false);
    }
  }

  function beginRenameSnapshot(s: SnapshotDTO): void {
    setRenamingSnapshotId(s.id);
    setSnapshotNameDraft(s.name);
  }

  async function commitRenameSnapshot(snapshotId: number): Promise<void> {
    const cleaned = snapshotNameDraft.trim();
    if (!cleaned) { setRenamingSnapshotId(null); setSnapshotNameDraft(""); return; }
    setError(null);
    try {
      const updated = await RenameSnapshot(snapshotId, cleaned);
      setSnapshots((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not rename snapshot";
      setError(msg);
    } finally {
      setRenamingSnapshotId(null);
      setSnapshotNameDraft("");
    }
  }

  function beginDeleteSnapshot(s: SnapshotDTO): void {
    setRenamingSnapshotId(null);
    setSnapshotNameDraft("");
    setSnapshotToDelete(s);
    setDeleteSnapshotModalOpen(true);
  }

  function closeDeleteSnapshotModal(): void {
    setDeleteSnapshotModalOpen(false);
    setSnapshotToDelete(null);
    setDeleteSnapshotBusy(false);
  }

  async function confirmDeleteSnapshot(): Promise<void> {
    if (!snapshotToDelete) return;
    setError(null);
    setDeleteSnapshotBusy(true);
    try {
      await DeleteSnapshot(snapshotToDelete.id);
      closeDeleteSnapshotModal();
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Snapshot could not be deleted";
      setError(msg);
    } finally {
      setDeleteSnapshotBusy(false);
    }
  }

  async function onCommitBoardName(): Promise<void> {
    if (boardNameDraft === (board?.name ?? "")) return;
    setError(null);
    try {
      const next = await UpdateBoard(boardNameDraft);
      setBoard(next);
      setBoardNameDraft(next.name);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not update board name";
      setError(msg);
    }
  }

  const activeStageId: StageId | null =
    activeSession.active === false || !activeSession.stage_id
      ? null
      : (activeSession.stage_id as StageId);

  const activeCommandId: number | null =
    activeSession.active === false || !activeSession.command_id
      ? null
      : activeSession.command_id;

  const effectiveFocusedStageId: StageId = activeStageId ?? focusedStageId;

  const activeCommand: CommandDTO | null = useMemo(() => {
    if (activeCommandId === null) return null;
    return commands.find((c) => c.id === activeCommandId) ?? null;
  }, [activeCommandId, commands]);

  const startDisabled = activeStageId !== null;

  function formatDuration(seconds: number): string {
    const s = Math.max(0, Math.floor(seconds));
    const hh = Math.floor(s / 3600).toString().padStart(2, "0");
    const mm = Math.floor((s % 3600) / 60).toString().padStart(2, "0");
    const ss = Math.floor(s % 60).toString().padStart(2, "0");
    return `${hh}:${mm}:${ss}`;
  }

  const sessionTimerText = useMemo(() => {
    if (activeStageId === null || !activeSession.started_at) return null;
    const startMs = Date.parse(activeSession.started_at);
    if (Number.isNaN(startMs)) return null;
    return formatDuration((nowMs - startMs) / 1000);
  }, [activeSession, activeStageId, nowMs]);

  const RUNTIME_TOOLTIP = "Locus runs in your system tray. You can reopen it from there at any time.";

  function sessionStateParts(): { glyph: string; text: string; timeText: string | null } {
    if (activeStageId === null) return { glyph: "○", text: "No active session", timeText: null };
    const title = activeCommand?.title?.trim() || "Active session";
    return { glyph: "▶", text: title, timeText: sessionTimerText };
  }

  async function onToggleStartMode(): Promise<void> {
    if (startDisabled) return;
    if (startMode) { setStartMode(false); return; }
    if (commands.length === 0) { setError("Create a task first, then Start."); return; }
    setStartMode(true);
  }

  async function onCommitStageLabel(stage_id: StageId, nextLabel: string): Promise<void> {
    const cleaned = nextLabel.trim();
    if (!cleaned) return;
    setError(null);
    try {
      const next = await UpdateStageLabels({
        ...(board?.stage_labels ?? {}),
        [stage_id]: cleaned,
      });
      setBoard(next);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Could not update stage label";
      setError(msg);
    }
  }

  function formatLocal(iso: string): string | null {
    const ms = Date.parse(iso);
    if (Number.isNaN(ms)) return null;
    const date = new Date(ms);
    const datePart = new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short", year: "numeric" }).format(date);
    const timePart = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", hour12: false }).format(date);
    return `${datePart} ${timePart}`;
  }

  return (
    <section className={styles.root}>
      <div className={styles.controlSurface}>
        <div className={styles.titleBar}>
          <div className={styles.titleLeft}>
            <img
              className={styles.titleLogo}
              src={locusLogo}
              alt="Locus"
              aria-hidden="true"
            />

            <button
              type="button"
              className={styles.boardNameEditButton}
              title="Rename board"
              aria-label="Rename board"
              onClick={() => { nameInputRef.current?.focus(); nameInputRef.current?.select(); }}
            >
              {pencilSvg()}
            </button>

            <input
              ref={nameInputRef}
              className={`${styles.boardName} ${board?.is_new_unnamed ? styles.boardNameCue : ""}`}
              value={boardNameDraft}
              aria-label="Board name"
              onChange={(e) => setBoardNameDraft(e.target.value)}
              onBlur={() => void onCommitBoardName()}
              onKeyDown={(e) => { if (e.key === "Enter") e.currentTarget.blur(); }}
            />
          </div>

          <div className={styles.titleCenter}>
            {activeStageId === null ? (
              <>
                <button type="button" className={styles.opButton} title="Start a session by selecting a task" onClick={() => void onToggleStartMode()}>
                  <span className={styles.opIcon} aria-hidden="true">▶</span>
                  <span>Start</span>
                </button>
                <button type="button" className={styles.opButton} title="Add a task" onClick={() => openCreate(effectiveFocusedStageId)}>
                  <span className={styles.opIcon} aria-hidden="true">+</span>
                  <span>Add</span>
                </button>
              </>
            ) : (
              <>
                <button type="button" className={`${styles.opButton} ${styles.stopButton}`} title="Stop active session" onClick={() => void onStopSession()}>
                  <span className={styles.stopIcon} aria-hidden="true">■</span>
                  <span>Stop</span>
                </button>
                <button type="button" className={styles.opButton} title={`Add a task in ${stageLabels[effectiveFocusedStageId]}`} onClick={() => openCreate(effectiveFocusedStageId)}>
                  <span className={styles.opIcon} aria-hidden="true">+</span>
                  <span>Add</span>
                </button>
              </>
            )}
          </div>

          <div className={styles.titleRight}>
            {(() => {
              const sess = sessionStateParts();
              return (
                <div className={styles.stateCluster}>
                  <span className={styles.runtimeState} title={RUNTIME_TOOLTIP}>
                    {"● Running in tray"}
                  </span>
                  <span className={styles.stateSep} aria-hidden="true">•</span>

                  {activeCommand ? (
                    <button type="button" className={styles.sessionStateButton} title="Open active task" onClick={() => setSelected(activeCommand)}>
                      <span className={styles.sessionGlyph} aria-hidden="true">{sess.glyph}</span>
                      <span className={styles.sessionText}>{sess.text}</span>
                    </button>
                  ) : (
                    <span className={styles.sessionStateText}>
                      <span className={styles.sessionGlyph} aria-hidden="true">{sess.glyph}</span>
                      <span className={styles.sessionText}>{sess.text}</span>
                    </span>
                  )}

                  {sess.timeText ? (
                    <>
                      <span className={styles.stateSep} aria-hidden="true">•</span>
                      <span className={styles.sessionTimerInline}>{sess.timeText}</span>
                    </>
                  ) : null}
                </div>
              );
            })()}

            <div className={styles.titleMetaRight}>
              {loading ? <span className={styles.muted}>Loading...</span> : null}
              {error ? <span className={styles.error}>{error}</span> : null}
            </div>
          </div>
        </div>

        <div className={styles.structuralRow}>
          <button
            type="button"
            className={styles.structuralLeftButton}
            onClick={() => void startGuardedAction({ type: "reset" })}
            title="Reset board"
          >
            Reset board
          </button>

          <div className={styles.snapshotsMenu}>
            <button
              type="button"
              className={styles.structuralRightButton}
              onClick={() => setSnapshotsOpen((v) => !v)}
              aria-haspopup="menu"
              aria-expanded={snapshotsOpen}
            >
              <span>Snapshots</span>
              <span className={styles.dropdownChevron} aria-hidden="true">▼</span>
            </button>
            {snapshotsOpen ? (
              <div className={styles.dropdown} role="menu" aria-label="Snapshots">
                <button
                  type="button"
                  className={styles.dropdownItem}
                  onClick={() => { setSnapshotsOpen(false); void onSaveSnapshot(); }}
                >
                  Save snapshot
                </button>
                <div className={styles.dropdownDivider} aria-hidden="true" />

                {snapshots.length === 0 ? (
                  <div className={styles.dropdownEmpty}>No snapshots yet</div>
                ) : (
                  snapshots.map((s) => (
                    <div key={s.id} className={styles.snapshotRow} role="none">
                      {renamingSnapshotId === s.id ? (
                        <input
                          ref={snapshotRenameInputRef}
                          className={styles.snapshotRenameInput}
                          value={snapshotNameDraft}
                          aria-label="Snapshot name"
                          onChange={(e) => setSnapshotNameDraft(e.target.value)}
                          onBlur={() => void commitRenameSnapshot(s.id)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") e.currentTarget.blur();
                            if (e.key === "Escape") { setRenamingSnapshotId(null); setSnapshotNameDraft(""); }
                          }}
                        />
                      ) : (
                        <>
                          <button
                            type="button"
                            className={styles.snapshotEditButton}
                            title="Rename (F2)"
                            aria-label="Rename snapshot"
                            onClick={(e) => { e.preventDefault(); e.stopPropagation(); beginRenameSnapshot(s); }}
                          >
                            {pencilSvg()}
                          </button>
                          <button
                            type="button"
                            className={styles.dropdownItem}
                            onClick={() => void startGuardedAction({ type: "loadSnapshot", snapshotId: s.id, snapshotName: s.name })}
                            onKeyDown={(e) => { if (e.key === "F2") { e.preventDefault(); beginRenameSnapshot(s); } }}
                          >
                            <span className={styles.snapshotRowText}>
                              {s.name} - {formatLocal(s.saved_at) ?? s.saved_at}
                            </span>
                          </button>
                          <button
                            type="button"
                            className={styles.snapshotDeleteButton}
                            title="Delete"
                            aria-label="Delete snapshot"
                            onClick={(e) => { e.preventDefault(); e.stopPropagation(); beginDeleteSnapshot(s); }}
                          >
                            Delete
                          </button>
                        </>
                      )}
                    </div>
                  ))
                )}
              </div>
            ) : null}
          </div>
        </div>

        <div className={styles.helperRow}>
          {startMode ? (
            <span className={styles.helperText}>Select a task card to start a session. Press Esc to cancel.</span>
          ) : (
            <span className={styles.helperText}>Tip: drag the grip on a card to reorder or move it between stages.</span>
          )}
        </div>
      </div>

      {destructiveModalOpen && pendingAction ? (
        <DestructiveGuardModal
          title={pendingAction.type === "reset" ? "Reset board?" : "Load snapshot?"}
          body={pendingAction.type === "reset"
            ? "Your current board has content. Do you want to save a snapshot before clearing it?"
            : "Loading this snapshot will replace your current board. Do you want to save a snapshot first?"}
          busy={destructiveBusy}
          onSaveAndContinue={() => void onModalSaveAndContinue()}
          onContinueWithoutSaving={() => void onModalContinueWithoutSaving()}
          onCancel={closeDestructiveModal}
        />
      ) : null}

      {deleteSnapshotModalOpen && snapshotToDelete ? (
        <ConfirmDangerModal
          title="Delete snapshot?"
          body={`Are you sure you want to delete "${snapshotToDelete.name}"? This cannot be undone.`}
          busy={deleteSnapshotBusy}
          confirmLabel="Yes"
          cancelLabel="No"
          onConfirm={() => void confirmDeleteSnapshot()}
          onCancel={closeDeleteSnapshotModal}
        />
      ) : null}

      {deleteCommandModalOpen && commandToDelete ? (
        <ConfirmDangerModal
          title="Delete task?"
          body={`Delete "${commandToDelete.title}"? This will permanently remove the task and its outcomes/sessions. This cannot be undone.`}
          busy={deleteCommandBusy}
          confirmLabel="Delete"
          cancelLabel="Cancel"
          onConfirm={() => void confirmDeleteCommand()}
          onCancel={closeDeleteCommandModal}
        />
      ) : null}

      {deleteOutcomeModalOpen && outcomeToDelete ? (
        <ConfirmDangerModal
          title="Delete outcome?"
          body={`Delete this outcome? "${outcomeToDelete.outcome.note}" This cannot be undone.`}
          busy={!!savingOutcomeByCommandId[outcomeToDelete.commandId]}
          confirmLabel="Delete"
          cancelLabel="Cancel"
          onConfirm={() => void confirmDeleteInlineOutcome()}
          onCancel={closeDeleteOutcomeModal}
        />
      ) : null}

      <div className={`${styles.board} ${activeStageId ? styles.boardHasActive : ""} ${startMode ? styles.boardStartMode : ""}`}>
        {STAGES.map((stage_id) => (
          <div
            key={stage_id}
            className={`${styles.column} ${activeStageId === stage_id ? styles.columnActive : ""} ${dropTarget?.stage_id === stage_id && draggingId !== null ? styles.columnDropTarget : ""} ${effectiveFocusedStageId === stage_id ? styles.columnFocused : ""}`}
            onMouseDown={() => setFocusedStageId(stage_id)}
          >
            <div className={styles.columnHeader}>
              {editingStageId === stage_id ? (
                <input
                  className={styles.stageLabelInput}
                  value={stageLabelDraft}
                  aria-label="Stage label"
                  onChange={(e) => setStageLabelDraft(e.target.value)}
                  onBlur={() => { setEditingStageId(null); void onCommitStageLabel(stage_id, stageLabelDraft); }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") (e.currentTarget as HTMLInputElement).blur();
                    if (e.key === "Escape") { setEditingStageId(null); setStageLabelDraft(stageLabels[stage_id]); }
                  }}
                  autoFocus
                />
              ) : (
                <div className={styles.stageTitleRow}>
                  <button
                    type="button"
                    className={styles.stageRenameIconButton}
                    title="Rename stage"
                    aria-label="Rename stage"
                    onMouseDown={(e) => e.stopPropagation()}
                    onClick={(e) => { e.stopPropagation(); setEditingStageId(stage_id); setStageLabelDraft(stageLabels[stage_id]); }}
                  >
                    {pencilSvg()}
                  </button>
                  <h2 className={styles.columnTitle}>{stageLabels[stage_id]}</h2>
                </div>
              )}
            </div>

            {latestByStageId[stage_id] ? (
              <div className={styles.paneSessionMeta}>
                {latestByStageId[stage_id]?.started_at ? (
                  <span>Started: {formatLocal(latestByStageId[stage_id]!.started_at!)}</span>
                ) : null}
                {latestByStageId[stage_id]?.ended_at ? (
                  <span>Ended: {formatLocal(latestByStageId[stage_id]!.ended_at!)}</span>
                ) : activeStageId === stage_id && sessionTimerText ? (
                  <span>Elapsed: {sessionTimerText}</span>
                ) : null}
              </div>
            ) : null}

            <div
              className={styles.cards}
              onDragOver={(e) => onColumnDragOver(e, stage_id)}
              onDrop={(e) => void onDrop(e)}
            >
              {(commandsByStageId.get(stage_id) ?? []).length === 0 ? (
                <div className={styles.empty}>No tasks yet</div>
              ) : null}

              {(commandsByStageId.get(stage_id) ?? []).map((cmd) => (
                <div
                  key={cmd.id}
                  className={`${styles.card} ${draggingId === cmd.id ? styles.cardDragging : ""} ${activeCommandId === cmd.id ? styles.cardActiveTask : ""} ${dropTarget?.stage_id === stage_id && dropTarget.beforeId === cmd.id ? styles.cardDropBefore : ""} ${dropTarget?.stage_id === stage_id && dropTarget.afterId === cmd.id ? styles.cardDropAfter : ""}`}
                  onClickCapture={(e) => {
                    if (!startMode || startDisabled) return;
                    e.preventDefault();
                    e.stopPropagation();
                    setStartMode(false);
                    void onStartSession(cmd.id);
                  }}
                  onClick={() => {
                    if (suppressNextCardClickByCommandId.current[cmd.id]) return;
                    setSelected(cmd);
                  }}
                  onDragOver={(e) => onCardDragOver(e, stage_id, cmd)}
                  onDrop={(e) => { e.stopPropagation(); void onDrop(e); }}
                >
                  <div className={styles.cardTop}>
                    <div className={styles.cardTitleRow}>
                      <div className={styles.cardTitleLeft}>
                        {titleEditorOpenByCommandId[cmd.id] ? (
                          <input
                            ref={(el) => { titleInputByCommandId.current[cmd.id] = el; }}
                            className={styles.cardTitleInput}
                            value={titleDraftByCommandId[cmd.id] ?? ""}
                            aria-label="Task title"
                            maxLength={200}
                            onMouseDown={(e) => e.stopPropagation()}
                            onClick={(e) => e.stopPropagation()}
                            onChange={(e) => setTitleDraftByCommandId((prev) => ({ ...prev, [cmd.id]: e.target.value }))}
                            onBlur={() => void commitTitleEditor(cmd.id)}
                            onKeyDown={(e) => {
                              if (e.key === "Escape") { e.preventDefault(); cancelTitleEditor(cmd.id); return; }
                              if (e.key === "Enter") { e.preventDefault(); void commitTitleEditor(cmd.id); }
                            }}
                          />
                        ) : (
                          <button
                            type="button"
                            className={styles.cardTitleButton}
                            title="Rename task"
                            aria-label="Rename task"
                            onMouseDown={(e) => e.stopPropagation()}
                            onClick={(e) => { e.stopPropagation(); openTitleEditor(cmd.id, cmd.title); }}
                          >
                            <span className={styles.cardTitleIcon} aria-hidden="true">{pencilSvg()}</span>
                            <span className={styles.cardTitle}>{cmd.title}</span>
                          </button>
                        )}
                        {savingTitleByCommandId[cmd.id] ? <span className={styles.cardTitleSaving}>Saving...</span> : null}
                      </div>
                    </div>

                    <div className={styles.cardControlsRow} onMouseDown={(e) => e.stopPropagation()} onClick={(e) => e.stopPropagation()}>
                      <div className={styles.cardControlsLeft}>
                        <label className={styles.statusLabel}>
                          <span className={styles.visuallyHidden}>Status</span>
                          <select
                            className={styles.statusSelect}
                            value={cmd.status}
                            onChange={(e) => void onChangeStatus(cmd.id, e.target.value as Status)}
                          >
                            {STATUSES.map((s) => (<option key={s} value={s}>{s}</option>))}
                          </select>
                        </label>
                      </div>

                      <div className={styles.cardControlsRight}>
                        <button
                          type="button"
                          className={styles.dragHandle}
                          draggable
                          aria-label="Reorder"
                          title="Drag to reorder"
                          onDragStart={(e) => onGripDragStart(e, cmd)}
                          onDragEnd={onGripDragEnd}
                        >
                          {gripSvg()}
                        </button>
                        <span className={`${styles.statusDot} ${statusClass(cmd.status)}`} aria-label={`Status: ${cmd.status}`} title={cmd.status} />
                        <button
                          type="button"
                          className={styles.dangerIconButton}
                          onClick={() => beginDeleteCommand(cmd)}
                          aria-label="Delete task"
                          title="Delete"
                        >
                          Del
                        </button>
                      </div>
                    </div>

                    <div className={styles.cardOutcomeInline} onMouseDown={(e) => e.stopPropagation()} onClick={(e) => e.stopPropagation()}>
                      {outcomeComposerOpenByCommandId[cmd.id] ? (
                        <textarea
                          ref={(el) => { outcomeTextareaByCommandId.current[cmd.id] = el; }}
                          className={styles.cardOutcomeTextarea}
                          value={outcomeDraftByCommandId[cmd.id] ?? ""}
                          placeholder="What happened?"
                          onChange={(e) => setOutcomeDraftByCommandId((prev) => ({ ...prev, [cmd.id]: e.target.value }))}
                          onKeyDown={(e) => {
                            if (e.key === "Escape") { e.preventDefault(); collapseOutcomeComposer(cmd.id); return; }
                            if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); void commitInlineOutcome(cmd.id); }
                          }}
                          onBlur={() => void commitInlineOutcome(cmd.id)}
                          rows={2}
                        />
                      ) : (
                        <>
                          {(outcomesByCommandId[cmd.id] ?? []).length === 0 ? (
                            <button
                              type="button"
                              className={styles.cardOutcomeRowButton}
                              title="Add an outcome"
                              aria-label="Add outcome"
                              onMouseDown={(e) => e.stopPropagation()}
                              onClick={(e) => { e.stopPropagation(); openOutcomeComposer(cmd.id); }}
                            >
                              <span className={styles.cardOutcomeRowIcon} aria-hidden="true">{plusSvg()}</span>
                              <span className={`${styles.cardOutcomeRowText} ${styles.cardOutcomeRowAddText}`}>Add outcome</span>
                            </button>
                          ) : (
                            <div className={styles.cardOutcomeList}>
                              {(outcomesByCommandId[cmd.id] ?? []).map((o) => (
                                <div key={o.id} className={styles.cardOutcomeItem}>
                                  <div className={styles.cardOutcomeRow}>
                                    <button
                                      type="button"
                                      className={styles.cardOutcomeRowButton}
                                      title="Edit latest outcome (saves as a new outcome entry)"
                                      aria-label="Edit latest outcome"
                                      onMouseDown={(e) => e.stopPropagation()}
                                      onClick={(e) => { e.stopPropagation(); beginEditLatestOutcome(cmd.id); }}
                                    >
                                      <span className={styles.cardOutcomeRowIcon} aria-hidden="true">{pencilSvg()}</span>
                                      <span className={styles.cardOutcomeRowText}>{o.note}</span>
                                    </button>
                                    <button
                                      type="button"
                                      className={styles.cardOutcomeDeleteButton}
                                      aria-label="Delete outcome"
                                      title="Delete"
                                      onMouseDown={(e) => e.stopPropagation()}
                                      onClick={(e) => { e.stopPropagation(); beginDeleteInlineOutcome(cmd.id, o); }}
                                      disabled={!!savingOutcomeByCommandId[cmd.id]}
                                    >
                                      Del
                                    </button>
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}

                          {(outcomesByCommandId[cmd.id] ?? []).length > 0 ? (
                            <div className={styles.cardOutcomeAddRow}>
                              <button
                                type="button"
                                className={styles.cardOutcomeAdd}
                                title="Add a new outcome"
                                aria-label="Add outcome"
                                onMouseDown={(e) => e.stopPropagation()}
                                onClick={(e) => { e.stopPropagation(); openOutcomeComposer(cmd.id); }}
                              >
                                + Add outcome
                              </button>
                            </div>
                          ) : null}
                        </>
                      )}

                      {savingOutcomeByCommandId[cmd.id] ? (
                        <div className={styles.cardOutcomeSaving}>Saving...</div>
                      ) : null}
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Bottom half: FocusPanel with fixed label */}
            <div className={styles.bottomPanelLabel}>{BOTTOM_PANEL_LABELS[stage_id]}</div>
            <FocusPanel stageId={stage_id} />
          </div>
        ))}
      </div>

      {createFor ? (
        <CreateCommandModal
          initialStageId={createFor}
          stageLabels={stageLabels}
          onClose={() => setCreateFor(null)}
          onCreated={refresh}
          setError={(msg) => setError(msg || null)}
        />
      ) : null}

      {selected ? (
        <CommandDrawer
          command={selected}
          stageLabels={stageLabels}
          onClose={() => setSelected(null)}
          onRefreshCommands={refresh}
          setError={(msg) => setError(msg || null)}
          outcomesRefreshNonce={drawerOutcomesNonce}
        />
      ) : null}
    </section>
  );
}
