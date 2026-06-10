import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  GitBranch,
  ChevronUp,
  ChevronDown,
  Fingerprint,
  Info,
  Loader2,
  AlertCircle,
  Check,
  X,
  Edit3,
  ShieldCheck,
} from "lucide-react";
import { updateVehicle } from "../../api/vehicles";
import { getForkData } from "../../api/fork";
import { useToast } from "../../contexts/ToastContext";
import { useAuth } from "../../contexts/AuthContext";
import FieldHistoryIndicator from "../Fieldhistoryindicator.jsx";
import { FORK_FIELDS, FORK_CONFIDENCE } from "./constants";

export function padSerial(s) {
  return String(s ?? "").padStart(6, "0");
}

// Human-friendly serial span. A base range [0, ∞] means "applies to the whole group".
export function formatSpan(start, end) {
  const s = Number(start) || 0;
  if (s === 0 && end == null) return "All build numbers";
  if (s === 0) return `Up to #${padSerial(end)}`;
  if (end == null) return `#${padSerial(s)} onward`;
  return `#${padSerial(s)} → #${padSerial(end)}`;
}

export function ConfidenceTag({ tier }) {
  const c = FORK_CONFIDENCE[tier];
  if (!c) return null;
  return (
    <span
      className="inline-flex items-center gap-1 text-[10px] font-semibold shrink-0 px-1.5 py-0.5 rounded-md"
      style={{
        color: c.color,
        background: `${c.color}18`,
        border: `1px solid ${c.color}30`,
      }}
      title={c.desc}
    >
      <span
        className="w-1 h-1 rounded-full shrink-0"
        style={{ background: c.color }}
      />
      {c.label}
    </span>
  );
}

