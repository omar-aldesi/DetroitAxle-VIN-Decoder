import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Trash2,
  Edit3,
  Check,
  X,
  Loader2,
  Tag,
  MessageSquare,
  Hash,
  ChevronDown,
  Copy,
  AlertTriangle,
  CheckCircle2,
} from "lucide-react";
import { updateNote, deleteNote } from "../api/notes";
import { copyText } from "../utils/clipboard";
import { useAuth } from "../contexts/AuthContext";
import { useToast } from "../contexts/ToastContext";

/* ── Relative time ──────────────────────────────────────────────────── */
function timeAgo(dateStr) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const m = Math.floor(diff / 60_000);
  const h = Math.floor(diff / 3_600_000);
  const d = Math.floor(diff / 86_400_000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  if (h < 24) return `${h}h ago`;
  if (d < 7) return `${d}d ago`;
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

/* ── Colour-stable avatar ───────────────────────────────────────────── */
const AVATAR_COLORS = [
  "bg-blue-600",
  "bg-violet-600",
  "bg-emerald-600",
  "bg-amber-600",
  "bg-rose-600",
  "bg-cyan-600",
];
function Avatar({ username }) {
  const initials = (username ?? "?")
    .split(/[._\s@-]/)
    .map((s) => s[0] ?? "")
    .join("")
    .toUpperCase()
    .slice(0, 2);
  const color =
    AVATAR_COLORS[(username?.charCodeAt(0) ?? 0) % AVATAR_COLORS.length];
  return (
    <div
      className={`${color} w-7 h-7 rounded-full flex items-center justify-center text-[11px] font-bold text-white shrink-0`}
    >
      {initials}
    </div>
  );
}

/* ── Styled select ──────────────────────────────────────────────────── */
function StyledSelect({ value, onChange, children }) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={onChange}
        className="appearance-none w-full bg-bg-card border border-border-subtle rounded-lg px-3 py-2 text-sm text-txt-primary focus:outline-none focus:border-accent transition-all pr-9"
      >
        {children}
      </select>
      <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-txt-muted pointer-events-none" />
    </div>
  );
}

