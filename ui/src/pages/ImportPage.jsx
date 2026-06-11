import { useState, useMemo, useRef, useEffect, useCallback } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Upload, FileText, X, Play, Ban, RotateCcw, Download,
  CheckCircle2, XCircle, SkipForward, AlertTriangle, Loader2,
  Zap, Clock, Database, ChevronDown, ChevronUp, Copy,
} from "lucide-react";
import Navbar from "../components/NavBar";
import { useAuth } from "../contexts/AuthContext";
import { useToast } from "../contexts/ToastContext";
import { startImport, getImportJob, cancelImport } from "../api/import";

/* ─── VIN validation (mirrors backend check-digit algorithm) ─────────── */
const TRANS = {
  A:1,B:2,C:3,D:4,E:5,F:6,G:7,H:8,
  J:1,K:2,L:3,M:4,N:5,P:7,R:9,
  S:2,T:3,U:4,V:5,W:6,X:7,Y:8,Z:9,
  0:0,1:1,2:2,3:3,4:4,5:5,6:6,7:7,8:8,9:9,
};
const WEIGHTS = [8,7,6,5,4,3,2,10,0,9,8,7,6,5,4,3,2];

function validateVIN(vin) {
  if (!vin || vin.length !== 17) return false;
  let sum = 0;
  for (let i = 0; i < 17; i++) {
    const v = TRANS[vin[i]];
    if (v === undefined) return false;
    sum += v * WEIGHTS[i];
  }
  const rem = sum % 11;
  const check = vin[8];
  return rem === 10 ? check === "X" : check === String(rem);
}

function isGMVIN(vin) {
  if (!vin || vin.length < 3) return false;
  const p2 = vin.slice(0, 2);
  if (p2 === "1G" || p2 === "2G" || p2 === "3G") return true;
  const p3 = vin.slice(0, 3);
  return p3 === "KL4" || p3 === "KL8" || p3 === "KL1" || p3 === "W0L";
}

function parseVINs(raw) {
  const tokens = raw.split(/[\s,;\t\r\n]+/).map((t) => t.trim().toUpperCase()).filter(Boolean);
  const seen = new Set();
  return tokens.map((vin) => {
    const dup = seen.has(vin);
    seen.add(vin);
    return { vin, valid: validateVIN(vin), gm: isGMVIN(vin), duplicate: dup };
  });
}

/* ─── Helpers ──────────────────────────────────────────────────────────── */
function fmtDuration(ms) {
  if (!ms) return "—";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function fmtElapsed(startedAt, completedAt) {
  const end = completedAt ? new Date(completedAt) : new Date();
  const ms = end - new Date(startedAt);
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s`;
  return `${Math.floor(s / 60)}m ${s % 60}s`;
}

function fmtEstimate(valid, concurrency) {
  const lo = Math.ceil(valid / concurrency * 2);
  const hi = Math.ceil(valid / concurrency * 5);
  const fmt = (s) => s >= 60 ? `${Math.round(s / 60)}m` : `${s}s`;
  return `~${fmt(lo)} – ${fmt(hi)}`;
}

/* ─── Status config ────────────────────────────────────────────────────── */
const STATUS_META = {
  success: { label: "Imported", Icon: CheckCircle2, color: "text-emerald-400", bg: "bg-emerald-500/10 border-emerald-500/25" },
  skipped: { label: "Skipped",  Icon: SkipForward,  color: "text-blue-400",    bg: "bg-blue-500/10 border-blue-500/25" },
  failed:  { label: "Failed",   Icon: XCircle,      color: "text-red-400",     bg: "bg-red-500/10 border-red-500/25" },
  invalid: { label: "Invalid",  Icon: AlertTriangle, color: "text-amber-400",  bg: "bg-amber-500/10 border-amber-500/25" },
};

function StatusBadge({ status }) {
  const m = STATUS_META[status] ?? STATUS_META.failed;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md border text-[10px] font-bold shrink-0 ${m.color} ${m.bg}`}>
      <m.Icon className="w-2.5 h-2.5" />{m.label}
    </span>
  );
}

/* ─── Concurrency picker ───────────────────────────────────────────────── */
const CONCURRENCY_OPTIONS = [
  { value: 1,  hint: "safest" },
  { value: 3,  hint: "recommended" },
  { value: 5,  hint: "faster" },
  { value: 10, hint: "fastest" },
];

