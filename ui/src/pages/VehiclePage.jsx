import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  Fingerprint,
  RefreshCw,
  AlertCircle,
  Car,
  Cpu,
  GitFork,
  Disc,
  Settings2,
  Layers,
  ChevronDown,
  ChevronUp,
  Edit3,
  Check,
  X,
  Loader2,
  Plus,
  WifiOff,
  Copy,
  ChevronsUpDown,
  ShieldCheck,
  Search,
  GitBranch,
  Info,
} from "lucide-react";
import { getVehicle, updateVehicle, fetchGMLive } from "../api/vehicles";
import { getForkData, recordForkPoint } from "../api/fork";
import { copyText } from "../utils/clipboard";
import { useToast } from "../contexts/ToastContext";
import ThemeToggle from "../components/ThemeToggle";
import NotesPanel from "../components/NotesPanel";
import FieldHistoryIndicator from "../components/Fieldhistoryindicator.jsx";
import CompatibleParts from "../components/CompatibleParts.jsx";
import { useAuth } from "../contexts/AuthContext";

const SPEC_SECTIONS = [
  {
    id: "identity",
    label: "Identity",
    Icon: Car,
    iconCls: "text-accent",
    defaultOpen: true,
    fields: [
      { label: "Year", jsonKey: "year", col: "year" },
      { label: "Make", jsonKey: "make", col: "make" },
      { label: "Model", jsonKey: "model", col: "model" },
      { label: "Trim", jsonKey: "trim", col: "trim" },
      { label: "Series", jsonKey: "series", col: "series" },
      { label: "Body Type", jsonKey: "body_type", col: "body_type" },
      { label: "Drive Type", jsonKey: "drive_type", col: "drive_type" },
      { label: "Country", jsonKey: "country", col: "country" },
    ],
  },
  {
    id: "engine",
    label: "Engine",
    Icon: Cpu,
    iconCls: "text-amber-400",
    defaultOpen: false,
    fields: [
      { label: "Cylinders", jsonKey: "cylinders", col: "cylinders" },
      {
        label: "Displacement (L)",
        jsonKey: "displacement_l",
        col: "displacement_l",
      },
      { label: "Fuel Type", jsonKey: "fuel_type", col: "fuel_type" },
    ],
  },
  {
    id: "transmission",
    label: "Transmission",
    Icon: GitFork,
    iconCls: "text-purple-400",
    defaultOpen: false,
    fields: [
      { label: "Type", jsonKey: "transmission_type", col: "transmission_type" },
      { label: "Speeds", jsonKey: "speeds", col: "speeds" },
    ],
  },
  {
    id: "brakes",
    label: "Brakes",
    Icon: Disc,
    iconCls: "text-red-400",
    defaultOpen: false,
    fields: [
      { label: "ABS", jsonKey: "abs", col: "abs" },
      {
        label: "Front Brake Type",
        jsonKey: "front_brake_type",
        col: "front_brake_type",
      },
      {
        label: "Rear Brake Type",
        jsonKey: "rear_brake_type",
        col: "rear_brake_type",
      },
      { label: "GVWR (lbs)", jsonKey: "gvwr_lbs", col: "gvwr_lbs" },
    ],
  },
];

/* Build-number-tier fields — these vary by individual VIN within the build group,
   so they live in the BuildNumberSpecs panel (resolved from the fork engine), not
   the shared spec sections above. Keys match the Vehicle columns / fork registry. */
const FORK_FIELDS = [
  { key: "brake_code", label: "Brake Code", mono: true },
  { key: "front_rotor_size", label: "Front Rotor" },
  { key: "rear_rotor_size", label: "Rear Rotor" },
  { key: "front_spring_type", label: "Front Suspension" },
  { key: "rear_spring_type", label: "Rear Suspension" },
  { key: "steering_type", label: "Steering" },
];

/* Friendly confidence labels — agents shouldn't have to learn engine jargon. */
const FORK_CONFIDENCE = {
  observed: {
    label: "Confirmed",
    color: "#10b981",
    desc: "Seen on two or more VINs across this range",
  },
  manual: {
    label: "Set by team",
    color: "#4f8ef7",
    desc: "Entered as a known range by the DNR team",
  },
  assumed: {
    label: "Assumed",
    color: "#f59e0b",
    desc: "Inferred between known VINs — verify if critical",
  },
  extrapolated: {
    label: "Estimated",
    color: "#f97316",
    desc: "Beyond the VINs we've confirmed — treat with caution",
  },
};

/* ── Copy-to-clipboard button ───────────────────────────────────────── */
function CopyBtn({ text, label = "Copied" }) {
  const toast = useToast();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await copyText(text);
      setCopied(true);
      toast(`${label} copied`, "info");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast("Could not copy to clipboard", "error");
    }
  };

  return (
    <button
      onClick={copy}
      title="Copy to clipboard"
      className="text-txt-muted hover:text-accent transition-colors"
    >
      {copied ? (
        <Check className="w-3.5 h-3.5 text-success" />
      ) : (
        <Copy className="w-3.5 h-3.5" />
      )}
    </button>
  );
}

