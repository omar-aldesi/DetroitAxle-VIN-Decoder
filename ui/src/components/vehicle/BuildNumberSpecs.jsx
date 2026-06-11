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
  Clock,
} from "lucide-react";
import { updateVehicle } from "../../api/vehicles";
import { getForkData } from "../../api/fork";
import { useToast } from "../../contexts/ToastContext";
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
      className="inline-flex items-center gap-1 text-[11px] font-semibold shrink-0 px-1.5 py-0.5 rounded-md"
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
  ranges,
  pending,
  proposed,
  serial,
  activeVin,
  pageVin,
  history,
  onSaved,
}) {
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const list = ranges ?? [];
  const pendingList = pending ?? [];
  const pendingCount = pendingList.length;
  const expandable = list.length > 0 || pendingCount > 0;

  const hasSerial = serial != null && activeVin.length === 17;

  // Resolve the value to show for the viewed VIN, strongest evidence first:
  //  1) a confirmed range covering this serial (the authoritative answer);
  //  2) a verified single sighting recorded for this exact serial, still waiting for a
  //     second matching VIN to confirm a range ("unconfirmed");
  //  3) an unverified edit saved for this serial, not yet in the engine ("pending review");
  //  4) build-key view (no serial): the lone range, or "varies" across several ranges.
  const resolvedVal = resolved?.Value ?? null;

  const pointForSerial = hasSerial
    ? pendingList.find((p) => Number(p.serial) === Number(serial))
    : null;

  // Latest unverified edit entered for this exact serial (from the fork endpoint's
  // `proposed` list — build-number history is not part of the vehicle payload).
  const unverifiedEdit = hasSerial
    ? (proposed ?? []).find((p) => Number(p.serial) === Number(serial))
    : null;

  const singleRange =
    !resolvedVal && !hasSerial && list.length === 1 ? list[0] : null;
  const variesNoSerial = !resolvedVal && !hasSerial && list.length > 1;

  // state: resolved | point | pending | single | none
  let value = null;
  let state = "none";
  let valueTier = null;
  if (resolvedVal) {
    value = resolvedVal;
    state = "resolved";
    valueTier = resolved.Confidence;
  } else if (pointForSerial) {
    value = pointForSerial.value;
    state = "point";
  } else if (unverifiedEdit) {
    value = unverifiedEdit.value;
    state = "pending";
  } else if (singleRange) {
    value = singleRange.Value;
    state = "single";
    // Confidence is serial-specific, and there's no serial in the build-key view. A manual
    // span is team-set across the whole group (serial-independent), so that label is safe;
    // a VIN-origin range stays neutral ("by build #") until a full VIN is entered.
    valueTier = singleRange.Origin === "manual" ? "manual" : null;
  }
  const display = value;

  const isManual = state === "resolved" && resolved.Origin === "manual";
  // A serial whose value is confirmed into a range or already recorded as a verified
  // sighting is locked from vehicle-page editing — correct it via the DNR page. An
  // unverified pending edit can still be amended before it's verified.
  const lockedConfirmed = state === "resolved" || state === "point";
  const canEdit = hasSerial && !isManual && !lockedConfirmed;

  const saveMut = useMutation({
    mutationFn: (v) => updateVehicle(activeVin, { [field.key]: v }),
    onSuccess: () => {
      toast(
        `${field.label} saved for this VIN — applies to the build-number range after an admin verifies it`,
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
      <div className="flex items-center gap-3 px-3 py-3">
        <span className="spec-label shrink-0 w-32">{field.label}</span>
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
              className="text-xs text-txt-muted italic shrink-0"
              title="This field changes across build numbers — enter a full VIN to see the exact value, or expand to view the ranges."
            >
              varies by build #
            </span>
          ) : value ? (
            <>
              <span
                className={`text-[15px] font-semibold text-txt-primary text-right truncate ${
                  field.mono ? "font-mono tracking-wider" : ""
                }`}
              >
                {value}
              </span>
              {(state === "resolved" || state === "single") && valueTier && (
                <ConfidenceTag tier={valueTier} />
              )}
              {state === "single" && !valueTier && (
                <span
                  title="From a build-number range — enter a full VIN above to see this value's confidence for that specific unit."
                  className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-txt-muted/10 border border-txt-muted/25 text-txt-secondary text-[11px] font-semibold shrink-0"
                >
                  <GitBranch className="w-3 h-3" />
                  By build #
                </span>
              )}
              {state === "point" && (
                <span
                  title="Recorded for this VIN — add a second matching VIN with the same value to confirm a build-number range."
                  className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-amber-500/10 border border-amber-500/25 text-amber-500 text-[11px] font-semibold shrink-0"
                >
                  <Info className="w-3 h-3" />
                  Unconfirmed
                </span>
              )}
              {state === "pending" && (
                <span
                  title="Saved for this VIN — applies to the build-number range once an admin verifies it."
                  className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-txt-muted/10 border border-txt-muted/25 text-txt-secondary text-[11px] font-semibold shrink-0"
                >
                  <Clock className="w-3 h-3" />
                  Pending review
                </span>
              )}
            </>
          ) : (
            <span className="text-sm text-txt-muted/30 font-mono select-none">
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
                <span className="text-[11px] text-txt-muted shrink-0">
                  {formatSpan(r.SerialStart, r.SerialEnd)}
                </span>
                <span
                  className={`text-[13px] font-medium text-txt-secondary ${
                    field.mono ? "font-mono tracking-wide" : ""
                  }`}
                >
                  {r.Value}
                </span>
                {r.Observations > 1 && (
                  <span className="ml-auto text-[10px] font-mono text-txt-muted/60 tabular-nums shrink-0">
                    &times;{r.Observations}
                  </span>
                )}
              </div>
            ))
          ) : (
            <p className="text-xs text-txt-muted/70 py-1 italic">
              No confirmed ranges.
            </p>
          )}
          {pendingCount > 0 && (
            <div className="flex flex-col gap-1 pt-0.5">
              <div className="flex items-center gap-1.5">
                <span
                  className="inline-flex items-center gap-1 text-[11px] font-medium px-1.5 py-0.5 rounded-md"
                  style={{
                    color: "#f59e0b",
                    background: "#f59e0b12",
                    border: "1px solid #f59e0b28",
                  }}
                >
                  <Info className="w-2.5 h-2.5 shrink-0" />
                  {pendingCount} unconfirmed
                </span>
                <span className="text-[11px] text-txt-muted">
                  awaiting a 2nd matching VIN to confirm a range
                </span>
              </div>
              {pendingList.map((p, i) => (
                <div key={i} className="flex items-center gap-2.5">
                  <span className="text-[11px] text-txt-muted shrink-0">
                    #{padSerial(p.serial)}
                  </span>
                  <span
                    className={`text-[13px] font-medium text-amber-500/90 ${
                      field.mono ? "font-mono tracking-wide" : ""
                    }`}
                  >
                    {p.value}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function BuildNumberSpecs({ vehicle, pageVin }) {
  const qc = useQueryClient();
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
  const proposed = data?.proposed ?? {};
  const serial = data?.serial;
  const hasSerial = serial != null;

  // A field carries something to show if it has a confirmed value, a verified sighting
  // awaiting a match, or an unverified edit pending review.
  const fieldHasData = (key) =>
    !!resolved[key]?.Value ||
    (fields[key]?.length ?? 0) > 0 ||
    (pending[key]?.length ?? 0) > 0 ||
    (proposed[key]?.length ?? 0) > 0;

  const resolvedCount = FORK_FIELDS.filter((f) => fieldHasData(f.key)).length;
  const anyData = FORK_FIELDS.some((f) => fieldHasData(f.key));
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
            <span className="ml-1.5 text-[11px] font-mono tabular-nums text-emerald-400/80">
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
                className={`w-full bg-bg-elevated border rounded-lg pl-8 pr-10 py-2 text-[13px] font-mono text-txt-primary placeholder:font-sans placeholder:text-txt-muted focus:outline-none transition-all ${
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
                  className="inline-flex items-center gap-1.5 text-[13px] font-mono px-2 py-1 rounded-md"
                  style={{
                    color: "#34d399",
                    background: "#34d39912",
                    border: "1px solid #34d39928",
                  }}
                >
                  <span
                    className="text-[10px] font-sans uppercase tracking-wider"
                    style={{ color: "#34d39980" }}
                  >
                    Build
                  </span>
                  #{padSerial(serial)}
                </span>
                <span className="text-[11px] text-txt-muted">
                  resolved for this VIN
                </span>
              </div>
            ) : (
              <p className="text-[11px] text-txt-muted flex items-center gap-1.5">
                <Info className="w-3 h-3 shrink-0 text-amber-500/70" />
                Type a full 17-char VIN — values update automatically for that
                build number.
              </p>
            )}
            {vinComplete && (
              <p className="text-[11px] text-amber-400/80 flex items-center gap-1.5">
                <Info className="w-3 h-3 shrink-0" />
                Values you enter are saved against this VIN and apply to the
                build-number range once an admin verifies them.
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
                    ranges={fields[f.key]}
                    pending={pending[f.key]}
                    proposed={proposed[f.key]}
                    serial={serial}
                    activeVin={activeVin}
                    pageVin={pageVin}
                    history={vehicle.history ?? []}
                    onSaved={refetchAfterEdit}
                  />
                ))}
              </div>
              {!anyData && (
                <div className="py-6 text-center">
                  <GitBranch className="w-6 h-6 text-txt-muted/25 mx-auto mb-2" />
                  <p className="text-[13px] text-txt-muted">
                    No build-number data yet
                  </p>
                  <p className="text-[11px] text-txt-muted/60 mt-0.5">
                    Add it from the DNR page.
                  </p>
                </div>
              )}
            </>
          )}

          {/* Legend */}
          <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1.5 pt-2.5 border-t border-border-subtle/40">
            {Object.entries(FORK_CONFIDENCE).map(([k, c]) => (
              <span
                key={k}
                className="inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-md"
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
