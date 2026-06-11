import { useState, useRef, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Clock, ArrowRight, X, ShieldCheck, ShieldAlert, Trash2 } from "lucide-react";
import { useAuth } from "../contexts/AuthContext";
import { deleteHistoryEntry } from "../api/history";

export default function FieldHistoryIndicator({
  fieldName,
  history = [],
  vin,
  /** "below" (default) | "above" — use "above" when the popover would be clipped */
  placement = "below",
  /** "start" (default) | "end" — align popover to trigger's left or right edge */
  align = "start",
}) {
  const entries = history.filter((h) => h.field_name === fieldName);
  const [open, setOpen] = useState(false);
  const [deletingId, setDeletingId] = useState(null); // id currently showing confirm
  const ref = useRef(null);
  const qc = useQueryClient();
  const { user } = useAuth();

  useEffect(() => {
    if (!open) return;
    const handler = (e) => {
      if (ref.current && !ref.current.contains(e.target)) {
        setOpen(false);
        setDeletingId(null);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  if (entries.length === 0) return null;

  const count = entries.length;

  const handleDelete = async (entryId) => {
    try {
      await deleteHistoryEntry(entryId);
      // Refetch the vehicle so the reverted field value + updated history show immediately
      if (vin) qc.invalidateQueries({ queryKey: ["vehicle", vin] });
      setDeletingId(null);
      if (entries.length <= 1) setOpen(false);
    } catch {
      // silently ignore — user can retry
    }
  };

  return (
    <div ref={ref} className="relative flex items-center">
      {/* ── Pill trigger ── */}
      <button
        onClick={() => { setOpen((v) => !v); setDeletingId(null); }}
        title="View edit history"
        className={`
          flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-[11px] font-medium
          transition-all duration-150 select-none
          ${
            open
              ? "bg-accent/15 border-accent/40 text-accent"
              : "bg-bg-elevated border-border-subtle text-txt-muted hover:border-accent/30 hover:text-accent hover:bg-accent/8"
          }
        `}
      >
        <Clock className="w-3 h-3 shrink-0" />
        <span className="font-mono tabular-nums">{count}</span>
      </button>

      {/* ── Popover ── */}
      {open && (
        <div
          className={`absolute z-[200] w-80 rounded-xl border border-border-subtle bg-bg-card shadow-card ${
            placement === "above" ? "bottom-full mb-2" : "top-full mt-2"
          } ${align === "end" ? "right-0" : "left-0"}`}
          style={{
            animation:
              placement === "above"
                ? "fhi-rise 140ms cubic-bezier(0.16,1,0.3,1) both"
                : "fhi-drop 140ms cubic-bezier(0.16,1,0.3,1) both",
          }}
        >
          <style>{`
            @keyframes fhi-drop {
              from { opacity: 0; transform: translateY(-6px) scale(0.97); }
              to   { opacity: 1; transform: translateY(0)   scale(1);    }
            }
            @keyframes fhi-rise {
              from { opacity: 0; transform: translateY(6px) scale(0.97); }
              to   { opacity: 1; transform: translateY(0)  scale(1);    }
            }
          `}</style>

          {/* Header */}
          <div className="flex items-center justify-between px-4 py-2.5 border-b border-border-subtle">
            <div className="flex items-center gap-2">
              <Clock className="w-3.5 h-3.5 text-accent" />
              <span className="text-xs font-semibold text-txt-primary tracking-wide">
                Edit History
              </span>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded-md bg-bg-elevated text-txt-muted border border-border-subtle tabular-nums">
                {count}
              </span>
            </div>
            <button
              onClick={() => { setOpen(false); setDeletingId(null); }}
              className="text-txt-muted hover:text-txt-primary transition-colors rounded-lg p-0.5 hover:bg-bg-elevated"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>

          {/* Entries */}
          <ul className="divide-y divide-border-subtle max-h-64 overflow-y-auto overscroll-contain">
            {[...entries].reverse().map((entry, i) => {
              const canDelete =
                user &&
                (user.role === "admin" || entry.username === user.username);

              const date = entry.created_at
                ? new Date(entry.created_at).toLocaleDateString(undefined, {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                  })
                : null;
              const time = entry.created_at
                ? new Date(entry.created_at).toLocaleTimeString(undefined, {
                    hour: "2-digit",
                    minute: "2-digit",
                  })
                : null;

              const isConfirming = deletingId === entry.id;

              return (
                <li key={entry.id ?? i} className="px-4 py-3 group/entry relative">
                  {/* Value change row */}
                  <div className="flex items-center gap-2 mb-2">
                    <span
                      className="inline-block px-2 py-0.5 rounded-md text-xs font-mono
                        bg-danger/10 text-danger border border-danger/20
                        line-through truncate max-w-[80px]"
                      title={entry.old_value || "empty"}
                    >
                      {entry.old_value || "empty"}
                    </span>

                    <ArrowRight className="w-3.5 h-3.5 text-txt-muted shrink-0" />

                    <span
                      className="inline-block px-2 py-0.5 rounded-md text-xs font-mono
                        bg-success/10 text-success border border-success/20
                        truncate max-w-[80px]"
                      title={entry.new_value || "empty"}
                    >
                      {entry.new_value || "empty"}
                    </span>

                    {entry.is_trusted !== undefined && (
                      <span
                        className="ml-auto shrink-0"
                        title={entry.is_trusted ? "Trusted edit" : "Unverified edit"}
                      >
                        {entry.is_trusted ? (
                          <ShieldCheck className="w-3.5 h-3.5 text-success/60" />
                        ) : (
                          <ShieldAlert className="w-3.5 h-3.5 text-warn/60" />
                        )}
                      </span>
                    )}
                  </div>

                  {/* Meta row */}
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      {entry.username && (
                        <span className="text-[11px] text-txt-secondary font-medium truncate">
                          {entry.username}
                        </span>
                      )}
                      {date && (
                        <span className="text-[10px] text-txt-muted font-mono shrink-0">
                          {date} · {time}
                        </span>
                      )}
                    </div>

                    {/* Delete control */}
                    {canDelete && (
                      <div className="shrink-0">
                        {!isConfirming ? (
                          <button
                            onClick={() => setDeletingId(entry.id)}
                            className="opacity-0 group-hover/entry:opacity-100 p-1 rounded text-txt-muted hover:text-danger hover:bg-danger/10 transition-all"
                            title="Delete this change"
                          >
                            <Trash2 className="w-3 h-3" />
                          </button>
                        ) : (
                          <div className="flex items-center gap-1">
                            <span className="text-[10px] text-danger">Revert & delete?</span>
                            <button
                              onClick={() => handleDelete(entry.id)}
                              className="px-1.5 py-0.5 text-[10px] font-bold bg-danger text-white rounded hover:opacity-90 transition-opacity"
                            >
                              Yes
                            </button>
                            <button
                              onClick={() => setDeletingId(null)}
                              className="p-0.5 text-txt-muted hover:text-txt-primary"
                            >
                              <X className="w-3 h-3" />
                            </button>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}