/* ── Editable spec row ──────────────────────────────────────────────── */
function SpecRow({ label, value, col, vin, history }) {
  const { user } = useAuth();
  const isTrusted = history.some(
    (h) => h.field_name === col && h.is_trusted === true,
  );
  const isVerified = history.some(
    (h) => h.field_name === col && h.is_verified === true,
  );

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const qc = useQueryClient();
  const toast = useToast();

  const mutation = useMutation({
    mutationFn: (v) => updateVehicle(vin, { [col]: v }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["vehicle", vin] });
      setEditing(false);
      toast(`${label} updated`, "success");
    },
    onError: (err) => {
      toast(err.response?.data?.error ?? `Failed to update ${label}`, "error");
    },
  });

  const display =
    value !== null && value !== undefined && value !== "" && value !== 0
      ? String(value)
      : null;

  const openEdit = () => {
    setDraft(display ?? "");
    setEditing(true);
  };
  const save = () => {
    if (draft.trim() === (display ?? "")) {
      setEditing(false);
      return;
    }
    mutation.mutate(draft.trim());
  };

  if (editing) {
    return (
      <div className="spec-row">
        <span className="spec-label">{label}</span>
        <div className="flex items-center gap-1.5">
          <input
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") save();
              if (e.key === "Escape") setEditing(false);
            }}
            className="bg-bg-elevated border border-accent/40 rounded-lg px-2.5 py-1 text-sm text-txt-primary w-36 focus:outline-none focus:border-accent transition-all"
          />
          <button
            onClick={save}
            disabled={mutation.isPending}
            className="text-success hover:opacity-75 transition-opacity"
          >
            {mutation.isPending ? (
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
        </div>
      </div>
    );
  }

  return (
    <div className="spec-row group">
      <span className="spec-label">{label}</span>
      <div className="flex items-center gap-2">
        <span className={`spec-value ${display ? "" : "opacity-30"}`}>
          {display ?? "—"}
        </span>

        <FieldHistoryIndicator fieldName={col} history={history} vin={vin} />
        {/* Show edit button only when neither verified nor trusted */}

        {isVerified && (
          <span
            title="This value has been verified and locked"
            className="flex items-center gap-1 px-1.5 py-0.5 rounded-md
              bg-success/10 border border-success/25 text-success text-[10px] font-semibold"
          >
            <ShieldCheck className="w-3 h-3" />
            Verified
          </span>
        )}

        {isTrusted && (
          <span
            title="This source has been marked as trusted"
            className="flex items-center gap-1 px-1.5 py-0.5 rounded-md
              bg-sky-500/10 border border-sky-500/25 text-sky-500 text-[10px] font-semibold"
          >
            <ShieldCheck className="w-3 h-3" />
            Trusted
          </span>
        )}
        {!isVerified && (
          <button
            onClick={openEdit}
            className="opacity-0 group-hover:opacity-100 transition-opacity text-txt-muted hover:text-accent"
            title={`Edit ${label}`}
          >
            <Edit3 className="w-3 h-3" />
          </button>
        )}
      </div>
    </div>
  );
}