/* ════════════════════════════════════════════════════════════════════
   NoteCard
══════════════════════════════════════════════════════════════════════ */
export default function NoteCard({ note, vin, categories }) {
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState(
    note.note_type === "free_text" || note.note_type === "listing_error"
      ? (note.free_text ?? "")
      : (note.part_number ?? ""),
  );
  const [editPartNumber, setEditPartNumber] = useState(note.part_number ?? "");
  const [editCategory, setEditCategory] = useState(note.part_category_id ?? "");
  const [confirming, setConfirming] = useState(false);

  const { user } = useAuth();
  const toast = useToast();
  const qc = useQueryClient();
  const isOwner =
    user && (user.id === note.user_id || user.username === note.username);

  const isFreeText = note.note_type === "free_text";
  const isListingError = note.note_type === "listing_error";
  const isResolved = isListingError && note.is_resolved;

  /* Escape dismisses delete confirm */
  useEffect(() => {
    if (!confirming) return;
    const h = (e) => {
      if (e.key === "Escape") setConfirming(false);
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [confirming]);

  const updateMut = useMutation({
    mutationFn: (p) => updateNote(note.note_id, p),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["vehicle", vin] });
      setEditing(false);
      toast("Note updated", "success");
    },
    onError: (err) =>
      toast(err.response?.data?.error ?? "Failed to update note", "error"),
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteNote(note.note_id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["vehicle", vin] });
      toast("Note deleted", "info");
    },
    onError: (err) =>
      toast(err.response?.data?.error ?? "Failed to delete note", "error"),
  });

  const saveEdit = () => {
    if (isFreeText) {
      if (!editText.trim()) return;
      updateMut.mutate({ free_text: editText.trim() });
    } else if (isListingError) {
      if (!editText.trim()) return;
      updateMut.mutate({
        free_text: editText.trim(),
        part_number: editPartNumber.trim() || null,
      });
    } else {
      if (!editText.trim()) return;
      updateMut.mutate({
        part_number: editText.trim(),
        part_category_id: editCategory ? Number(editCategory) : null,
      });
    }
  };

  const copyPart = async () => {
    try {
      await copyText(note.part_number ?? "");
      toast("Part number copied", "info");
    } catch {
      toast("Could not copy", "error");
    }
  };

  const copyNoteText = async () => {
    try {
      await copyText(note.free_text ?? "");
      toast("Note text copied", "info");
    } catch {
      toast("Could not copy", "error");
    }
  };

  return (
    <div
      className={`rounded-xl border p-4 transition-all duration-150 animate-fade-in ${
        isListingError
          ? isResolved
            ? "bg-success/5 border-success/20 hover:border-success/35"
            : "bg-danger/5 border-danger/20 hover:border-danger/35"
          : isFreeText
            ? "bg-bg-elevated border-border-subtle hover:border-border"
            : "bg-warn/5 border-warn/20 hover:border-warn/35"
      }`}
    >
      {/* ── Header ───────────────────────────────────────────────── */}
      <div className="flex items-start justify-between gap-2 mb-3">
        <div className="flex items-center gap-2 min-w-0">
          <Avatar username={note.username} />
          <div className="min-w-0">
            <p className="text-xs font-semibold text-txt-primary leading-none truncate">
              {note.username}
            </p>
            <div className="flex items-center gap-1.5 mt-0.5">
              <p
                className="text-[10px] text-txt-muted"
                title={new Date(note.created_at).toLocaleString()}
              >
                {timeAgo(note.created_at)}
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-1 shrink-0">
          {/* Build-number scope — warns when this note may not apply to the viewed VIN */}
          {note.scope === "warn" && (
            <span
              className="badge bg-warn/10 text-warn border border-warn/25"
              title={
                note.origin_serial != null
                  ? `Entered for build number …${String(note.origin_serial).padStart(6, "0")}. It may not apply to the VIN you're viewing.`
                  : "Entered without a build number — it may not apply to this exact VIN."
              }
            >
              <AlertTriangle className="w-3 h-3" />
              Check VIN
            </span>
          )}
          {/* Dynamic Badge for Note Type / Status */}
          {isListingError ? (
            isResolved ? (
              <span className="badge bg-success/10 text-success">
                <CheckCircle2 className="w-3 h-3" />
                Resolved
              </span>
            ) : (
              <span className="badge bg-danger/10 text-danger">
                <AlertTriangle className="w-3 h-3" />
                Action Required
              </span>
            )
          ) : isFreeText ? (
            <span className="badge bg-accent/10 text-accent">
              <MessageSquare className="w-3 h-3" />
              Note
            </span>
          ) : (
            <span className="badge bg-warn/10 text-warn">
              <Hash className="w-3 h-3" />
              Part
            </span>
          )}

          {isOwner && !editing && (
            <>
              <button
                onClick={() => {
                  setEditing(true);
                  setConfirming(false);
                }}
                className="p-1 rounded text-txt-muted hover:text-txt-primary transition-colors"
                title="Edit"
              >
                <Edit3 className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={() => {
                  setConfirming(true);
                  setEditing(false);
                }}
                className="p-1 rounded text-txt-muted hover:text-danger transition-colors"
                title="Delete"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            </>
          )}
        </div>
      </div>

      {/* ── Content ──────────────────────────────────────────────── */}
      {editing ? (
        <div className="space-y-2.5">
          <textarea
            autoFocus
            value={editText}
            onChange={(e) => setEditText(e.target.value)}
            rows={isFreeText || isListingError ? 3 : 1}
            placeholder={
              isFreeText
                ? "Note text…"
                : isListingError
                  ? "Error description…"
                  : "Part number…"
            }
            className={`w-full bg-bg-card border rounded-lg px-3 py-2 text-sm
                       text-txt-primary placeholder:text-txt-muted resize-none
                       focus:outline-none transition-all ${
                         isListingError
                           ? isResolved
                             ? "border-success/30 focus:border-success font-sans"
                             : "border-danger/30 focus:border-danger font-sans"
                           : isFreeText
                             ? "border-accent/30 focus:border-accent font-sans"
                             : "border-warn/30 focus:border-warn font-mono"
                       }`}
          />
          {isListingError && (
            <input
              type="text"
              value={editPartNumber}
              onChange={(e) => setEditPartNumber(e.target.value.toUpperCase())}
              placeholder="Related part # (optional)"
              className="w-full bg-bg-card border border-border-subtle rounded-lg px-3 py-2 text-sm font-mono text-txt-primary placeholder:text-txt-muted placeholder:font-sans focus:outline-none focus:border-danger transition-all"
            />
          )}
          {!isFreeText && !isListingError && (
            <StyledSelect
              value={editCategory}
              onChange={(e) => setEditCategory(e.target.value)}
            >
              <option value="">No category</option>
              {categories.map((c) => (
                <option key={c.category_id} value={c.category_id}>
                  {c.name}
                </option>
              ))}
            </StyledSelect>
          )}
          <div className="flex gap-2 justify-end">
            <button
              onClick={() => setEditing(false)}
              className="px-3 py-1.5 text-xs border border-border-subtle rounded-lg text-txt-muted hover:text-txt-primary transition-all"
            >
              Cancel
            </button>
            <button
              onClick={saveEdit}
              disabled={updateMut.isPending}
              className={`px-3 py-1.5 text-xs disabled:opacity-60 text-white rounded-lg transition-all flex items-center gap-1 ${
                isListingError
                  ? isResolved
                    ? "bg-success hover:opacity-90"
                    : "bg-danger hover:opacity-90"
                  : "bg-accent hover:bg-accent-hover"
              }`}
            >
              {updateMut.isPending && (
                <Loader2 className="w-3 h-3 animate-spin" />
              )}
              Save
            </button>
          </div>
        </div>
      ) : isListingError ? (
        <div className="space-y-2.5">
          {/* Description */}
          <div className="group/copy relative">
            <p className="text-sm text-txt-primary leading-relaxed whitespace-pre-wrap pr-6">
              {note.free_text}
            </p>
            <button
              onClick={copyNoteText}
              title="Copy description"
              className="absolute top-0 right-0 opacity-0 group-hover/copy:opacity-100 transition-opacity text-txt-muted hover:text-danger"
            >
              <Copy className="w-3.5 h-3.5" />
            </button>
          </div>

          {/* Related part number — only shown if present */}
          {note.part_number && (
            <button
              onClick={copyPart}
              title="Click to copy part number"
              className={`group/pn flex items-center gap-2 w-full text-left border rounded-lg px-3 py-2 ${
                isResolved
                  ? "bg-success/10 border-success/20 text-success"
                  : "bg-danger/8 border-danger/15 text-danger"
              }`}
            >
              <Hash className="w-3.5 h-3.5 shrink-0" />
              <span className="font-mono text-xs text-txt-primary font-semibold tracking-wide">
                {note.part_number}
              </span>
              <span className="text-[10px] text-txt-muted ml-1">
                related part
              </span>
              <Copy className="w-3 h-3 text-txt-muted opacity-0 group-hover/pn:opacity-100 transition-opacity ml-auto" />
            </button>
          )}

          {/* Resolution Note — only shown if resolved and note exists */}
          {isResolved && note.resolve_note && (
            <div className="mt-3 p-3 bg-success/10 border border-success/20 rounded-lg">
              <p className="text-[10px] text-success font-bold uppercase tracking-wider mb-1 flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" />
                Resolution Note
              </p>
              <p className="text-sm text-txt-secondary leading-relaxed whitespace-pre-wrap">
                {note.resolve_note}
              </p>
            </div>
          )}
        </div>
      ) : isFreeText ? (
        <div className="group/copy relative">
          <p className="text-sm text-txt-primary leading-relaxed whitespace-pre-wrap pr-6">
            {note.free_text}
          </p>
          <button
            onClick={copyNoteText}
            title="Copy text"
            className="absolute top-0 right-0 opacity-0 group-hover/copy:opacity-100 transition-opacity text-txt-muted hover:text-accent"
          >
            <Copy className="w-3.5 h-3.5" />
          </button>
        </div>
      ) : (
        <div className="space-y-1.5">
          {/* Part number — click to copy */}
          <button
            onClick={copyPart}
            title="Click to copy part number"
            className="group/pn flex items-center gap-2 w-full text-left"
          >
            <Hash className="w-3.5 h-3.5 text-warn shrink-0" />
            <span className="font-mono text-sm text-txt-primary font-semibold tracking-wide">
              {note.part_number}
            </span>
            <Copy className="w-3 h-3 text-txt-muted opacity-0 group-hover/pn:opacity-100 transition-opacity ml-auto" />
          </button>
          {note.part_category && (
            <div className="flex items-center gap-1.5">
              <Tag className="w-3 h-3 text-txt-muted" />
              <span className="text-xs text-txt-muted">
                {note.part_category.name}
              </span>
            </div>
          )}
        </div>
      )}

      {/* ── Delete confirm ───────────────────────────────────────── */}
      {confirming && (
        <div className="mt-3 flex items-center gap-2 bg-danger/10 border border-danger/20 rounded-lg px-3 py-2 animate-fade-in">
          <p className="text-xs text-danger flex-1">Delete this note?</p>
          <button
            onClick={() => {
              deleteMut.mutate();
              setConfirming(false);
            }}
            disabled={deleteMut.isPending}
            className="px-2.5 py-1 text-xs bg-danger hover:opacity-90 text-white rounded-md transition-opacity flex items-center gap-1"
          >
            {deleteMut.isPending ? (
              <Loader2 className="w-3 h-3 animate-spin" />
            ) : (
              "Delete"
            )}
          </button>
          <button
            onClick={() => setConfirming(false)}
            className="px-2.5 py-1 text-xs border border-border-subtle rounded-md text-txt-muted hover:text-txt-primary transition-colors"
          >
            Cancel
          </button>
        </div>
      )}
    </div>
  );
}
