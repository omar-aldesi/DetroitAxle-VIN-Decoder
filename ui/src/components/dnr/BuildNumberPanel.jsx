import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  GitBranch,
  ChevronUp,
  ChevronDown,
  CheckCircle2,
  AlertTriangle,
  Info,
  Fingerprint,
  ArrowRight,
  Loader2,
  Plus,
} from "lucide-react";
import { useToast } from "../../contexts/ToastContext";
import { getForkData, recordForkPoint, recordForkRange } from "../../api/fork";
import { FORK_FIELDS, primaryKnownVin } from "./constants";

export function forkOutcomeMsg(o) {
  switch (o) {
    case "pending":
      return "Saved as a sighting — add one more matching VIN in the same span to confirm the range.";
    case "range_created":
      return "Range confirmed from matching VINs!";
    case "reinforced":
      return "Confirmed — strengthened the existing range.";
    case "forked":
      return "Exception confirmed — carved a new range inside the existing one.";
    case "manual_range":
      return "Range saved — applied immediately. Overlapping ranges were trimmed.";
    case "base_range":
      return "Base value saved — filled all build ranges that had no data yet.";
    default:
      return "Saved.";
  }
}

export function serialInRange(serial, start, end) {
  if (serial == null) return false;
  return serial >= start && (end == null || serial <= end);
}

// Human-friendly serial span. A base range [0, ∞] means "the whole group".
export function fmtSpan(start, end) {
  const pad = (n) => String(n).padStart(6, "0");
  const s = Number(start) || 0;
  if (s === 0 && end == null) return "All build numbers";
  if (s === 0) return `Up to #${pad(end)}`;
  if (end == null) return `#${pad(s)} onward`;
  return `#${pad(s)} → #${pad(end)}`;
}

export function ConfidencePip({ exact }) {
  return exact ? (
    <span className="text-[8px] font-bold text-emerald-400 bg-emerald-500/10 border border-emerald-500/25 rounded px-1 py-0.5 leading-none">
      EXACT
    </span>
  ) : (
    <span
      className="text-[9px] text-amber-400/70 font-mono leading-none"
      title="Approximate boundary"
    >
      ~
    </span>
  );
}