function ConcurrencyPicker({ value, onChange }) {
  return (
    <div className="flex gap-1.5">
      {CONCURRENCY_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          title={opt.hint}
          className={`flex-1 py-2 rounded-lg border text-xs font-bold transition-all ${
            value === opt.value
              ? "bg-accent text-white border-accent shadow-glow-sm"
              : "bg-bg-elevated border-border-subtle text-txt-muted hover:text-txt-secondary hover:border-border"
          }`}
        >
          {opt.value}
        </button>
      ))}
    </div>
  );
}

/* ─── Toggle ───────────────────────────────────────────────────────────── */
function Toggle({ checked, onChange, label, sub }) {
  return (
    <label className="flex items-start gap-3 cursor-pointer group">
      <button
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative mt-0.5 w-9 h-5 rounded-full shrink-0 transition-colors focus:outline-none ${
          checked ? "bg-accent" : "bg-bg-elevated border border-border"
        }`}
      >
        <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${checked ? "translate-x-4" : ""}`} />
      </button>
      <div>
        <p className="text-xs font-semibold text-txt-primary leading-tight group-hover:text-accent transition-colors">{label}</p>
        {sub && <p className="text-[11px] text-txt-muted mt-0.5 leading-relaxed">{sub}</p>}
      </div>
    </label>
  );
}