/* ── Collapsible spec section with completeness badge ───────────────── */
function SpecSection({ section, vehicle, vin, expandAll }) {
  const [open, setOpen] = useState(section.defaultOpen);
  const { Icon, iconCls, label, fields } = section;

  /* Sync with expand-all control */
  useEffect(() => {
    if (expandAll !== null) setOpen(expandAll);
  }, [expandAll]);

  const filled = fields.filter(({ jsonKey }) => {
    const v = vehicle[jsonKey];
    return v !== null && v !== undefined && v !== "" && v !== 0;
  }).length;
  const allFilled = filled === fields.length;

  return (
    <div className="section-card animate-fade-in">
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between"
      >
        <span className="section-title">
          <Icon className={`w-4 h-4 ${iconCls}`} />
          {label}
          {/* Completeness bar */}
          <span className="ml-2 flex items-center gap-1 shrink-0">
            <span className="flex gap-px">
              {fields.map((_, i) => (
                <span
                  key={i}
                  className={`w-1 h-2.5 rounded-sm transition-colors ${
                    i < filled
                      ? allFilled
                        ? "bg-success/50"
                        : "bg-accent/40"
                      : "bg-border-subtle"
                  }`}
                />
              ))}
            </span>
          </span>
        </span>
        {open ? (
          <ChevronUp className="w-4 h-4 text-txt-muted" />
        ) : (
          <ChevronDown className="w-4 h-4 text-txt-muted" />
        )}
      </button>
      {open && (
        <div className="mt-3">
          {fields.map(({ label: lbl, jsonKey, col }) => (
            <SpecRow
              key={col}
              label={lbl}
              value={vehicle[jsonKey]}
              col={col}
              vin={vin}
              history={vehicle.history ?? []}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/* ── Custom fields section ──────────────────────────────────────────── */
function CustomFieldsSection({ customFields, vin }) {
  const [open, setOpen] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newVal, setNewVal] = useState("");
  const qc = useQueryClient();
  const toast = useToast();

  const entries = Object.entries(customFields ?? {});

  const addMut = useMutation({
    mutationFn: () =>
      updateVehicle(vin, {
        custom_fields: {
          ...(customFields ?? {}),
          [newKey.trim()]: newVal.trim(),
        },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["vehicle", vin] });
      toast("Custom field added", "success");
      setNewKey("");
      setNewVal("");
      setShowAdd(false);
    },
    onError: (err) =>
      toast(err.response?.data?.error ?? "Failed to add field", "error"),
  });

  const removeField = async (key) => {
    try {
      const updated = { ...(customFields ?? {}) };
      delete updated[key];
      await updateVehicle(vin, { custom_fields: updated });
      qc.invalidateQueries({ queryKey: ["vehicle", vin] });
      toast("Field removed", "info");
    } catch (err) {
      toast(err.response?.data?.error ?? "Failed to remove field", "error");
    }
  };

  return (
    <div className="section-card animate-fade-in">
      <div className="flex items-center justify-between">
        <button
          onClick={() => setOpen((v) => !v)}
          className="flex items-center gap-2 flex-1 min-w-0"
        >
          <span className="section-title">
            <Layers className="w-4 h-4 text-cyan-400" />
            Custom Fields
            <span className="ml-1.5 text-[10px] font-mono text-txt-muted/40 tabular-nums">
              ({entries.length})
            </span>
          </span>
        </button>
        <div className="flex items-center gap-1 ml-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              setShowAdd((v) => !v);
              if (!open) setOpen(true);
            }}
            className="p-1.5 rounded-lg text-txt-muted hover:text-accent hover:bg-bg-elevated transition-all"
            title="Add custom field"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setOpen((v) => !v)}
            className="p-1 text-txt-muted"
          >
            {open ? (
              <ChevronUp className="w-4 h-4" />
            ) : (
              <ChevronDown className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>

      {open && (
        <div className="mt-3">
          {showAdd && (
            <div className="flex gap-2 mb-3 animate-fade-in">
              <input
                autoFocus
                value={newKey}
                onChange={(e) => setNewKey(e.target.value)}
                placeholder="Field name"
                className="flex-1 bg-bg-elevated border border-border-subtle rounded-lg px-3 py-1.5 text-xs text-txt-primary placeholder:text-txt-muted focus:outline-none focus:border-accent transition-all"
              />
              <input
                value={newVal}
                onChange={(e) => setNewVal(e.target.value)}
                onKeyDown={(e) =>
                  e.key === "Enter" &&
                  newKey.trim() &&
                  newVal.trim() &&
                  addMut.mutate()
                }
                placeholder="Value"
                className="flex-1 bg-bg-elevated border border-border-subtle rounded-lg px-3 py-1.5 text-xs text-txt-primary placeholder:text-txt-muted focus:outline-none focus:border-accent transition-all"
              />
              <button
                onClick={() => addMut.mutate()}
                disabled={!newKey.trim() || !newVal.trim() || addMut.isPending}
                className="px-3 py-1.5 bg-accent hover:bg-accent-hover disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-1 transition-all"
              >
                {addMut.isPending ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Check className="w-3.5 h-3.5" />
                )}
              </button>
              <button
                onClick={() => setShowAdd(false)}
                className="text-txt-muted hover:text-txt-secondary transition-colors p-1"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
          {entries.length === 0 ? (
            <p className="text-xs text-txt-muted text-center py-4 opacity-50">
              No custom fields — click <span className="font-mono">+</span> to
              add one
            </p>
          ) : (
            entries.map(([k, v]) => (
              <div key={k} className="spec-row group">
                <span className="spec-label capitalize">
                  {k.replace(/_/g, " ")}
                </span>
                <div className="flex items-center gap-2">
                  <span className="spec-value">{String(v)}</span>
                  <button
                    onClick={() => removeField(k)}
                    className="opacity-0 group-hover:opacity-100 transition-opacity text-txt-muted hover:text-danger"
                    title={`Remove ${k}`}
                  >
                    <X className="w-3 h-3" />
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}

/* ── GM Live Build Options section ──────────────────────────────────── */

// WMI prefixes that belong to GM-manufactured vehicles.
const GM_WMI2 = ["1G", "2G", "3G"];
const GM_WMI3 = ["KL4", "KL8", "KL1", "W0L"];
const GM_MAKES = new Set([
  "chevrolet",
  "gmc",
  "buick",
  "cadillac",
  "pontiac",
  "saturn",
  "oldsmobile",
  "hummer",
  "opel",
  "vauxhall",
]);

function isGMVehicle(vehicle) {
  if (GM_MAKES.has((vehicle.make || "").toLowerCase())) return true;
  const vin = (vehicle.example_build_number || "").toUpperCase();
  if (vin.length >= 3) {
    if (GM_WMI2.some((p) => vin.startsWith(p))) return true;
    if (GM_WMI3.some((p) => vin.startsWith(p))) return true;
  }
  return false;
}

// Title-case alphabetic runs; keep numbers, slashes, punctuation intact.
// Also strips a trailing dash/space left over from codes like "PACKAGE OPTION-".
function formatGMText(raw) {
  return String(raw)
    .replace(
      /[A-Za-z]+/g,
      (w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase(),
    )
    .replace(/[\s-]+$/, "")
    .trim();
}

// Friendly labels for the GM summary rows (major attributes + vehicle info).
const GM_FRIENDLY = {
  Productiondate: "Production Date",
  CatalogCode: "Catalog Code",
  MakeCode: "Make Code",
  ModelCode: "Model Code",
  Vehicle: "Vehicle",
  Engine: "Engine",
  "Model String": "Model String",
  Transmission: "Transmission",
};

// Split a GM spec description into individual RPO entries.
//   "AE8-ADJUSTER FRT ST POWER, 8 WAY"  -> [{ code:"AE8", text:"Adjuster Frt St Power, 8 Way" }]
//   "1SZ-PACKAGE OPTION-;PCW-CONTROL…"  -> two entries (semicolon-separated)
function parseRPO(desc) {
  return String(desc)
    .split(";")
    .map((seg) => seg.trim())
    .filter(Boolean)
    .map((seg) => {
      const m = seg.match(/^([A-Z0-9]{2,4})-(.*)$/);
      if (m) return { code: m[1], text: formatGMText(m[2]) };
      return { code: null, text: formatGMText(seg) };
    });
}

/* Renders GM Parts Giant's native structure: a short summary plus the full,
   per-VIN RPO / build-option list. */
function GMNativeData({ data }) {
  const major = data.major_attributes ?? [];
  const info = data.vehicle_information ?? [];
  const specs = data.specifications ?? [];

  // Flatten every spec entry into individual RPO rows.
  const rpoRows = [];
  for (const s of specs) {
    for (const e of parseRPO(s.desc)) {
      rpoRows.push(e);
    }
  }

  const SummaryRow = ({ label, value }) => (
    <div className="spec-row items-start">
      <span className="spec-label pt-px shrink-0">{label}</span>
      <span className="text-xs text-txt-primary font-medium text-right leading-snug">
        {value}
      </span>
    </div>
  );

  return (
    <div className="mt-3 space-y-4">
      {/* Summary */}
      {(major.length > 0 || info.length > 0) && (
        <div>
          <p className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider mb-1.5 pb-1 border-b border-border-subtle/40">
            Summary
          </p>
          {major.map((x) => (
            <SummaryRow
              key={`m-${x.name}`}
              label={GM_FRIENDLY[x.name] ?? x.name}
              value={formatGMText(x.desc)}
            />
          ))}
          {info.map((x) => (
            <SummaryRow
              key={`i-${x.name}`}
              label={GM_FRIENDLY[x.name] ?? x.name}
              value={x.desc}
            />
          ))}
        </div>
      )}

      {/* Full RPO / build-option list */}
      {rpoRows.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-1.5 pb-1 border-b border-border-subtle/40">
            <p className="text-[10px] font-semibold text-txt-muted uppercase tracking-wider">
              Build Options &amp; RPO Codes
            </p>
            <span className="text-[10px] font-mono text-txt-muted/40 tabular-nums">
              {rpoRows.length}
            </span>
          </div>
          <div className="space-y-0.5">
            {rpoRows.map((r, i) => (
              <div key={i} className="flex items-start gap-2 py-1">
                {r.code ? (
                  <span className="shrink-0 font-mono text-[10px] font-semibold text-accent bg-accent/10 border border-accent/20 rounded px-1.5 py-0.5 leading-none mt-px tabular-nums">
                    {r.code}
                  </span>
                ) : (
                  <span className="shrink-0 w-[38px]" />
                )}
                <span className="text-xs text-txt-secondary leading-snug">
                  {r.text || "—"}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {data.redirect_url && (
        <a
          href={data.redirect_url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1.5 text-[11px] text-accent hover:underline mt-1"
        >
          View on GM Parts Giant ↗
        </a>
      )}
    </div>
  );
}

function GMLiveSection({ vehicle }) {
  const isGM = isGMVehicle(vehicle);
  const [open, setOpen] = useState(true);
  const [vinInput, setVinInput] = useState(vehicle.example_build_number ?? "");
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState(null);
  const hasFetched = useRef(false);

  const doFetch = async (vin) => {
    const target = (vin ?? vinInput).trim().toUpperCase();
    if (target.length !== 17) return;
    setLoading(true);
    setErr(null);
    setData(null);
    try {
      const res = await fetchGMLive(target);
      setData(res.data);
      hasFetched.current = true;
    } catch (e) {
      setErr(
        e?.response?.data?.error ?? e?.message ?? "Failed to fetch GM data",
      );
      hasFetched.current = true;
    } finally {
      setLoading(false);
    }
  };

  // Auto-fetch on page load for GM cars — the user shouldn't have to press
  // anything; GM's per-VIN data should be on screen as soon as the page opens.
  useEffect(() => {
    if (
      isGM &&
      !hasFetched.current &&
      (vehicle.example_build_number ?? "").length === 17
    ) {
      doFetch(vehicle.example_build_number);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isGM, vehicle.example_build_number]);

  if (!isGM) return null;

  const toggle = () => setOpen((v) => !v);

  return (
    <div className="section-card animate-fade-in">
      {/* Header */}
      <button
        onClick={toggle}
        className="w-full flex items-center justify-between"
      >
        <span className="section-title">
          <Car className="w-4 h-4 text-blue-400" />
          GM Build Options
          <span className="ml-1.5 text-[10px] font-mono text-txt-muted/40 tabular-nums">
            live
          </span>
        </span>
        {open ? (
          <ChevronUp className="w-4 h-4 text-txt-muted" />
        ) : (
          <ChevronDown className="w-4 h-4 text-txt-muted" />
        )}
      </button>

      {open && (
        <div className="mt-3">
          {/* VIN input */}
          <div className="flex gap-2 mb-3">
            <input
              value={vinInput}
              onChange={(e) =>
                setVinInput(
                  e.target.value
                    .replace(/[^a-zA-Z0-9]/g, "")
                    .toUpperCase()
                    .slice(0, 17),
                )
              }
              onKeyDown={(e) => e.key === "Enter" && doFetch()}
              placeholder="Enter full VIN…"
              className="flex-1 bg-bg-elevated border border-border-subtle rounded-lg px-3 py-1.5 text-xs font-mono text-txt-primary placeholder:font-sans placeholder:text-txt-muted focus:outline-none focus:border-accent/60 transition-all"
            />
            <button
              onClick={() => doFetch()}
              disabled={loading || vinInput.length !== 17}
              className="px-3 py-1.5 bg-accent/10 hover:bg-accent/20 border border-accent/30 text-accent rounded-lg text-xs font-semibold disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-1.5 transition-all"
            >
              {loading ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Search className="w-3.5 h-3.5" />
              )}
              Look Up
            </button>
          </div>

          <p className="text-[10px] text-txt-muted/60 mb-3 leading-relaxed">
            RPO codes are specific to each individual VIN off the assembly line
            — this data is fetched live and is not saved. Two vehicles of the
            same model may carry different options.
          </p>

          {/* States */}
          {loading && (
            <div className="flex items-center justify-center py-8 gap-2 text-txt-muted text-xs">
              <Loader2 className="w-4 h-4 animate-spin text-blue-400" />
              Fetching from GM Parts Giant…
            </div>
          )}

          {err && !loading && (
            <div className="flex items-start gap-2 bg-danger/5 border border-danger/20 rounded-xl px-3 py-2.5 text-xs text-danger">
              <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-px" />
              {err}
            </div>
          )}

          {data && !loading && <GMNativeData data={data} />}

          {!data && !loading && !err && (
            <p className="text-xs text-txt-muted/50 text-center py-6">
              Enter a full 17-character VIN and click Look Up to see build
              options.
            </p>
          )}
        </div>
      )}
    </div>
  );
}

/* ── Build-Number Specs (fork/range data) ──────────────────────────── */
function padSerial(s) {
  return String(s ?? "").padStart(6, "0");
}

function forkOutcomeMsg(o) {
  switch (o) {
    case "pending":
      return "Sighting saved — one more matching VIN confirms the range.";
    case "range_created":
      return "Range confirmed from two matching VINs!";
    case "reinforced":
      return "Confirmed — strengthened the existing range.";
    case "forked":
      return "New range created.";
    default:
      return "Saved.";
  }
}

function ConfidenceTag({ tier }) {
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

function ForkFieldRow({
  field,
  resolved,
  columnVal,
  ranges,
  pending,
  hasSerial,
  canEdit,
  activeVin,
  onSaved,
}) {
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const list = ranges ?? [];
  const value = resolved?.Value ?? null;
  const fallback = !value && columnVal ? String(columnVal) : null;
  const display = value ?? fallback;
  const pendingCount = pending?.length ?? 0;
  const expandable = list.length > 0 || pendingCount > 0;

  const pointMut = useMutation({
    mutationFn: (v) =>
      recordForkPoint(activeVin, {
        field_key: field.key,
        value: v,
      }).then((r) => r.data),
    onSuccess: (d) => {
      toast(forkOutcomeMsg(d?.outcome), "success");
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
    pointMut.mutate(draft.trim());
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
                disabled={pointMut.isPending || !draft.trim()}
                className="text-success hover:opacity-75 transition-opacity"
              >
                {pointMut.isPending ? (
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
          ) : value ? (
            <>
              <span
                className={`text-sm font-semibold text-txt-primary text-right truncate ${
                  field.mono ? "font-mono tracking-wider" : ""
                }`}
              >
                {value}
              </span>
              <ConfidenceTag tier={resolved.Confidence} />
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
          {!editing && canEdit && (
            <button
              onClick={openEdit}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-txt-muted hover:text-accent shrink-0"
              title={`Record ${field.label} for this VIN`}
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
                <span className="font-mono text-[10px] text-txt-muted/60 tabular-nums shrink-0 tracking-tight">
                  #{padSerial(r.SerialStart)}
                  {" → "}
                  {r.SerialEnd != null ? `#${padSerial(r.SerialEnd)}` : "∞"}
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

function BuildNumberSpecs({ vehicle }) {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [open, setOpen] = useState(true);
  const [vinInput, setVinInput] = useState(vehicle.example_build_number ?? "");
  const [activeVin, setActiveVin] = useState(
    vehicle.example_build_number ?? "",
  );

  const lookupKey = activeVin || vehicle.build_key;
  const { data, isLoading, isError } = useQuery({
    queryKey: ["fork", lookupKey],
    queryFn: () => getForkData(lookupKey).then((r) => r.data),
    enabled: open && !!lookupKey,
    retry: false,
    refetchOnWindowFocus: false,
  });

  /* Trusted agents record per-VIN sightings here; manual ranges stay DNR-only. */
  const canEditFork =
    (user?.isTrusted || user?.isDNR || user?.isAdmin) &&
    activeVin.trim().length === 17;
  const refetchFork = () =>
    qc.invalidateQueries({ queryKey: ["fork", lookupKey] });

  const resolved = data?.resolved ?? {};
  const fields = data?.fields ?? {};
  const pending = data?.pending ?? {};
  const serial = data?.serial;
  const hasSerial = serial != null;

  const applyVin = () => {
    const v = vinInput.trim().toUpperCase();
    if (v.length === 17 || v.length === 10) setActiveVin(v);
  };

  const resolvedCount = FORK_FIELDS.filter(
    (f) => resolved[f.key]?.Value,
  ).length;
  const anyData = FORK_FIELDS.some(
    (f) =>
      resolved[f.key]?.Value || vehicle[f.key] || (fields[f.key]?.length ?? 0),
  );
  const vinLen = vinInput.length;
  const vinReady = vinLen === 17 || vinLen === 10;

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
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Fingerprint className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-txt-muted pointer-events-none" />
                <input
                  value={vinInput}
                  onChange={(e) =>
                    setVinInput(
                      e.target.value
                        .replace(/[^a-zA-Z0-9]/g, "")
                        .toUpperCase()
                        .slice(0, 17),
                    )
                  }
                  onKeyDown={(e) => e.key === "Enter" && applyVin()}
                  placeholder="Enter a VIN to check…"
                  className="w-full bg-bg-elevated border border-border-subtle rounded-lg pl-8 pr-10 py-1.5 text-xs font-mono text-txt-primary placeholder:font-sans placeholder:text-txt-muted focus:outline-none focus:border-accent/60 transition-all"
                />
                {vinLen > 0 && (
                  <span
                    className={`absolute right-2.5 top-1/2 -translate-y-1/2 text-[9px] font-mono tabular-nums transition-colors pointer-events-none ${
                      vinReady ? "text-emerald-400/80" : "text-txt-muted/35"
                    }`}
                  >
                    {vinLen}/17
                  </span>
                )}
              </div>
              <button
                onClick={applyVin}
                disabled={!vinReady}
                className="px-3 py-1.5 bg-accent/10 hover:bg-accent/20 border border-accent/30 text-accent rounded-lg text-xs font-semibold disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              >
                Check
              </button>
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
                Enter a 17-char VIN to see values for that specific build
                number.
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
              <div className="rounded-xl overflow-hidden border border-border-subtle/50">
                {FORK_FIELDS.map((f) => (
                  <ForkFieldRow
                    key={f.key}
                    field={f}
                    resolved={resolved[f.key]}
                    columnVal={vehicle[f.key]}
                    ranges={fields[f.key]}
                    pending={pending[f.key]}
                    hasSerial={hasSerial}
                    canEdit={canEditFork}
                    activeVin={activeVin.trim().toUpperCase()}
                    onSaved={refetchFork}
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

/* ── Record metadata strip ──────────────────────────────────────────── */
function RecordMeta({ createdAt, updatedAt }) {
  const fmt = (iso) => {
    if (!iso) return "—";
    return new Date(iso).toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    });
  };
  return (
    <div className="section-card animate-fade-in mt-3">
      <span className="section-title mb-3">
        <Layers className="w-4 h-4 text-txt-muted/60" />
        Record
      </span>
      <div className="mt-3">
        <div className="spec-row">
          <span className="spec-label">Created</span>
          <span className="spec-value font-mono text-xs">{fmt(createdAt)}</span>
        </div>
        <div className="spec-row">
          <span className="spec-label">Last Updated</span>
          <span className="spec-value font-mono text-xs">{fmt(updatedAt)}</span>
        </div>
      </div>
    </div>
  );
}

/* ── Hero badge pills ───────────────────────────────────────────────── */
function HeroBadge({ label, variant = "default" }) {
  if (!label) return null;
  const styles = {
    default: "bg-bg-elevated border-border-subtle text-txt-secondary",
    fuel: "bg-amber-400/10 border-warn/25 text-warn",
    drive: "bg-accent/10 border-accent/25 text-accent",
  };
  return <span className={`badge border ${styles[variant]}`}>{label}</span>;
}

/* ── Quick VIN lookup — lives in the header ─────────────────────────── */
function QuickVinSearch() {
  const [raw, setRaw] = useState("");
  const navigate = useNavigate();
  const isReady = raw.length === 17;

  const go = () => {
    if (!isReady) return;
    navigate(`/v/${raw}`);
    setRaw("");
  };

  return (
    <div className="hidden sm:flex items-center gap-1.5">
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3 h-3 text-txt-muted/60 pointer-events-none" />
        <input
          value={raw}
          onChange={(e) =>
            setRaw(
              e.target.value
                .replace(/[^a-zA-Z0-9]/g, "")
                .toUpperCase()
                .slice(0, 17),
            )
          }
          onKeyDown={(e) => e.key === "Enter" && go()}
          placeholder="Quick VIN lookup…"
          className={`w-36 lg:w-44 bg-bg-elevated border rounded-lg pl-7 pr-2 py-1.5 text-xs font-mono
                      text-txt-primary placeholder:font-sans placeholder:text-txt-muted/50
                      focus:outline-none transition-all
                      ${isReady ? "border-accent" : "border-border-subtle focus:border-accent/50"}`}
        />
      </div>
      <button
        onClick={go}
        disabled={!isReady}
        title="Go to VIN"
        className="p-1.5 rounded-lg bg-accent/10 text-accent disabled:opacity-30 disabled:cursor-not-allowed hover:bg-accent/20 transition-all"
      >
        <ArrowRight className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

/* ════════════════════════════════════════════════════════════════════
   VehiclePage
══════════════════════════════════════════════════════════════════════ */
export default function VehiclePage() {
  const { vin } = useParams();
  const navigate = useNavigate();
  const handleBack = () => {
    if (window.history.length > 1) {
      navigate(-1);
    } else {
      navigate("/");
    }
  };
  /* null = use section defaults, true = all open, false = all closed */
  const [expandAll, setExpandAll] = useState(null);

  const {
    data: vehicle,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["vehicle", vin],
    queryFn: () => getVehicle(vin).then((r) => r.data),
    enabled: !!vin,
    refetchOnWindowFocus: false,
    retry: false,
  });

  /* ── Loading ─────────────────────────────────────────────────────── */
  if (isLoading) {
    return (
      <div className="min-h-screen bg-bg-base flex items-center justify-center">
        <div className="text-center">
          <div className="w-12 h-12 border-2 border-accent/20 border-t-accent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-txt-secondary text-sm font-medium">
            Decoding VIN…
          </p>
          <p className="font-mono text-xs text-txt-muted mt-1.5 tracking-widest">
            {vin}
          </p>
        </div>
      </div>
    );
  }

  /* ── Error ───────────────────────────────────────────────────────── */
  if (isError) {
    const errMsg =
      error?.response?.data?.error ??
      error?.message ??
      "An unexpected error occurred";
    const isNetwork = !error?.response;
    return (
      <div className="min-h-screen bg-bg-base flex flex-col">
        <header className="border-b border-border-subtle px-6 h-14 flex items-center">
          <button
            onClick={handleBack}
            className="flex items-center gap-2 text-sm text-txt-muted hover:text-txt-primary transition-colors"
          >
            <ArrowLeft className="w-4 h-4" /> Back
          </button>
        </header>
        <div className="flex-1 flex items-center justify-center px-4 py-12">
          <div className="w-full max-w-md">
            <div className="bg-bg-card border border-danger/20 rounded-2xl p-8 shadow-card text-center">
              <div className="w-16 h-16 rounded-2xl bg-danger/10 border border-danger/20 flex items-center justify-center mx-auto mb-5">
                {isNetwork ? (
                  <WifiOff className="w-8 h-8 text-danger/80" />
                ) : (
                  <AlertCircle className="w-8 h-8 text-danger/80" />
                )}
              </div>
              <h2 className="text-xl font-bold text-txt-primary mb-2">
                Decode Failed
              </h2>
              <div className="inline-flex items-center gap-2 bg-bg-elevated border border-border-subtle rounded-lg px-3 py-1.5 mb-4">
                <Fingerprint className="w-3.5 h-3.5 text-accent" />
                <span className="font-mono text-xs text-txt-muted tracking-widest">
                  {vin}
                </span>
              </div>
              <div className="bg-bg-elevated border border-border-subtle rounded-xl px-4 py-3 mb-6 text-left">
                <p className="text-xs font-semibold text-txt-muted uppercase tracking-wider mb-1">
                  {isNetwork ? "Network Error" : "Server Error"}
                </p>
                <p className="text-sm text-txt-secondary leading-relaxed">
                  {errMsg}
                </p>
              </div>
              <div className="flex gap-3">
                <button
                  onClick={handleBack}
                  className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-bg-elevated border border-border-subtle rounded-xl text-sm text-txt-primary hover:border-border transition-all"
                >
                  <ArrowLeft className="w-4 h-4" /> Go back
                </button>
                <button
                  onClick={() => refetch()}
                  className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-accent/10 border border-accent/30 rounded-xl text-sm text-accent hover:bg-accent/20 transition-all"
                >
                  <RefreshCw className="w-4 h-4" /> Retry
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  const title = `${vehicle.year} ${vehicle.make} ${vehicle.model}`;

  return (
    <div className="min-h-screen bg-bg-base">
      {/* ── Sticky header ─────────────────────────────────────────── */}
      <header className="sticky top-0 z-40 bg-bg-base/90 backdrop-blur-md border-b border-border-subtle">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 h-14 flex items-center gap-3">
          <button
            onClick={handleBack}
            className="flex items-center gap-1.5 text-txt-muted hover:text-txt-primary text-sm transition-colors shrink-0"
          >
            <ArrowLeft className="w-4 h-4" />
            <span className="hidden sm:inline">Back</span>
          </button>
          <div className="h-4 w-px bg-border-subtle" />
          <Fingerprint className="w-3.5 h-3.5 text-accent shrink-0" />
          <span className="font-mono text-xs text-txt-muted tracking-widest hidden md:block">
            {vehicle.example_build_number}
          </span>
          <CopyBtn text={vin} label="VIN" />
          <div className="flex-1 min-w-0 ml-1">
            <p className="text-sm font-semibold text-txt-primary truncate">
              {title}
            </p>
          </div>
          <QuickVinSearch />
          <ThemeToggle />
        </div>
      </header>

      {/* ── Body ─────────────────────────────────────────────────── */}
      <div className="max-w-screen-xl mx-auto px-4 sm:px-6 py-7">
        {/* Hero */}
        <div className="mb-7 animate-slide-up">
          <p className="text-xs font-semibold text-txt-muted uppercase tracking-widest mb-1">
            {vehicle.make}
          </p>
          <h1 className="text-4xl sm:text-5xl font-extrabold text-txt-primary leading-none">
            {vehicle.year}&nbsp;
            <span className="text-txt-secondary font-bold">
              {vehicle.model}
            </span>
          </h1>
          {(vehicle.trim || vehicle.series) && (
            <p className="mt-2 text-txt-secondary">
              {[vehicle.trim, vehicle.series].filter(Boolean).join(" · ")}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2 mt-4">
            <HeroBadge label={vehicle.body_type} />
            {vehicle.doors && <HeroBadge label={`${vehicle.doors}-door`} />}
            <HeroBadge label={vehicle.fuel_type} variant="fuel" />
            <HeroBadge label={vehicle.drive_type} variant="drive" />
            {vehicle.cylinders > 0 && (
              <HeroBadge label={`${vehicle.cylinders}-cyl`} />
            )}
            {vehicle.displacement_l > 0 && (
              <HeroBadge label={`${vehicle.displacement_l}L`} />
            )}
          </div>
          {/* VIN chip with copy */}
          <div className="mt-4 inline-flex items-center gap-2 bg-bg-card border border-border-subtle rounded-lg px-3 py-1.5">
            <Fingerprint className="w-3.5 h-3.5 text-accent" />
            <span className="font-mono text-xs text-txt-muted tracking-widest">
              {vin}
            </span>
            <CopyBtn text={vin} label="VIN" />
          </div>
        </div>

        {/* Two-column grid */}
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_420px] gap-5">
          {/* ── Specs column ───────────────────────────────────── */}
          <div>
            {/* Expand / Collapse all bar */}
            <div className="flex items-center justify-between mb-3">
              <span className="text-xs font-semibold text-txt-muted uppercase tracking-widest">
                Specifications
              </span>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => setExpandAll(true)}
                  className="flex items-center gap-1 px-2 py-1 text-xs text-txt-muted hover:text-txt-primary rounded-lg hover:bg-bg-elevated transition-all"
                >
                  <ChevronsUpDown className="w-3 h-3" />
                  Expand all
                </button>
                <span className="text-border opacity-60 select-none">·</span>
                <button
                  onClick={() => setExpandAll(false)}
                  className="px-2 py-1 text-xs text-txt-muted hover:text-txt-primary rounded-lg hover:bg-bg-elevated transition-all"
                >
                  Collapse all
                </button>
              </div>
            </div>

            <div className="space-y-3">
              {SPEC_SECTIONS.map((section) => (
                <SpecSection
                  key={section.id}
                  section={section}
                  vehicle={vehicle}
                  vin={vin}
                  expandAll={expandAll}
                />
              ))}
              <BuildNumberSpecs vehicle={vehicle} />
              <CustomFieldsSection
                customFields={vehicle.custom_fields}
                vin={vin}
              />
              <GMLiveSection vehicle={vehicle} />
            </div>
          </div>

          {/* Notes + Compatible Parts column */}
          <div className="xl:sticky xl:top-[3.75rem] xl:self-start xl:max-h-[calc(100vh-5rem)] xl:overflow-y-auto pb-4 space-y-4">
            <NotesPanel vehicle={vehicle} vin={vin} />
            {/* <CompatibleParts vin={vin} />*/}
          </div>
        </div>
      </div>
    </div>
  );
}