export function BuildNumberPanel({ vehicle, open, onToggle }) {
  const toast = useToast();
  const qc = useQueryClient();
  const [mode, setMode] = useState("vin"); // "vin" | "range"
  const [field, setField] = useState("brake_code");
  const [value, setValue] = useState("");
  const [vin, setVin] = useState(() => primaryKnownVin(vehicle));
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [allBuilds, setAllBuilds] = useState(false);
  const [overwrite, setOverwrite] = useState(false);

  useEffect(() => {
    setValue("");
    setStart("");
    setEnd("");
    setAllBuilds(false);
    setOverwrite(false);
    setVin(primaryKnownVin(vehicle));
    setField("brake_code");
    setMode("vin");
  }, [vehicle?.id]);

  const knownVins = vehicle.known_vins ?? [];
  const activeVin = vin.trim().toUpperCase();
  const lookup =
    activeVin.length === 17
      ? activeVin
      : primaryKnownVin(vehicle).length === 17
        ? primaryKnownVin(vehicle).trim().toUpperCase()
        : vehicle.build_key;
  const { data, isLoading } = useQuery({
    queryKey: ["fork", lookup],
    queryFn: () => getForkData(lookup).then((r) => r.data),
    enabled: open && !!lookup,
  });

  // New API shape: PascalCase range fields + resolved + serial at top level
  const ranges = data?.fields ?? {};
  const pending = data?.pending ?? {};
  const resolved = data?.resolved ?? {};
  const serial = data?.serial ?? null;

  const withData = FORK_FIELDS.filter(
    (f) => (ranges[f.key]?.length ?? 0) > 0,
  ).length;
  const resolvedKeys = Object.keys(resolved);
  const pendingTotal = Object.values(pending).reduce((a, v) => a + v.length, 0);

  const reset = () => {
    setValue("");
    setStart("");
    setEnd("");
  };
  const refetch = () => qc.invalidateQueries({ queryKey: ["fork"] });

  const pointMut = useMutation({
    mutationFn: () =>
      recordForkPoint(vin.trim().toUpperCase(), {
        field_key: field,
        value: value.trim(),
      }).then((r) => r.data),
    onSuccess: (d) => {
      toast(forkOutcomeMsg(d?.outcome), "success");
      reset();
      refetch();
    },
    onError: (e) => toast(e.response?.data?.error ?? "Could not save", "error"),
  });
  const rangeMut = useMutation({
    mutationFn: () =>
      recordForkRange(vehicle.build_key, {
        field_key: field,
        value: value.trim(),
        serial_start: allBuilds ? 0 : Number(start || 0),
        serial_end: allBuilds || end === "" ? null : Number(end),
        // "All builds" defaults to gap-fill so confirmed ranges survive;
        // overwrite is an explicit opt-in.
        base: allBuilds && !overwrite,
      }).then((r) => r.data),
    onSuccess: (d) => {
      toast(forkOutcomeMsg(d?.outcome), "success");
      reset();
      refetch();
    },
    onError: (e) => toast(e.response?.data?.error ?? "Could not save", "error"),
  });

  const busy = pointMut.isPending || rangeMut.isPending;
  const vinReady = mode === "vin" && vin.trim().length === 17 && value.trim();
  const rangeReady =
    mode === "range" && value.trim() && (allBuilds || start !== "");
  const ready = vinReady || rangeReady;
  const submit = () => {
    if (!ready || busy) return;
    mode === "vin" ? pointMut.mutate() : rangeMut.mutate();
  };

  const inputCls =
    "w-full text-sm px-3 py-2.5 rounded-xl border border-border-subtle bg-bg-elevated text-txt-primary placeholder:text-txt-muted/50 focus:outline-none focus:border-accent transition-all";

  return (
    <div className="border border-emerald-500/20 rounded-2xl overflow-hidden">
      {/* Header */}
      <button
        onClick={onToggle}
        className={`w-full flex items-center gap-3 px-4 py-3.5 transition-all ${open ? "bg-emerald-500/[0.06]" : "hover:bg-bg-elevated/30"}`}
      >
        <span className="w-2.5 h-2.5 rounded-full shrink-0 bg-emerald-500/70" />
        <GitBranch className="w-4 h-4 shrink-0 text-emerald-400" />
        <span className="text-sm font-bold text-txt-primary flex-1 text-left">
          Build-Number Data
        </span>
        {pendingTotal > 0 && (
          <span className="text-[9px] font-bold text-amber-400 bg-amber-500/10 border border-amber-500/20 rounded px-1.5 py-0.5 shrink-0">
            {pendingTotal} PENDING
          </span>
        )}
        <span className="text-[9px] font-semibold text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 rounded px-1.5 py-0.5 shrink-0">
          PER VIN
        </span>
        <span className="text-xs font-mono tabular-nums text-txt-muted shrink-0">
          {withData}/{FORK_FIELDS.length}
        </span>
        {open ? (
          <ChevronUp className="w-4 h-4 text-txt-muted shrink-0" />
        ) : (
          <ChevronDown className="w-4 h-4 text-txt-muted shrink-0" />
        )}
      </button>

      {open && (
        <div className="px-4 pb-4 pt-3 border-t border-border-subtle space-y-4">
          {/* Resolved for this VIN */}
          {!isLoading && resolvedKeys.length > 0 && (
            <div className="rounded-xl border border-emerald-500/25 overflow-hidden">
              <div className="flex items-center gap-2 px-3 py-2 bg-emerald-500/[0.08] border-b border-emerald-500/15">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                <span className="text-[11px] font-bold text-emerald-400 flex-1">
                  Resolved for this VIN
                </span>
                {serial != null && (
                  <span className="font-mono text-[10px] text-emerald-500/70 bg-emerald-500/10 rounded px-1.5 py-0.5 shrink-0">
                    #{serial.toLocaleString()}
                  </span>
                )}
              </div>
              <div className="p-2 grid grid-cols-2 gap-1">
                {FORK_FIELDS.map((f) => {
                  const r = resolved[f.key];
                  return (
                    <div
                      key={f.key}
                      className={`flex items-center gap-1.5 px-2 py-1.5 rounded-lg ${r ? "bg-emerald-500/5" : "bg-bg-elevated/30"}`}
                    >
                      <span className="text-[10px] text-txt-muted truncate flex-1 min-w-0">
                        {f.label}
                      </span>
                      {r ? (
                        <span className="font-mono text-[10px] font-bold text-emerald-300 shrink-0">
                          {r.Value}
                        </span>
                      ) : (
                        <span className="text-[10px] text-txt-muted/25 shrink-0">
                          —
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Pending-only state (no resolved yet) */}
          {!isLoading && resolvedKeys.length === 0 && pendingTotal > 0 && (
            <div className="flex items-start gap-2.5 bg-amber-500/8 border border-amber-500/20 rounded-xl px-3 py-2.5">
              <AlertTriangle className="w-3.5 h-3.5 text-amber-400 shrink-0 mt-0.5" />
              <div>
                <p className="text-[11px] font-bold text-amber-400">
                  Sighting recorded — not yet confirmed
                </p>
                <p className="text-[10px] text-txt-muted mt-0.5 leading-relaxed">
                  Add one more matching VIN to confirm a range for serial #
                  {serial?.toLocaleString()}.
                </p>
              </div>
            </div>
          )}

          {/* Explanation */}
          <div className="flex items-start gap-2 text-[11px] text-txt-secondary leading-relaxed">
            <Info className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
            <p>
              DNR writes go <strong className="text-txt-primary">directly</strong>{" "}
              into the build-number engine — no verification queue. Add a value
              for one VIN (two matching VINs confirm a range), or set a serial
              span manually. Agent edits on the vehicle page wait for admin
              verification before they affect ranges.
            </p>
          </div>

          {/* Mode tabs */}
          <div className="flex gap-1 p-1 bg-bg-elevated border border-border-subtle rounded-xl">
            {[
              { id: "vin", label: "Single VIN" },
              { id: "range", label: "Range / All" },
            ].map((t) => (
              <button
                key={t.id}
                onClick={() => setMode(t.id)}
                className={`flex-1 py-2 text-xs font-semibold rounded-lg transition-all ${mode === t.id ? "bg-bg-card text-txt-primary shadow-sm border border-border-subtle" : "text-txt-muted hover:text-txt-secondary"}`}
              >
                {t.label}
              </button>
            ))}
          </div>

          {/* Entry form */}
          <div className="space-y-2.5">
            <div className="grid grid-cols-2 gap-2.5">
              <div>
                <label className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                  Field
                </label>
                <select
                  value={field}
                  onChange={(e) => setField(e.target.value)}
                  className={`${inputCls} appearance-none cursor-pointer`}
                >
                  {FORK_FIELDS.map((f) => {
                    const hasRanges = (ranges[f.key]?.length ?? 0) > 0;
                    const hasPending = (pending[f.key]?.length ?? 0) > 0;
                    return (
                      <option key={f.key} value={f.key}>
                        {hasRanges ? "✓ " : hasPending ? "⏳ " : ""}
                        {f.label}
                      </option>
                    );
                  })}
                </select>
              </div>
              <div>
                <label className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                  Value
                </label>
                <input
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && submit()}
                  placeholder={
                    FORK_FIELDS.find((f) => f.key === field)?.placeholder ??
                    "Value"
                  }
                  className={inputCls}
                />
              </div>
            </div>

            {mode === "vin" ? (
              <div>
                <label className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                  VIN (full 17)
                </label>
                <div className="relative">
                  <Fingerprint className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-txt-muted pointer-events-none" />
                  <input
                    value={vin}
                    list={`known-vins-${vehicle.id}`}
                    onChange={(e) =>
                      setVin(
                        e.target.value
                          .replace(/[^a-zA-Z0-9]/g, "")
                          .toUpperCase()
                          .slice(0, 17),
                      )
                    }
                    onKeyDown={(e) => e.key === "Enter" && submit()}
                    placeholder="1G1AK55F177000001"
                    className={`${inputCls} pl-10 font-mono tracking-wider`}
                  />
                  {knownVins.length > 0 && (
                    <datalist id={`known-vins-${vehicle.id}`}>
                      {knownVins.map((v) => (
                        <option key={v} value={v} />
                      ))}
                    </datalist>
                  )}
                </div>
                <p className="text-[10px] text-txt-muted mt-1">
                  {vin.length}/17
                  {vin.length === 17 && (
                    <span className="text-success font-semibold"> ✓ ready</span>
                  )}
                  {knownVins.length > 0 && (
                    <span className="text-txt-muted/60">
                      {" "}
                      · {knownVins.length} unit
                      {knownVins.length !== 1 ? "s" : ""} on file
                    </span>
                  )}
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                <label className="flex items-center gap-2 text-xs text-txt-secondary cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={allBuilds}
                    onChange={(e) => setAllBuilds(e.target.checked)}
                    className="accent-emerald-500 w-3.5 h-3.5"
                  />
                  Applies to <strong className="text-txt-primary">all</strong>{" "}
                  build numbers in this group
                </label>
                {allBuilds && (
                  <div className="ml-5.5 pl-0.5 space-y-1">
                    <label className="flex items-center gap-2 text-xs text-txt-secondary cursor-pointer select-none">
                      <input
                        type="checkbox"
                        checked={overwrite}
                        onChange={(e) => setOverwrite(e.target.checked)}
                        className="accent-red-500 w-3.5 h-3.5"
                      />
                      Overwrite existing confirmed ranges
                    </label>
                    <p className="text-[10px] text-txt-muted leading-relaxed">
                      {overwrite
                        ? "⚠ Replaces ALL ranges for this field — confirmed VIN data will be lost."
                        : "Fills only build spans with no data yet — confirmed ranges are kept."}
                    </p>
                  </div>
                )}
                {!allBuilds && (
                  <div className="flex items-end gap-2">
                    <div className="flex-1">
                      <label className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                        From #
                      </label>
                      <input
                        value={start}
                        onChange={(e) =>
                          setStart(e.target.value.replace(/[^0-9]/g, ""))
                        }
                        placeholder="1"
                        className={`${inputCls} font-mono`}
                      />
                    </div>
                    <ArrowRight className="w-4 h-4 text-txt-muted mb-3 shrink-0" />
                    <div className="flex-1">
                      <label className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                        To #{" "}
                        <span className="text-txt-muted/50 font-normal normal-case">
                          (blank = end)
                        </span>
                      </label>
                      <input
                        value={end}
                        onChange={(e) =>
                          setEnd(e.target.value.replace(/[^0-9]/g, ""))
                        }
                        placeholder="end"
                        className={`${inputCls} font-mono`}
                      />
                    </div>
                  </div>
                )}
              </div>
            )}

            <button
              onClick={submit}
              disabled={!ready || busy}
              className="w-full flex items-center justify-center gap-2 py-2.5 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-bold rounded-xl transition-all"
            >
              {busy ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Plus className="w-4 h-4" />
              )}
              {mode === "vin" ? "Add for this VIN" : "Set range"}
            </button>
          </div>

          {/* All field ranges */}
          <div className="pt-1 space-y-2.5">
            <p className="text-[10px] font-bold text-txt-muted uppercase tracking-wider">
              All ranges for this build
            </p>
            {isLoading ? (
              <div className="flex items-center gap-2 text-xs text-txt-muted py-2">
                <Loader2 className="w-3.5 h-3.5 animate-spin" /> Loading…
              </div>
            ) : (
              <div className="space-y-2">
                {FORK_FIELDS.map((f) => {
                  const rs = ranges[f.key] ?? [];
                  const pend = pending[f.key] ?? [];
                  const hasAny = rs.length > 0 || pend.length > 0;

                  return (
                    <div
                      key={f.key}
                      className={`rounded-xl border overflow-hidden transition-all ${
                        hasAny
                          ? "border-border-subtle"
                          : "border-border-subtle/30"
                      }`}
                    >
                      {/* Field label row */}
                      <div
                        className={`flex items-center gap-2 px-3 py-2 ${
                          hasAny
                            ? "bg-bg-elevated/50 border-b border-border-subtle/40"
                            : "bg-bg-elevated/15"
                        }`}
                      >
                        <span
                          className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                            rs.length > 0
                              ? "bg-emerald-500"
                              : pend.length > 0
                                ? "bg-amber-400"
                                : "bg-border-subtle"
                          }`}
                        />
                        <span
                          className={`text-[11px] font-semibold ${hasAny ? "text-txt-secondary" : "text-txt-muted/40"}`}
                        >
                          {f.label}
                        </span>
                        {!hasAny && (
                          <span className="ml-auto text-[10px] text-txt-muted/25 italic">
                            no data
                          </span>
                        )}
                        {rs.length > 0 && (
                          <span className="ml-auto text-[9px] font-mono text-txt-muted">
                            {rs.length} range{rs.length !== 1 ? "s" : ""}
                          </span>
                        )}
                      </div>

                      {/* Range cards */}
                      {rs.length > 0 && (
                        <div className="p-2 space-y-1.5">
                          {rs.map((r, i) => {
                            const active = serialInRange(
                              serial,
                              r.SerialStart,
                              r.SerialEnd,
                            );
                            return (
                              <div
                                key={i}
                                className={`flex items-center gap-2 rounded-lg px-2.5 py-2 border transition-all ${
                                  active
                                    ? "bg-emerald-500/10 border-emerald-500/30"
                                    : r.Origin === "manual"
                                      ? "bg-accent/5 border-accent/15"
                                      : "bg-bg-elevated/60 border-border-subtle/50"
                                }`}
                              >
                                <span className="text-[10px] text-txt-muted shrink-0">
                                  {fmtSpan(r.SerialStart, r.SerialEnd)}
                                </span>
                                <span
                                  className={`text-[11px] font-bold ${active ? "text-emerald-300" : "text-txt-primary"}`}
                                >
                                  {r.Value}
                                </span>
                                <div className="ml-auto flex items-center gap-1.5 shrink-0">
                                  <ConfidencePip exact={r.BoundaryExact} />
                                  <span
                                    className="text-[9px] text-txt-muted"
                                    title={`${r.Observations} observation${r.Observations !== 1 ? "s" : ""}`}
                                  >
                                    {r.Observations}×
                                  </span>
                                  {active && (
                                    <span className="flex items-center gap-1 text-[9px] font-bold text-emerald-400 ml-0.5">
                                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" />
                                      this VIN
                                    </span>
                                  )}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}

                      {/* Pending serials */}
                      {pend.length > 0 && (
                        <div
                          className={`flex items-center gap-1.5 px-3 py-2 ${rs.length > 0 ? "border-t border-border-subtle/40" : ""}`}
                        >
                          <Info className="w-3 h-3 text-amber-400 shrink-0" />
                          <span className="text-[10px] text-amber-400/80">
                            {pend.length === 1
                              ? "1 pending"
                              : `${pend.length} pending`}
                            :{" "}
                            {pend
                              .map((s) => `#${s.toLocaleString()}`)
                              .join(", ")}
                          </span>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