/* ═══════════════════════════════════════════════════════
   Setup panel
═══════════════════════════════════════════════════════ */
function SetupPanel({ onJobStarted }) {
  const toast = useToast();
  const [raw, setRaw] = useState("");
  const [concurrency, setConcurrency] = useState(3);
  const [skipExisting, setSkipExisting] = useState(true);
  const [showInvalid, setShowInvalid] = useState(false);
  const fileRef = useRef(null);
  const textareaRef = useRef(null);

  const parsed = useMemo(() => parseVINs(raw), [raw]);

  const stats = useMemo(() => {
    const unique = parsed.filter((v) => !v.duplicate);
    return {
      total:      unique.length,
      valid:      unique.filter((v) =>  v.valid).length,
      invalid:    unique.filter((v) => !v.valid).length,
      gm:         unique.filter((v) =>  v.valid && v.gm).length,
      duplicates: parsed.filter((v) =>  v.duplicate).length,
    };
  }, [parsed]);

  const vinsToSubmit = useMemo(
    () => parsed.filter((v) => v.valid && !v.duplicate).map((v) => v.vin),
    [parsed],
  );

  const invalidEntries = useMemo(
    () => parsed.filter((v) => !v.valid && !v.duplicate),
    [parsed],
  );

  const handleFile = useCallback((e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const content = ev.target.result ?? "";
      setRaw((prev) => (prev ? prev + "\n" + content : content));
    };
    reader.readAsText(file);
    e.target.value = "";
  }, []);

  const handlePaste = useCallback(async () => {
    try {
      const text = await navigator.clipboard.readText();
      setRaw((prev) => (prev ? prev + "\n" + text : text));
      textareaRef.current?.focus();
    } catch {
      textareaRef.current?.focus();
    }
  }, []);

  const mutation = useMutation({
    mutationFn: () =>
      startImport({ vins: vinsToSubmit, concurrency, skip_existing: skipExisting }).then((r) => r.data),
    onSuccess: (data) => onJobStarted(data.job_id),
    onError: (err) => toast(err.response?.data?.error ?? "Failed to start import", "error"),
  });

  useEffect(() => {
    const h = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter" && stats.valid > 0 && !mutation.isPending) {
        mutation.mutate();
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [stats.valid, mutation]);

  const canStart = stats.valid > 0 && !mutation.isPending;

  return (
    <div className="max-w-5xl mx-auto px-4 sm:px-6 py-8">
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_300px] gap-5 items-start">

        {/* ── VIN input card ─────────────────────────────────── */}
        <div className="bg-bg-card border border-border-subtle rounded-2xl overflow-hidden flex flex-col">

          {/* Card header */}
          <div className="flex items-center justify-between px-5 py-3.5 border-b border-border-subtle">
            <div className="flex items-center gap-2">
              <FileText className="w-4 h-4 text-accent" />
              <span className="text-sm font-bold text-txt-primary">VIN List</span>
              {stats.total > 0 && (
                <span className="text-xs text-txt-muted tabular-nums">
                  · {stats.total.toLocaleString()} total
                </span>
              )}
            </div>
            <div className="flex items-center gap-1.5">
              <button
                onClick={handlePaste}
                className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium text-txt-muted hover:text-txt-primary bg-bg-elevated border border-border-subtle rounded-lg transition-all"
              >
                <Copy className="w-3 h-3" /> Paste
              </button>
              <button
                onClick={() => fileRef.current?.click()}
                className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium text-txt-muted hover:text-txt-primary bg-bg-elevated border border-border-subtle rounded-lg transition-all"
              >
                <Upload className="w-3 h-3" /> Load file
              </button>
              {raw && (
                <button
                  onClick={() => setRaw("")}
                  title="Clear"
                  className="p-1.5 text-txt-muted hover:text-danger rounded-lg hover:bg-danger/10 transition-all"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>

          <input ref={fileRef} type="file" accept=".txt,.csv" className="hidden" onChange={handleFile} />

          {/* Textarea */}
          <textarea
            ref={textareaRef}
            value={raw}
            onChange={(e) => setRaw(e.target.value)}
            placeholder={"Paste VINs here — one per line, or comma / space / semicolon separated\n\n1GCUKREC9EG435220\n2GNFLNEK5E6229516\n..."}
            spellCheck={false}
            className="flex-1 min-h-[280px] resize-none w-full px-5 py-4 bg-transparent text-sm font-mono text-txt-primary placeholder:text-txt-muted/35 focus:outline-none leading-relaxed"
          />

          {/* Stats footer */}
          {stats.total > 0 && (
            <div className="border-t border-border-subtle">
              <div className="flex items-center gap-4 px-5 py-3 flex-wrap text-xs">
                <span className="flex items-center gap-1.5 font-semibold text-emerald-400">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  {stats.valid.toLocaleString()} valid
                </span>
                {stats.gm > 0 && (
                  <span className="flex items-center gap-1.5 font-semibold text-accent">
                    <Zap className="w-3.5 h-3.5" />
                    {stats.gm} GM
                  </span>
                )}
                {stats.invalid > 0 && (
                  <button
                    onClick={() => setShowInvalid((v) => !v)}
                    className="flex items-center gap-1.5 font-semibold text-amber-400 hover:opacity-80 transition-opacity"
                  >
                    <AlertTriangle className="w-3.5 h-3.5" />
                    {stats.invalid} invalid
                    {showInvalid ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                  </button>
                )}
                {stats.duplicates > 0 && (
                  <span className="text-txt-muted">{stats.duplicates} dupes removed</span>
                )}
              </div>

              {/* Invalid VIN list */}
              {showInvalid && invalidEntries.length > 0 && (
                <div className="border-t border-border-subtle px-5 py-3 bg-amber-500/5 max-h-32 overflow-y-auto">
                  <p className="text-[10px] font-bold text-amber-400 uppercase tracking-wider mb-2">
                    Invalid — will be skipped
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {invalidEntries.slice(0, 60).map((e) => (
                      <span
                        key={e.vin}
                        className="font-mono text-[10px] text-amber-300/70 bg-amber-500/10 border border-amber-500/20 rounded px-1.5 py-0.5"
                      >
                        {e.vin}
                      </span>
                    ))}
                    {invalidEntries.length > 60 && (
                      <span className="text-[10px] text-txt-muted self-center">
                        +{invalidEntries.length - 60} more
                      </span>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Settings card ──────────────────────────────────── */}
        <div className="bg-bg-card border border-border-subtle rounded-2xl p-5 flex flex-col gap-6">

          {/* Concurrency */}
          <div>
            <div className="flex items-center justify-between mb-2.5">
              <p className="text-xs font-bold text-txt-muted uppercase tracking-wider">Parallel Decodes</p>
              <span className="text-xs font-bold text-txt-primary tabular-nums">{concurrency}</span>
            </div>
            <ConcurrencyPicker value={concurrency} onChange={setConcurrency} />
            <p className="text-[11px] text-txt-muted mt-2.5 leading-relaxed">
              Each VIN calls 2–3 external APIs.{" "}
              <span className="font-semibold text-txt-secondary">3 is safe for most plans.</span>
            </p>
          </div>

          {/* Skip existing toggle */}
          <Toggle
            checked={skipExisting}
            onChange={setSkipExisting}
            label="Skip existing VINs"
            sub="Already-decoded VINs won't be re-fetched. Saves API quota."
          />

          {/* Divider + actions */}
          <div className="mt-auto border-t border-border-subtle pt-5 space-y-3">

            {/* Time estimate */}
            {stats.valid > 0 && (
              <div className="flex items-center justify-between text-xs">
                <span className="text-txt-muted flex items-center gap-1.5">
                  <Clock className="w-3.5 h-3.5" /> Estimate
                </span>
                <span className="font-semibold text-txt-secondary tabular-nums">
                  {fmtEstimate(stats.valid, concurrency)}
                </span>
              </div>
            )}

            {/* Start button */}
            <button
              disabled={!canStart}
              onClick={() => mutation.mutate()}
              className="w-full flex items-center justify-center gap-2.5 py-3 bg-accent hover:bg-accent-hover disabled:opacity-35 disabled:cursor-not-allowed text-white font-bold rounded-xl text-sm transition-all shadow-glow-sm"
            >
              {mutation.isPending ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Play className="w-4 h-4" />
              )}
              <span>
                {mutation.isPending ? "Starting…" : "Start Import"}
                {stats.valid > 0 && !mutation.isPending && (
                  <span className="font-normal opacity-70 ml-1.5">({stats.valid.toLocaleString()})</span>
                )}
              </span>
              <kbd className="ml-auto text-white/30 text-[10px] font-mono hidden sm:block">⌘↵</kbd>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Result row
═══════════════════════════════════════════════════════ */
function ResultRow({ r }) {
  const vehicle = r.year && r.make ? `${r.year} ${r.make}${r.model ? " " + r.model : ""}` : null;
  return (
    <div className="grid grid-cols-[1fr_auto_auto] sm:grid-cols-[190px_auto_1fr_auto] items-center gap-x-3 gap-y-0.5 px-5 py-2.5 border-b border-border-subtle/40 last:border-0 hover:bg-bg-elevated/40 transition-colors">
      {/* VIN */}
      <span className="font-mono text-[11px] text-txt-secondary tracking-wider truncate" title={r.vin}>
        {r.vin}
      </span>
      {/* Status */}
      <StatusBadge status={r.status} />
      {/* Vehicle / error */}
      <span className="hidden sm:block text-xs text-txt-secondary truncate min-w-0">
        {vehicle ?? (r.error ? <span className="text-red-400/80">{r.error}</span> : "—")}
        {r.is_gm && (
          <span className="ml-2 inline-flex items-center gap-0.5 text-[9px] font-bold text-accent align-middle">
            <Zap className="w-2.5 h-2.5" />GM
          </span>
        )}
      </span>
      {/* Duration */}
      <span className="text-[11px] text-txt-muted tabular-nums text-right shrink-0">
        {fmtDuration(r.duration_ms)}
      </span>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Progress panel
═══════════════════════════════════════════════════════ */
function ProgressPanel({ job, jobId, onReset }) {
  const toast = useToast();
  const [filter, setFilter] = useState("all");

  const running    = job?.status === "running";
  const completed  = job?.status === "completed";
  const pct        = job ? Math.round((job.processed / Math.max(job.total, 1)) * 100) : 0;
  const elapsed    = job ? fmtElapsed(job.started_at, job.completed_at) : "—";

  const cancelMutation = useMutation({
    mutationFn: () => cancelImport(jobId),
    onSuccess: () => toast("Import cancelled", "info"),
    onError:   () => toast("Could not cancel", "error"),
  });

  const filteredResults = useMemo(() => {
    if (!job?.results) return [];
    const all = [...job.results].reverse();
    return filter === "all" ? all : all.filter((r) => r.status === filter);
  }, [job?.results, filter]);

  const exportFailed = useCallback(() => {
    const failed = job?.results?.filter((r) => r.status === "failed") ?? [];
    if (!failed.length) return;
    const blob = new Blob([failed.map((r) => r.vin).join("\n")], { type: "text/plain" });
    const url  = URL.createObjectURL(blob);
    const a    = document.createElement("a");
    a.href     = url;
    a.download = `import_failed_${jobId.slice(0, 8)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }, [job?.results, jobId]);

  if (!job) {
    return (
      <div className="flex items-center justify-center py-32">
        <Loader2 className="w-6 h-6 animate-spin text-accent" />
      </div>
    );
  }

  const FILTERS = [
    { key: "all",     label: "All",      count: job.total },
    { key: "success", label: "Imported", count: job.succeeded },
    { key: "failed",  label: "Failed",   count: job.failed },
    { key: "skipped", label: "Skipped",  count: job.skipped },
    { key: "invalid", label: "Invalid",  count: job.invalid },
  ];

  const STATS = [
    { key: "succeeded", label: "Imported", value: job.succeeded, color: "text-emerald-400", bg: "bg-emerald-500/10 border-emerald-500/20" },
    { key: "skipped",   label: "Skipped",  value: job.skipped,   color: "text-blue-400",    bg: "bg-blue-500/10 border-blue-500/20" },
    { key: "failed",    label: "Failed",   value: job.failed,    color: "text-red-400",     bg: "bg-red-500/10 border-red-500/20" },
    { key: "invalid",   label: "Invalid",  value: job.invalid,   color: "text-amber-400",   bg: "bg-amber-500/10 border-amber-500/20" },
  ];

  return (
    <div className="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-5">

      {/* ── Job overview card ── */}
      <div className="bg-bg-card border border-border-subtle rounded-2xl p-6 space-y-5">

        {/* Top row: status + meta + actions */}
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            {/* Status pill */}
            <div className="flex items-center gap-2 mb-1">
              {running ? (
                <span className="flex items-center gap-1.5 text-xs font-bold text-accent">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" /> Running
                </span>
              ) : completed ? (
                <span className="flex items-center gap-1.5 text-xs font-bold text-emerald-400">
                  <CheckCircle2 className="w-3.5 h-3.5" /> Complete
                </span>
              ) : (
                <span className="flex items-center gap-1.5 text-xs font-bold text-red-400">
                  <Ban className="w-3.5 h-3.5" /> Cancelled
                </span>
              )}
              <span className="text-xs text-txt-muted">· {job.created_by_name}</span>
            </div>
            <p className="text-lg font-extrabold text-txt-primary leading-tight">
              {job.total.toLocaleString()} VINs
            </p>
            <p className="text-xs text-txt-muted font-mono mt-0.5 flex items-center gap-1.5">
              <Clock className="w-3 h-3" />{elapsed}
              <span className="opacity-40">·</span>
              {jobId.slice(0, 12)}…
            </p>
          </div>

          {/* Action buttons */}
          <div className="flex gap-2 items-start shrink-0">
            {job.failed > 0 && (
              <button
                onClick={exportFailed}
                className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold text-txt-muted hover:text-txt-primary bg-bg-elevated border border-border-subtle rounded-xl transition-all"
              >
                <Download className="w-3.5 h-3.5" /> Export failed
              </button>
            )}
            {running ? (
              <button
                onClick={() => cancelMutation.mutate()}
                disabled={cancelMutation.isPending}
                className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold text-red-400 bg-red-500/10 border border-red-500/25 rounded-xl hover:bg-red-500/20 transition-all disabled:opacity-50"
              >
                <Ban className="w-3.5 h-3.5" /> Cancel
              </button>
            ) : (
              <button
                onClick={onReset}
                className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold text-accent bg-accent/10 border border-accent/25 rounded-xl hover:bg-accent/20 transition-all"
              >
                <RotateCcw className="w-3.5 h-3.5" /> New import
              </button>
            )}
          </div>
        </div>

        {/* Progress bar */}
        <div className="space-y-1.5">
          <div className="w-full h-2.5 bg-bg-elevated rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-500"
              style={{
                width: `${pct}%`,
                background: running
                  ? "linear-gradient(90deg, #4f8ef7 0%, #818cf8 100%)"
                  : completed ? "#10b981" : "#ef4444",
              }}
            />
          </div>
          <div className="flex items-center justify-between text-[11px] text-txt-muted tabular-nums">
            <span>{job.processed.toLocaleString()} / {job.total.toLocaleString()} processed</span>
            <span className="font-bold text-txt-secondary">{pct}%</span>
          </div>
        </div>

        {/* Stat chips */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
          {STATS.map(({ key, label, value, color, bg }) => (
            <div key={key} className={`flex items-center justify-between px-3 py-2 rounded-xl border ${bg}`}>
              <span className="text-xs text-txt-muted">{label}</span>
              <span className={`text-sm font-extrabold tabular-nums ${color}`}>{value.toLocaleString()}</span>
            </div>
          ))}
        </div>
      </div>

      {/* ── Results table ── */}
      <div className="bg-bg-card border border-border-subtle rounded-2xl overflow-hidden">

        {/* Filter tabs */}
        <div className="flex items-center gap-0.5 px-3 py-2.5 border-b border-border-subtle overflow-x-auto">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              onClick={() => setFilter(f.key)}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all shrink-0 ${
                filter === f.key
                  ? "bg-bg-elevated text-txt-primary border border-border-subtle"
                  : "text-txt-muted hover:text-txt-secondary"
              }`}
            >
              {f.label}
              <span className={`tabular-nums ${filter === f.key ? "text-txt-secondary" : "text-txt-muted/50"}`}>
                {f.count.toLocaleString()}
              </span>
            </button>
          ))}
          <span className="ml-auto text-[10px] text-txt-muted/50 shrink-0 pr-1">newest first</span>
        </div>

        {/* Column headers */}
        <div className="hidden sm:grid grid-cols-[190px_auto_1fr_auto] items-center gap-x-3 px-5 py-2 border-b border-border-subtle bg-bg-elevated/40">
          {["VIN", "Status", "Vehicle / Error", "Time"].map((h, i) => (
            <span key={h} className={`text-[10px] font-bold text-txt-muted uppercase tracking-wider ${i === 3 ? "text-right" : ""}`}>
              {h}
            </span>
          ))}
        </div>

        {/* Rows */}
        <div className="max-h-[520px] overflow-y-auto overscroll-contain">
          {filteredResults.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              {running ? (
                <>
                  <Loader2 className="w-6 h-6 animate-spin text-accent/30" />
                  <p className="text-sm text-txt-muted">Waiting for results…</p>
                </>
              ) : (
                <>
                  <Database className="w-6 h-6 text-txt-muted/25" />
                  <p className="text-sm text-txt-muted">No results match this filter</p>
                </>
              )}
            </div>
          ) : (
            filteredResults.map((r, i) => <ResultRow key={r.vin + i} r={r} />)
          )}
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Page root
═══════════════════════════════════════════════════════ */
const JOB_STORAGE_KEY = "import_active_job_id";

export default function ImportPage() {
  const { user, logout } = useAuth();

  const [jobId, setJobId] = useState(() => localStorage.getItem(JOB_STORAGE_KEY));

  const handleJobStarted = useCallback((id) => {
    localStorage.setItem(JOB_STORAGE_KEY, id);
    setJobId(id);
  }, []);

  const handleReset = useCallback(() => {
    localStorage.removeItem(JOB_STORAGE_KEY);
    setJobId(null);
  }, []);

  const { data: job } = useQuery({
    queryKey: ["import-job", jobId],
    queryFn: () => getImportJob(jobId),
    enabled: !!jobId,
    refetchInterval: (query) => (query.state.data?.status === "running" ? 1500 : false),
    retry: false,
  });

  const jobNotFound = jobId && job === undefined && !job;

  return (
    <div className="min-h-dvh bg-bg-base flex flex-col">
      <Navbar user={user} logout={logout} />

      {/* Page header */}
      <div className="shrink-0 border-b border-border-subtle">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-xl bg-accent/15 border border-accent/25 flex items-center justify-center shrink-0">
              <Upload className="w-4 h-4 text-accent" />
            </div>
            <div>
              <h1 className="text-sm font-extrabold text-txt-primary leading-tight">Data Import</h1>
              <p className="text-[11px] text-txt-muted leading-tight">Bulk VIN decode · up to 1,000 at a time</p>
            </div>
            <span className="hidden sm:inline-flex items-center gap-1 text-[10px] font-bold text-accent bg-accent/10 border border-accent/20 rounded-md px-2 py-0.5 ml-1">
              <Zap className="w-2.5 h-2.5" /> GM enrichment
            </span>
          </div>

          {jobId && job && (
            <button
              onClick={handleReset}
              className="flex items-center gap-1.5 text-xs text-txt-muted hover:text-txt-primary transition-colors"
            >
              <RotateCcw className="w-3.5 h-3.5" /> New import
            </button>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1">
        {!jobId || jobNotFound ? (
          <SetupPanel onJobStarted={handleJobStarted} />
        ) : (
          <ProgressPanel job={job} jobId={jobId} onReset={handleReset} />
        )}
      </div>
    </div>
  );
}