export function ForkFieldRow({
  field,
  resolved,
  columnVal,
  ranges,
  pending,
  activeVin,
  pageVin,
  history,
  feedsForkOnEdit,
  onSaved,
}) {
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const list = ranges ?? [];
  const pendingCount = pending?.length ?? 0;
  const expandable = list.length > 0 || pendingCount > 0;

  // 1) precise value for the active serial (full VIN);
  // 2) else the value of the single range that covers the whole group;
  // 3) else "varies" when there are several ranges but no serial to pick one;
  // 4) else the stored column value (genuinely unverified — no fork data exists).
  const resolvedVal = resolved?.Value ?? null;
  const singleRange = !resolvedVal && list.length === 1 ? list[0] : null;
  const variesNoSerial = !resolvedVal && list.length > 1;
  const fallback =
    !resolvedVal && list.length === 0 && columnVal ? String(columnVal) : null;

  const value = resolvedVal ?? singleRange?.Value ?? null;
  const valueTier = resolvedVal
    ? resolved.Confidence
    : singleRange
      ? singleRange.Origin === "manual"
        ? "manual"
        : "observed"
      : null;
  const display = value ?? fallback;

  const isVerified = history.some(
    (h) => h.field_name === field.key && h.is_verified === true,
  );
  const isTeamSet =
    resolved?.Confidence === "manual" || singleRange?.Origin === "manual";
  const canEdit =
    activeVin.length === 17 &&
    !isVerified &&
    (feedsForkOnEdit || !isTeamSet);

  const saveMut = useMutation({
    mutationFn: (v) => updateVehicle(activeVin, { [field.key]: v }),
    onSuccess: () => {
      toast(
        feedsForkOnEdit
          ? `${field.label} updated`
          : `${field.label} saved — build-number range applies after verification`,
        "success",
      );
      setEditing(false);
      onSaved?.();
    },
    onError: (e) =>
      toast(e.response?.data?.error ?? "Could not save", "error"),
  });

  const openEdit = () => {
    setDraft(display ?? "");
    setEditing(true);
  };
  const save = () => {
    if (!draft.trim()) return;
    if (draft.trim() === (display ?? "")) {
      setEditing(false);
      return;
    }
    saveMut.mutate(draft.trim());
  };

  return (
    <div className="border-b border-border-subtle/30 last:border-0 group">
      {/* Main row */}
      <div className="flex items-center gap-3 px-3 py-2.5">
        <span className="spec-label shrink-0 w-28">{field.label}</span>
        <div className="flex-1 min-w-0 flex items-center justify-end gap-2">
          {editing ? (
            <>
              <input
                autoFocus
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") save();
                  if (e.key === "Escape") setEditing(false);
                }}
                className={`bg-bg-elevated border border-accent/40 rounded-lg px-2.5 py-1 text-sm text-txt-primary w-32 focus:outline-none focus:border-accent transition-all text-right ${
                  field.mono ? "font-mono tracking-wider" : ""
                }`}
              />
              <button
                onClick={save}
                disabled={saveMut.isPending || !draft.trim()}
                className="text-success hover:opacity-75 transition-opacity"
              >
                {saveMut.isPending ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Check className="w-3.5 h-3.5" />
                )}
              </button>
              <button
                onClick={() => setEditing(false)}
                className="text-txt-muted hover:text-txt-secondary transition-colors"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </>
          ) : variesNoSerial ? (
            <span
              className="text-[11px] text-txt-muted/70 italic shrink-0"
              title="This field changes across build numbers — enter a full VIN to see the exact value, or expand to view the ranges."
            >
              varies by build #
            </span>
          ) : value ? (
            <>
              <span
                className={`text-sm font-semibold text-txt-primary text-right truncate ${
                  field.mono ? "font-mono tracking-wider" : ""
                }`}
              >
                {value}
              </span>
              <ConfidenceTag tier={valueTier} />
            </>
          ) : fallback ? (
            <>
              <span
                className={`text-sm font-medium text-txt-secondary text-right truncate ${
                  field.mono ? "font-mono tracking-wider" : ""
                }`}
              >
                {fallback}
              </span>
              <span
                className="text-[10px] font-medium text-txt-muted/50 shrink-0 italic"
                title="Stored value — not yet confirmed as a build-number range"
              >
                unverified
              </span>
            </>
          ) : (
            <span className="text-xs text-txt-muted/25 font-mono select-none">
              —
            </span>
          )}
          {!editing && (
            <FieldHistoryIndicator
              fieldName={field.key}
              history={history}
              vin={pageVin}
              placement="above"
              align="end"
            />
          )}
          {!editing && isVerified && (
            <span
              title="This value has been verified and locked"
              className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-success/10 border border-success/25 text-success text-[10px] font-semibold shrink-0"
            >
              <ShieldCheck className="w-3 h-3" />
              Verified
            </span>
          )}
          {!editing && isTeamSet && !feedsForkOnEdit && (
            <span
              title="Set by the DNR team — agents cannot override"
              className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-sky-500/10 border border-sky-500/25 text-sky-500 text-[10px] font-semibold shrink-0"
            >
              <ShieldCheck className="w-3 h-3" />
              Locked
            </span>
          )}
          {!editing && canEdit && (
            <button
              onClick={openEdit}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-txt-muted hover:text-accent shrink-0"
              title={`Edit ${field.label} for this VIN`}
            >
              <Edit3 className="w-3 h-3" />
            </button>
          )}
          {!editing &&
            (expandable ? (
              <button
                onClick={() => setOpen((v) => !v)}
                className="flex items-center gap-1 ml-1 text-txt-muted hover:text-txt-primary transition-colors shrink-0"
                title={`${list.length} range${list.length !== 1 ? "s" : ""}`}
              >
                {list.length > 1 && (
                  <span className="text-[9px] font-mono tabular-nums text-txt-muted/40">
                    {list.length}
                  </span>
                )}
                {open ? (
                  <ChevronUp className="w-3.5 h-3.5" />
                ) : (
                  <ChevronDown className="w-3.5 h-3.5" />
                )}
              </button>
            ) : (
              <span className="w-3.5 shrink-0" />
            ))}
        </div>
      </div>

      {/* Expanded ranges — indented under the label column */}
      {open && (
        <div className="mb-2.5 ml-[calc(7rem+0.75rem)] pl-3 border-l-2 border-border-subtle/60 space-y-1.5 animate-fade-in">
          {list.length > 0 ? (
            list.map((r, i) => (
              <div key={i} className="flex items-center gap-2.5 py-0.5">
                <span className="text-[10px] text-txt-muted/70 shrink-0">
                  {formatSpan(r.SerialStart, r.SerialEnd)}
                </span>
                <span
                  className={`text-xs font-medium text-txt-secondary ${
                    field.mono ? "font-mono tracking-wide" : ""
                  }`}
                >
                  {r.Value}
                </span>
                {r.Observations > 1 && (
                  <span className="ml-auto text-[9px] font-mono text-txt-muted/35 tabular-nums shrink-0">
                    &times;{r.Observations}
                  </span>
                )}
              </div>
            ))
          ) : (
            <p className="text-[11px] text-txt-muted/40 py-1 italic">
              No confirmed ranges.
            </p>
          )}
          {pendingCount > 0 && (
            <div className="flex items-center gap-1.5 pt-0.5">
              <span
                className="inline-flex items-center gap-1 text-[10px] font-medium px-1.5 py-0.5 rounded-md"
                style={{
                  color: "#f59e0b",
                  background: "#f59e0b12",
                  border: "1px solid #f59e0b28",
                }}
              >
                <Info className="w-2.5 h-2.5 shrink-0" />
                {pendingCount} pending
              </span>
              <span className="text-[10px] text-txt-muted/40">
                needs a matching VIN
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function BuildNumberSpecs({ vehicle, pageVin }) {
  const { user } = useAuth();
  const qc = useQueryClient();
  const feedsForkOnEdit = user?.isDNR || user?.isAdmin;
  const [open, setOpen] = useState(true);
  const [checkVin, setCheckVin] = useState(() =>
    (pageVin ?? "").trim().toUpperCase(),
  );

  useEffect(() => {
    const v = (pageVin ?? "").trim().toUpperCase();
    if (v.length === 17) setCheckVin(v);
  }, [pageVin]);

  const trimmed = checkVin.trim().toUpperCase();
  const lookupKey =
    trimmed.length === 17 || trimmed.length === 10
      ? trimmed
      : vehicle.build_key;
  const activeVin = trimmed.length === 17 ? trimmed : "";
  const { data, isLoading, isError } = useQuery({
    queryKey: ["fork", lookupKey],
    queryFn: () => getForkData(lookupKey).then((r) => r.data),
    enabled: open && !!lookupKey,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const refetchAfterEdit = () => {
    qc.invalidateQueries({ queryKey: ["fork", lookupKey] });
    qc.invalidateQueries({ queryKey: ["vehicle", pageVin] });
  };

  const resolved = data?.resolved ?? {};
  const fields = data?.fields ?? {};
  const pending = data?.pending ?? {};
  const serial = data?.serial;
  const hasSerial = serial != null;

  const resolvedCount = FORK_FIELDS.filter(
    (f) => resolved[f.key]?.Value || (fields[f.key]?.length ?? 0) > 0,
  ).length;
  const anyData = FORK_FIELDS.some(
    (f) =>
      resolved[f.key]?.Value || vehicle[f.key] || (fields[f.key]?.length ?? 0),
  );
  const vinLen = checkVin.length;
  const vinComplete = vinLen === 17;

  return (
    <div className="section-card animate-fade-in">
      {/* Header */}
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between"
      >
        <span className="section-title">
          <GitBranch className="w-4 h-4 text-emerald-400" />
          Build-Number Specs
          {resolvedCount > 0 && (
            <span className="ml-1.5 text-[10px] font-mono tabular-nums text-emerald-400/70">
              {resolvedCount}/{FORK_FIELDS.length}
            </span>
          )}
        </span>
        {open ? (
          <ChevronUp className="w-4 h-4 text-txt-muted" />
        ) : (
          <ChevronDown className="w-4 h-4 text-txt-muted" />
        )}
      </button>

      {open && (
        <div className="mt-3 space-y-3">
          {/* VIN input + serial context */}
          <div className="space-y-2">
            <div className="relative">
              <Fingerprint className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-txt-muted pointer-events-none" />
              <input
                value={checkVin}
                onChange={(e) =>
                  setCheckVin(
                    e.target.value
                      .replace(/[^a-zA-Z0-9]/g, "")
                      .toUpperCase()
                      .slice(0, 17),
                  )
                }
                placeholder="VIN for this unit (auto-resolves at 17 chars)…"
                className={`w-full bg-bg-elevated border rounded-lg pl-8 pr-10 py-1.5 text-xs font-mono text-txt-primary placeholder:font-sans placeholder:text-txt-muted focus:outline-none transition-all ${
                  vinComplete
                    ? "border-emerald-500/40 focus:border-emerald-500/60"
                    : "border-border-subtle focus:border-accent/60"
                }`}
              />
              {vinLen > 0 && (
                <span
                  className={`absolute right-2.5 top-1/2 -translate-y-1/2 text-[9px] font-mono tabular-nums transition-colors pointer-events-none ${
                    vinComplete ? "text-emerald-400/80" : "text-txt-muted/35"
                  }`}
                >
                  {vinLen}/17
                </span>
              )}
            </div>

            {/* Serial badge or hint */}
            {hasSerial ? (
              <div className="flex items-center gap-2">
                <span
                  className="inline-flex items-center gap-1.5 text-xs font-mono px-2 py-1 rounded-md"
                  style={{
                    color: "#34d399",
                    background: "#34d39912",
                    border: "1px solid #34d39928",
                  }}
                >
                  <span
                    className="text-[9px] font-sans uppercase tracking-wider"
                    style={{ color: "#34d39980" }}
                  >
                    Build
                  </span>
                  #{padSerial(serial)}
                </span>
                <span className="text-[10px] text-txt-muted/50">
                  resolved for this VIN
                </span>
              </div>
            ) : (
              <p className="text-[10px] text-txt-muted/50 flex items-center gap-1">
                <Info className="w-3 h-3 shrink-0 text-amber-500/60" />
                Type a full 17-char VIN — values update automatically for that
                build number.
              </p>
            )}
            {vinComplete && !feedsForkOnEdit && (
              <p className="text-[10px] text-amber-400/70 flex items-center gap-1">
                <Info className="w-3 h-3 shrink-0" />
                Edits here update the spec immediately; the build-number range
                applies after admin verification.
              </p>
            )}
          </div>

          {/* Field rows */}
          {isLoading ? (
            <div className="flex items-center justify-center py-6 gap-2 text-txt-muted text-xs">
              <Loader2 className="w-4 h-4 animate-spin text-emerald-400" />
              Loading…
            </div>
          ) : isError ? (
            <div className="flex items-center gap-2 py-3 text-xs text-txt-muted/60">
              <AlertCircle className="w-4 h-4 shrink-0 text-danger/50" />
              Couldn't load build-number data.
            </div>
          ) : (
            <>
              <div className="rounded-xl border border-border-subtle/50">
                {FORK_FIELDS.map((f) => (
                  <ForkFieldRow
                    key={f.key}
                    field={f}
                    resolved={resolved[f.key]}
                    columnVal={vehicle[f.key]}
                    ranges={fields[f.key]}
                    pending={pending[f.key]}
                    activeVin={activeVin}
                    pageVin={pageVin}
                    history={vehicle.history ?? []}
                    feedsForkOnEdit={feedsForkOnEdit}
                    onSaved={refetchAfterEdit}
                  />
                ))}
              </div>
              {!anyData && (
                <div className="py-6 text-center">
                  <GitBranch className="w-6 h-6 text-txt-muted/15 mx-auto mb-2" />
                  <p className="text-xs text-txt-muted/40">
                    No build-number data yet
                  </p>
                  <p className="text-[10px] text-txt-muted/25 mt-0.5">
                    Add it from the DNR page.
                  </p>
                </div>
              )}
            </>
          )}

          {/* Legend */}
          <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1.5 pt-2 border-t border-border-subtle/30">
            {Object.entries(FORK_CONFIDENCE).map(([k, c]) => (
              <span
                key={k}
                className="inline-flex items-center gap-1 text-[10px] font-medium px-1.5 py-0.5 rounded-md"
                style={{
                  color: c.color,
                  background: `${c.color}12`,
                  border: `1px solid ${c.color}25`,
                }}
                title={c.desc}
              >
                <span
                  className="w-1 h-1 rounded-full shrink-0"
                  style={{ background: c.color }}
                />
                {c.label}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
