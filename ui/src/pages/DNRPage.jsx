import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Database,
  Search,
  ChevronLeft,
  ChevronRight,
  Save,
  Share2,
  CheckCircle2,
  Loader2,
  X,
  Plus,
  BookOpen,
  Zap,
  Disc,
  Settings2,
  Cpu,
  GitFork,
  Car,
  Fingerprint,
  AlertTriangle,
  SlidersHorizontal,
  ChevronDown,
  ChevronUp,
  Layers,
  Trash2,
  GitBranch,
  Info,
  ArrowRight,
} from "lucide-react";
import Navbar from "../components/NavBar";
import { useAuth } from "../contexts/AuthContext";
import { useToast } from "../contexts/ToastContext";
import {
  getDNRQueue,
  getDNRStats,
  getSimilarVehicles,
  propagateSpecs,
  createVehicleManual,
} from "../api/dnr";
import { getVehicle, updateVehicle } from "../api/vehicles";
import { getForkData, recordForkPoint, recordForkRange } from "../api/fork";

/* ═══════════════════════════════════════════════════════
   Field catalogue
═══════════════════════════════════════════════════════ */
const SECTIONS = [
  {
    id: "identity",
    label: "Identity",
    Icon: Car,
    color: "#4f8ef7",
    fields: [
      { key: "trim", label: "Trim", type: "text" },
      { key: "series", label: "Series", type: "text" },
      { key: "body_type", label: "Body Type", type: "text" },
      { key: "doors", label: "Doors", type: "text", placeholder: "2, 4" },
      {
        key: "drive_type",
        label: "Drive Type",
        type: "select",
        options: ["FWD", "RWD", "AWD", "4WD"],
      },
      { key: "country", label: "Country", type: "text" },
    ],
  },
  {
    id: "engine",
    label: "Engine",
    Icon: Cpu,
    color: "#f59e0b",
    fields: [
      { key: "cylinders", label: "Cylinders", type: "text", placeholder: "8" },
      {
        key: "displacement_l",
        label: "Displacement (L)",
        type: "text",
        placeholder: "5.3",
      },
      {
        key: "fuel_type",
        label: "Fuel Type",
        type: "select",
        options: ["Gasoline", "Diesel", "Electric", "Hybrid", "Flex Fuel"],
      },
      {
        key: "engine_configuration",
        label: "Configuration",
        type: "text",
        placeholder: "V-type, In-line",
      },
    ],
  },
  {
    id: "transmission",
    label: "Transmission",
    Icon: GitFork,
    color: "#8b5cf6",
    fields: [
      {
        key: "transmission_type",
        label: "Type",
        type: "select",
        options: ["Automatic", "Manual", "CVT", "DCT"],
      },
      { key: "speeds", label: "Speeds", type: "text", placeholder: "6, 8, 10" },
    ],
  },
  {
    id: "brakes",
    label: "Brakes",
    Icon: Disc,
    color: "#ef4444",
    fields: [
      {
        key: "abs",
        label: "ABS",
        type: "select",
        options: ["4-Wheel ABS", "2-Wheel ABS", "None"],
      },
      { key: "brake_system_type", label: "System Type", type: "text" },
      {
        key: "front_brake_type",
        label: "Front Type",
        type: "select",
        options: ["Disc", "Drum"],
      },
      {
        key: "rear_brake_type",
        label: "Rear Type",
        type: "select",
        options: ["Disc", "Drum"],
      },
      {
        key: "gvwr_lbs",
        label: "GVWR (lbs)",
        type: "text",
        placeholder: "7200",
      },
    ],
  },
];

/* Build-number-tier fields — entered per VIN / per range via the Build-Number panel,
   not as shared column edits. Keys match the Vehicle columns / fork registry. */
const FORK_FIELDS = [
  { key: "brake_code", label: "Brake Code", placeholder: "JL9, J55…" },
  { key: "front_rotor_size", label: "Front Rotor", placeholder: "325mm" },
  { key: "rear_rotor_size", label: "Rear Rotor", placeholder: "298mm" },
  {
    key: "front_spring_type",
    label: "Front Suspension",
    placeholder: "Coil, Torsion Bar",
  },
  {
    key: "rear_spring_type",
    label: "Rear Suspension",
    placeholder: "Coil, Leaf",
  },
  { key: "steering_type", label: "Steering", placeholder: "Rack & Pinion" },
];
const FORK_LABEL = Object.fromEntries(FORK_FIELDS.map((f) => [f.key, f.label]));

const ALL_KEYS = SECTIONS.flatMap((s) => s.fields.map((f) => f.key));
const TOTAL = ALL_KEYS.length;

// Keys by category — used for missing-category dots in queue
const CAT_KEYS = {
  brakes:
    SECTIONS.find((s) => s.id === "brakes")?.fields.map((f) => f.key) ?? [],
  engine:
    SECTIONS.find((s) => s.id === "engine")?.fields.map((f) => f.key) ?? [],
  transmission:
    SECTIONS.find((s) => s.id === "transmission")?.fields.map((f) => f.key) ??
    [],
};
const CAT_META = [
  { id: "brakes", color: "#ef4444", label: "Brakes" },
  { id: "engine", color: "#f59e0b", label: "Engine" },
  { id: "transmission", color: "#8b5cf6", label: "Transmission" },
];

/* ── helpers ──────────────────────────────────────────── */
function isFilled(val) {
  return val !== null && val !== undefined && val !== "" && Number(val) !== 0;
}
function pctOf(v) {
  if (!v) return 0;
  return Math.round(
    (ALL_KEYS.filter((k) => isFilled(v[k])).length / TOTAL) * 100,
  );
}
function pctColor(p) {
  if (p >= 90) return "#10b981";
  if (p >= 60) return "#4f8ef7";
  if (p >= 30) return "#f59e0b";
  return "#ef4444";
}

/* ── SVG completeness ring ──────────────────────────────── */
function Ring({ pct, size = 40, strokeWidth = 4 }) {
  const r = (size - strokeWidth) / 2;
  const circ = 2 * Math.PI * r;
  const dash = (pct / 100) * circ;
  return (
    <svg
      width={size}
      height={size}
      className="shrink-0"
      style={{ transform: "rotate(-90deg)" }}
    >
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="rgba(255,255,255,0.07)"
        strokeWidth={strokeWidth}
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke={pctColor(pct)}
        strokeWidth={strokeWidth}
        strokeDasharray={`${dash} ${circ}`}
        strokeLinecap="round"
        style={{ transition: "stroke-dasharray .5s ease" }}
      />
    </svg>
  );
}

/* ═══════════════════════════════════════════════════════
   Add Vehicle Modal
═══════════════════════════════════════════════════════ */
function AddVehicleModal({ onClose, onAdded }) {
  const [mode, setMode] = useState("vin");
  const [vin, setVin] = useState("");
  const [m, setM] = useState({
    year: "",
    make: "",
    model: "",
    trim: "",
    build_key: "",
  });
  const [error, setError] = useState("");
  const toast = useToast();

  const vinMut = useMutation({
    mutationFn: () => getVehicle(vin.trim().toUpperCase()).then((r) => r.data),
    onSuccess: (v) => {
      toast(`Loaded: ${v.year} ${v.make} ${v.model}`, "success");
      onAdded(v);
      onClose();
    },
    onError: (e) => setError(e.response?.data?.error ?? "VIN decode failed"),
  });
  const manMut = useMutation({
    mutationFn: () =>
      createVehicleManual({
        year: Number(m.year),
        make: m.make.trim(),
        model: m.model.trim(),
        trim: m.trim.trim(),
        build_key: m.build_key.trim() || undefined,
      }).then((r) => r.data),
    onSuccess: (v) => {
      toast(`Created: ${v.year} ${v.make} ${v.model}`, "success");
      onAdded(v);
      onClose();
    },
    onError: (e) => setError(e.response?.data?.error ?? "Create failed"),
  });

  const ready =
    mode === "vin"
      ? vin.trim().length === 17 || vin.trim().length === 10
      : m.year && m.make && m.model;
  const pending = vinMut.isPending || manMut.isPending;
  const go = () => {
    setError("");
    mode === "vin" ? vinMut.mutate() : manMut.mutate();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 animate-fade-in"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="w-full max-w-md bg-bg-card border border-border rounded-2xl shadow-2xl animate-pop overflow-hidden">
        <div className="flex items-center justify-between px-6 py-4 border-b border-border-subtle">
          <span className="font-bold text-txt-primary flex items-center gap-2">
            <Plus className="w-4 h-4 text-accent" />
            Add Vehicle
          </span>
          <button
            onClick={onClose}
            className="text-txt-muted hover:text-txt-primary"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        {/* Tabs */}
        <div className="flex gap-1 mx-6 mt-5 p-1 bg-bg-elevated border border-border-subtle rounded-xl">
          {[
            { id: "vin", label: "Decode VIN" },
            { id: "manual", label: "Manual Entry" },
          ].map((t) => (
            <button
              key={t.id}
              onClick={() => {
                setMode(t.id);
                setError("");
              }}
              className={`flex-1 py-2 text-xs font-semibold rounded-lg transition-all ${mode === t.id ? "bg-bg-card text-txt-primary shadow-sm border border-border-subtle" : "text-txt-muted hover:text-txt-secondary"}`}
            >
              {t.label}
            </button>
          ))}
        </div>
        <div className="px-6 py-5 space-y-4">
          {mode === "vin" ? (
            <>
              <div>
                <label className="text-xs font-semibold text-txt-muted uppercase tracking-wider block mb-1.5">
                  VIN or Build Key
                </label>
                <div className="relative">
                  <Fingerprint className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-txt-muted pointer-events-none" />
                  <input
                    autoFocus
                    value={vin}
                    onChange={(e) =>
                      setVin(
                        e.target.value
                          .replace(/[^a-zA-Z0-9]/g, "")
                          .toUpperCase()
                          .slice(0, 17),
                      )
                    }
                    onKeyDown={(e) => e.key === "Enter" && ready && go()}
                    placeholder="1GCUKPEC8GXXXXXXX"
                    className="input-base pl-10 font-mono tracking-widest"
                  />
                </div>
                <p className="text-[10px] text-txt-muted mt-1">
                  {vin.length}/17 &nbsp;
                  {(vin.length === 17 || vin.length === 10) && (
                    <span className="text-success font-semibold">✓ ready</span>
                  )}
                </p>
              </div>
              <p className="text-xs text-txt-muted/60">
                If the VIN isn't in the database yet it will be decoded from
                auto.dev + NHTSA and saved automatically.
              </p>
            </>
          ) : (
            <>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="text-xs font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                    Year *
                  </label>
                  <input
                    type="number"
                    value={m.year}
                    onChange={(e) =>
                      setM((v) => ({ ...v, year: e.target.value }))
                    }
                    placeholder="2019"
                    className="input-base"
                  />
                </div>
                <div className="col-span-2">
                  <label className="text-xs font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                    Make *
                  </label>
                  <input
                    value={m.make}
                    onChange={(e) =>
                      setM((v) => ({ ...v, make: e.target.value }))
                    }
                    placeholder="Chevrolet"
                    className="input-base"
                  />
                </div>
              </div>
              <div>
                <label className="text-xs font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                  Model *
                </label>
                <input
                  value={m.model}
                  onChange={(e) =>
                    setM((v) => ({ ...v, model: e.target.value }))
                  }
                  placeholder="Silverado 1500"
                  className="input-base"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                    Trim
                  </label>
                  <input
                    value={m.trim}
                    onChange={(e) =>
                      setM((v) => ({ ...v, trim: e.target.value }))
                    }
                    placeholder="LT"
                    className="input-base"
                  />
                </div>
                <div>
                  <label className="text-xs font-semibold text-txt-muted uppercase tracking-wider block mb-1">
                    Build Key{" "}
                    <span className="text-txt-muted/50 font-normal">
                      (auto)
                    </span>
                  </label>
                  <input
                    value={m.build_key}
                    onChange={(e) =>
                      setM((v) => ({
                        ...v,
                        build_key: e.target.value.toUpperCase().slice(0, 10),
                      }))
                    }
                    placeholder="optional"
                    className="input-base font-mono"
                  />
                </div>
              </div>
            </>
          )}
          {error && (
            <div className="flex items-center gap-2 bg-danger/10 border border-danger/25 text-danger rounded-xl px-3 py-2 text-xs">
              <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
              {error}
            </div>
          )}
        </div>
        <div className="flex gap-2 px-6 pb-6">
          <button
            onClick={onClose}
            className="flex-1 py-2.5 border border-border-subtle rounded-xl text-sm text-txt-secondary hover:border-border transition-all"
          >
            Cancel
          </button>
          <button
            disabled={!ready || pending}
            onClick={go}
            className="flex-1 bg-accent hover:bg-accent-hover disabled:opacity-40 text-white font-semibold py-2.5 rounded-xl text-sm transition-all shadow-glow-sm flex items-center justify-center gap-2"
          >
            {pending ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : mode === "vin" ? (
              <Fingerprint className="w-4 h-4" />
            ) : (
              <Plus className="w-4 h-4" />
            )}
            {mode === "vin" ? "Decode & Open" : "Create & Open"}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Work Queue
═══════════════════════════════════════════════════════ */
const MISSING_OPTS = [
  { value: "", label: "All vehicles" },
  { value: "brakes", label: "Missing brakes" },
  { value: "engine", label: "Missing engine" },
  { value: "transmission", label: "Missing transmission" },
];
const FUEL_OPTS = ["", "Gasoline", "Diesel", "Electric", "Hybrid", "Flex Fuel"];
const DRIVE_OPTS = ["", "FWD", "RWD", "AWD", "4WD"];
const TRANS_OPTS = ["", "Automatic", "Manual", "CVT", "DCT"];

function WorkQueue({ selectedId, onSelect, onAddClick }) {
  const [draft, setDraft] = useState("");
  const [search, setSearch] = useState("");
  const [missing, setMissing] = useState("");
  const [showAdv, setShowAdv] = useState(false);
  const [page, setPage] = useState(1);
  const [adv, setAdv] = useState({
    year: "",
    make: "",
    model: "",
    cylinders: "",
    displacement_l: "",
    fuel_type: "",
    drive_type: "",
    body_type: "",
    transmission_type: "",
  });

  const activeAdv = Object.values(adv).filter((v) => v !== "").length;
  const params = {
    q: search,
    missing,
    page,
    page_size: 25,
    ...Object.fromEntries(Object.entries(adv).filter(([, v]) => v !== "")),
  };

  const { data, isLoading } = useQuery({
    queryKey: ["dnr-queue", params],
    queryFn: () => getDNRQueue(params).then((r) => r.data),
    keepPreviousData: true,
    staleTime: 30_000,
  });

  const items = data?.items ?? [];
  const totalPages = data?.total_pages ?? 1;
  const sf = (k) => (e) => {
    setAdv((p) => ({ ...p, [k]: e.target.value }));
    setPage(1);
  };
  const clearAll = () => {
    setDraft("");
    setSearch("");
    setMissing("");
    setPage(1);
    setAdv({
      year: "",
      make: "",
      model: "",
      cylinders: "",
      displacement_l: "",
      fuel_type: "",
      drive_type: "",
      body_type: "",
      transmission_type: "",
    });
  };

  return (
    <div className="flex flex-col min-h-0 overflow-hidden border-r border-border-subtle bg-bg-base">
      {/* ── Toolbar ── */}
      <div className="shrink-0 space-y-2 p-3 border-b border-border-subtle">
        {/* Search + Add */}
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-txt-muted pointer-events-none" />
            <input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  setSearch(draft);
                  setPage(1);
                }
              }}
              placeholder="Search vehicles…"
              className="w-full pl-8 pr-3 py-2 text-xs bg-bg-elevated border border-border-subtle rounded-xl placeholder:text-txt-muted focus:outline-none focus:border-accent transition-all"
            />
          </div>
          <button
            onClick={onAddClick}
            className="flex items-center gap-1.5 px-3 py-2 bg-accent hover:bg-accent-hover text-white text-xs font-bold rounded-xl transition-all shadow-glow-sm shrink-0"
          >
            <Plus className="w-3.5 h-3.5" />
            Add
          </button>
        </div>

        {/* Missing filter + advanced toggle */}
        <div className="flex gap-2">
          <select
            value={missing}
            onChange={(e) => {
              setMissing(e.target.value);
              setPage(1);
            }}
            className="flex-1 text-xs bg-bg-elevated border border-border-subtle rounded-xl px-3 py-2 text-txt-secondary focus:outline-none focus:border-accent appearance-none cursor-pointer"
          >
            {MISSING_OPTS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <button
            onClick={() => setShowAdv((v) => !v)}
            className={`flex items-center gap-1.5 px-3 py-2 text-xs font-semibold rounded-xl border transition-all shrink-0 ${showAdv || activeAdv > 0 ? "bg-accent/10 border-accent/30 text-accent" : "bg-bg-elevated border-border-subtle text-txt-muted hover:text-txt-secondary"}`}
          >
            <SlidersHorizontal className="w-3.5 h-3.5" />
            {activeAdv > 0 && (
              <span className="w-4 h-4 rounded-full bg-accent text-white text-[9px] font-bold flex items-center justify-center">
                {activeAdv}
              </span>
            )}
            {showAdv ? (
              <ChevronUp className="w-3 h-3" />
            ) : (
              <ChevronDown className="w-3 h-3" />
            )}
          </button>
        </div>

        {/* Advanced filters */}
        {showAdv && (
          <div className="animate-fade-in bg-bg-elevated border border-border-subtle rounded-xl p-3 space-y-2">
            <p className="text-[10px] font-bold text-txt-muted uppercase tracking-wider">
              Advanced Filters
            </p>
            <div className="grid grid-cols-3 gap-1.5">
              <input
                value={adv.year}
                onChange={sf("year")}
                type="number"
                placeholder="Year"
                className="text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 placeholder:text-txt-muted focus:outline-none focus:border-accent"
              />
              <input
                value={adv.make}
                onChange={sf("make")}
                placeholder="Make"
                className="text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 placeholder:text-txt-muted focus:outline-none focus:border-accent"
              />
              <input
                value={adv.model}
                onChange={sf("model")}
                placeholder="Model"
                className="text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 placeholder:text-txt-muted focus:outline-none focus:border-accent"
              />
            </div>
            <div className="grid grid-cols-2 gap-1.5">
              <input
                value={adv.cylinders}
                onChange={sf("cylinders")}
                placeholder="Cylinders (8)"
                className="text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 placeholder:text-txt-muted focus:outline-none focus:border-accent"
              />
              <input
                value={adv.displacement_l}
                onChange={sf("displacement_l")}
                placeholder="Displacement (5.3)"
                className="text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 placeholder:text-txt-muted focus:outline-none focus:border-accent"
              />
            </div>
            <div className="grid grid-cols-3 gap-1.5">
              {[
                ["fuel_type", FUEL_OPTS, "Fuel"],
                ["drive_type", DRIVE_OPTS, "Drive"],
                ["transmission_type", TRANS_OPTS, "Trans"],
              ].map(([k, opts, ph]) => (
                <select
                  key={k}
                  value={adv[k]}
                  onChange={sf(k)}
                  className="text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 text-txt-secondary focus:outline-none appearance-none cursor-pointer"
                >
                  {opts.map((o) => (
                    <option key={o} value={o}>
                      {o || ph}
                    </option>
                  ))}
                </select>
              ))}
            </div>
            <div className="flex gap-1.5">
              <input
                value={adv.body_type}
                onChange={sf("body_type")}
                placeholder="Body type (Truck, Sedan…)"
                className="flex-1 text-xs bg-bg-card border border-border-subtle rounded-lg px-2 py-1.5 placeholder:text-txt-muted focus:outline-none focus:border-accent"
              />
              {(activeAdv > 0 || search || missing) && (
                <button
                  onClick={clearAll}
                  className="text-xs text-txt-muted hover:text-danger transition-colors px-2 shrink-0"
                >
                  Clear all
                </button>
              )}
            </div>
          </div>
        )}

        {data && (
          <p className="text-[10px] text-txt-muted">
            {data.total_count} vehicle{data.total_count !== 1 ? "s" : ""}
          </p>
        )}
      </div>

      {/* ── List ── */}
      <div className="flex-1 overflow-y-auto overscroll-contain min-h-0">
        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <div className="w-5 h-5 border-2 border-accent/30 border-t-accent rounded-full animate-spin" />
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center px-6">
            <CheckCircle2 className="w-9 h-9 text-success/25 mb-3" />
            <p className="text-sm font-semibold text-txt-secondary">
              Nothing here
            </p>
            <p className="text-xs text-txt-muted mt-1">
              Try adjusting the filters
            </p>
          </div>
        ) : (
          items.map((item) => {
            const pct = item.completeness ?? 0;
            const active = item.id === selectedId;
            const mf = new Set(item.missing_fields ?? []);
            return (
              <button
                key={item.id}
                onClick={() => onSelect(item)}
                className={`w-full text-left flex items-start gap-3 px-4 py-3.5 border-b border-border-subtle/40 transition-all ${active ? "bg-accent/10 border-l-[3px] border-l-accent pl-3.5" : "hover:bg-bg-elevated/60"}`}
              >
                {/* Ring */}
                <div className="relative shrink-0 mt-0.5">
                  <Ring pct={pct} size={42} strokeWidth={4} />
                  <span
                    className="absolute inset-0 flex items-center justify-center text-[9px] font-extrabold"
                    style={{ color: pctColor(pct) }}
                  >
                    {Math.round(pct)}
                  </span>
                </div>
                {/* Info */}
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-bold text-txt-primary truncate leading-snug">
                    {item.year} {item.make} {item.model}
                  </p>
                  {item.trim && (
                    <p className="text-[10px] text-txt-muted truncate">
                      {item.trim}
                    </p>
                  )}
                  <div className="flex items-center gap-1 mt-0.5">
                    <Fingerprint className="w-2.5 h-2.5 text-accent/50 shrink-0" />
                    <p className="text-[10px] font-mono text-txt-muted/60 truncate">
                      {item.build_key}
                    </p>
                  </div>
                  {/* Missing category dots */}
                  <div className="flex items-center gap-1.5 mt-1.5 flex-wrap">
                    {CAT_META.filter((cat) =>
                      CAT_KEYS[cat.id].some((k) => mf.has(k)),
                    ).map((cat) => (
                      <span
                        key={cat.id}
                        className="flex items-center gap-1 text-[9px] font-semibold px-1.5 py-0.5 rounded-md border"
                        style={{
                          color: cat.color,
                          backgroundColor: `${cat.color}18`,
                          borderColor: `${cat.color}35`,
                        }}
                      >
                        <span
                          className="w-1 h-1 rounded-full shrink-0"
                          style={{ backgroundColor: cat.color }}
                        />
                        {cat.label}
                      </span>
                    ))}
                  </div>
                </div>
              </button>
            );
          })
        )}
      </div>

      {/* ── Pagination ── */}
      {totalPages > 1 && (
        <div className="shrink-0 flex items-center justify-between px-4 py-2.5 border-t border-border-subtle">
          <button
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="p-1.5 rounded-lg text-txt-muted hover:text-txt-primary disabled:opacity-30 transition-all"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <span className="text-xs text-txt-muted tabular-nums">
            {page} / {totalPages}
          </span>
          <button
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="p-1.5 rounded-lg text-txt-muted hover:text-txt-primary disabled:opacity-30 transition-all"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Section Accordion
═══════════════════════════════════════════════════════ */
function SectionAccordion({
  section,
  form,
  dirtyKeys,
  open,
  onToggle,
  onChange,
}) {
  const { Icon } = section;

  const filled = section.fields.filter((f) => isFilled(form[f.key])).length;
  const total = section.fields.length;
  const complete = filled === total;
  const empty = filled === 0;
  const dotColor = complete ? "#10b981" : empty ? "#ef4444" : "#f59e0b";
  const dirty = section.fields.some((f) => dirtyKeys.includes(f.key));

  return (
    <div
      className={`border border-border-subtle rounded-2xl overflow-hidden transition-all ${empty ? "border-l-2 border-l-[#ef4444]" : ""}`}
    >
      {/* Header */}
      <button
        onClick={onToggle}
        className={`w-full flex items-center gap-3 px-4 py-3.5 transition-all ${open ? "bg-bg-elevated/50" : "hover:bg-bg-elevated/30"}`}
      >
        <span
          className="w-2.5 h-2.5 rounded-full shrink-0 transition-colors"
          style={{ backgroundColor: dotColor }}
        />
        <Icon className="w-4 h-4 shrink-0" style={{ color: section.color }} />
        <span className="text-sm font-bold text-txt-primary flex-1 text-left">
          {section.label}
        </span>
        {dirty && (
          <span className="text-[9px] font-bold text-accent bg-accent/10 border border-accent/20 rounded px-1.5 py-0.5 shrink-0">
            CHANGED
          </span>
        )}
        <span
          className={`text-xs font-mono tabular-nums shrink-0 ${complete ? "text-success/70" : "text-txt-muted"}`}
        >
          {filled}/{total}
        </span>
        {open ? (
          <ChevronUp className="w-4 h-4 text-txt-muted shrink-0" />
        ) : (
          <ChevronDown className="w-4 h-4 text-txt-muted shrink-0" />
        )}
      </button>

      {/* Fields */}
      {open && (
        <div className="px-4 pb-4 pt-1 grid grid-cols-1 sm:grid-cols-2 gap-3 border-t border-border-subtle">
          {section.fields.map((field) => {
            const val = form[field.key] ?? "";
            const isDirty = dirtyKeys.includes(field.key);
            const isEmpty = !isFilled(val);
            const inputBase =
              "w-full text-sm px-3 py-2.5 rounded-xl border focus:outline-none transition-all bg-bg-elevated text-txt-primary placeholder:text-txt-muted/50";
            const inputRing = isDirty
              ? "border-accent ring-1 ring-accent/20"
              : isEmpty
                ? "border-warn/30 bg-warn/5 focus:border-warn/60"
                : "border-border-subtle focus:border-accent";

            return (
              <div key={field.key}>
                <div className="flex items-center gap-1.5 mb-1.5">
                  <span
                    className="w-1.5 h-1.5 rounded-full shrink-0"
                    style={{
                      backgroundColor: isDirty
                        ? "#4f8ef7"
                        : isEmpty
                          ? "#f59e0b"
                          : "#10b981",
                    }}
                  />
                  <label className="text-xs font-semibold text-txt-secondary">
                    {field.label}
                  </label>
                  {isDirty && (
                    <span className="ml-auto text-[9px] text-accent font-bold">
                      CHANGED
                    </span>
                  )}
                </div>
                {field.type === "select" ? (
                  <select
                    value={val}
                    onChange={(e) => onChange(field.key, e.target.value)}
                    className={`${inputBase} ${inputRing} appearance-none cursor-pointer`}
                  >
                    <option value="">— select —</option>
                    {field.options.map((o) => (
                      <option key={o} value={o}>
                        {o}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    value={val}
                    onChange={(e) => onChange(field.key, e.target.value)}
                    placeholder={
                      field.placeholder ?? `Enter ${field.label.toLowerCase()}`
                    }
                    className={`${inputBase} ${inputRing}`}
                  />
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Custom Fields Accordion
═══════════════════════════════════════════════════════ */
function CustomAccordion({ fields, onChange, dirty, open, onToggle }) {
  const [newKey, setNewKey] = useState("");
  const [newVal, setNewVal] = useState("");
  const entries = Object.entries(fields ?? {});

  const add = () => {
    const k = newKey.trim();
    if (!k) return;
    onChange({ ...(fields ?? {}), [k]: newVal.trim() });
    setNewKey("");
    setNewVal("");
  };

  return (
    <div className="border border-border-subtle rounded-2xl overflow-hidden">
      <button
        onClick={onToggle}
        className={`w-full flex items-center gap-3 px-4 py-3.5 transition-all ${open ? "bg-bg-elevated/50" : "hover:bg-bg-elevated/30"}`}
      >
        <span className="w-2.5 h-2.5 rounded-full shrink-0 bg-cyan-400/60" />
        <Layers className="w-4 h-4 text-cyan-400 shrink-0" />
        <span className="text-sm font-bold text-txt-primary flex-1 text-left">
          Custom Fields
        </span>
        {dirty && (
          <span className="text-[9px] font-bold text-accent bg-accent/10 border border-accent/20 rounded px-1.5 py-0.5 shrink-0">
            CHANGED
          </span>
        )}
        <span className="text-xs font-mono tabular-nums text-txt-muted shrink-0">
          {entries.length}
        </span>
        {open ? (
          <ChevronUp className="w-4 h-4 text-txt-muted shrink-0" />
        ) : (
          <ChevronDown className="w-4 h-4 text-txt-muted shrink-0" />
        )}
      </button>

      {open && (
        <div className="px-4 pb-4 pt-2 border-t border-border-subtle space-y-3">
          <p className="text-xs text-txt-muted">
            Store any extra data that doesn't fit the standard fields — supplier
            codes, catalog references, fitment notes, etc.
          </p>
          {entries.length === 0 ? (
            <div className="flex items-center justify-center py-6 border border-dashed border-border-subtle rounded-xl">
              <p className="text-xs text-txt-muted">No custom fields yet</p>
            </div>
          ) : (
            <div className="space-y-2">
              {entries.map(([k, v]) => (
                <div key={k} className="flex items-center gap-2 group">
                  <span className="w-32 shrink-0 px-3 py-2.5 bg-bg-elevated border border-border-subtle rounded-xl text-xs font-mono text-txt-secondary truncate">
                    {k}
                  </span>
                  <input
                    value={String(v ?? "")}
                    onChange={(e) =>
                      onChange({ ...(fields ?? {}), [k]: e.target.value })
                    }
                    className="flex-1 text-sm px-3 py-2.5 bg-bg-elevated border border-border-subtle rounded-xl text-txt-primary focus:outline-none focus:border-accent transition-all"
                  />
                  <button
                    onClick={() => {
                      const n = { ...(fields ?? {}) };
                      delete n[k];
                      onChange(n);
                    }}
                    className="p-2 rounded-lg text-txt-muted hover:text-danger hover:bg-danger/10 transition-all opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
          {/* Add new */}
          <div className="flex items-center gap-2 pt-1 border-t border-border-subtle/50">
            <input
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && add()}
              placeholder="field_name"
              className="w-36 shrink-0 text-xs px-3 py-2 bg-bg-elevated border border-border-subtle rounded-xl font-mono placeholder:text-txt-muted/60 focus:outline-none focus:border-accent transition-all"
            />
            <input
              value={newVal}
              onChange={(e) => setNewVal(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && add()}
              placeholder="value"
              className="flex-1 text-xs px-3 py-2 bg-bg-elevated border border-border-subtle rounded-xl placeholder:text-txt-muted/60 focus:outline-none focus:border-accent transition-all"
            />
            <button
              onClick={add}
              disabled={!newKey.trim()}
              className="flex items-center gap-1 px-3 py-2 bg-accent/10 hover:bg-accent/20 disabled:opacity-30 text-accent text-xs font-semibold rounded-xl transition-all shrink-0"
            >
              <Plus className="w-3.5 h-3.5" />
              Add
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Build-Number Data Panel  (fork / range entry)
═══════════════════════════════════════════════════════ */
function forkOutcomeMsg(o) {
  switch (o) {
    case "pending":
      return "Saved as a sighting — add one more matching VIN to confirm the range.";
    case "range_created":
      return "Range confirmed from two matching VINs!";
    case "reinforced":
      return "Confirmed — strengthened the existing range.";
    case "forked":
      return "New range created.";
    case "manual_range":
      return "Range saved.";
    default:
      return "Saved.";
  }
}

/** True if serial falls within [start, end] (end=null means open) */
function serialInRange(serial, start, end) {
  if (serial == null) return false;
  return serial >= start && (end == null || serial <= end);
}

function ConfidencePip({ exact }) {
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

function BuildNumberPanel({ vehicle, open, onToggle }) {
  const toast = useToast();
  const qc = useQueryClient();
  const [mode, setMode] = useState("vin"); // "vin" | "range"
  const [field, setField] = useState("brake_code");
  const [value, setValue] = useState("");
  const [vin, setVin] = useState(vehicle.example_build_number ?? "");
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [allBuilds, setAllBuilds] = useState(false);

  useEffect(() => {
    setValue("");
    setStart("");
    setEnd("");
    setAllBuilds(false);
    setVin(vehicle?.example_build_number ?? "");
    setField("brake_code");
    setMode("vin");
  }, [vehicle?.id]);

  const lookup = vehicle.example_build_number || vehicle.build_key;
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
              These specs differ between individual VINs. Add a value for one
              VIN, or set a range of build numbers directly. Two matching VINs
              confirm a range automatically.
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
                    onChange={(e) =>
                      setVin(
                        e.target.value
                          .replace(/[^a-zA-Z0-9]/g, "")
                          .toUpperCase()
                          .slice(0, 17),
                      )
                    }
                    onKeyDown={(e) => e.key === "Enter" && submit()}
                    placeholder="1G1AK55F177000050"
                    className={`${inputCls} pl-10 font-mono tracking-wider`}
                  />
                </div>
                <p className="text-[10px] text-txt-muted mt-1">
                  {vin.length}/17
                  {vin.length === 17 && (
                    <span className="text-success font-semibold"> ✓ ready</span>
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
                                <span className="font-mono text-[10px] tabular-nums text-txt-muted shrink-0">
                                  {r.SerialStart.toLocaleString()} →{" "}
                                  {r.SerialEnd != null
                                    ? r.SerialEnd.toLocaleString()
                                    : "∞"}
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

/* ═══════════════════════════════════════════════════════
   Vehicle Editor
═══════════════════════════════════════════════════════ */
function VehicleEditor({ vehicle, onSaved }) {
  const toast = useToast();
  const qc = useQueryClient();

  const [form, setForm] = useState({});
  const [customFields, setCF] = useState({});
  const [source, setSource] = useState("");
  const [showProp, setShowProp] = useState(false);

  // Track which sections are open — all open by default
  const [open, setOpen] = useState(() => {
    const init = {};
    SECTIONS.forEach((s) => {
      init[s.id] = true;
    });
    init.buildnum = true;
    init.custom = false;
    return init;
  });

  useEffect(() => {
    if (!vehicle) {
      setForm({});
      setCF({});
      setSource("");
      return;
    }
    const init = {};
    ALL_KEYS.forEach((k) => {
      init[k] = vehicle[k] ?? "";
    });
    setForm(init);
    setCF(vehicle.custom_fields ?? {});
    setSource("");
    // Re-open all spec sections, close custom
    const o = {};
    SECTIONS.forEach((s) => {
      o[s.id] = true;
    });
    o.buildnum = true;
    o.custom = false;
    setOpen(o);
  }, [vehicle?.id]);

  const dirtyKeys = vehicle
    ? ALL_KEYS.filter(
        (k) =>
          String(vehicle[k] ?? "") !== String(form[k] ?? "") && form[k] !== "",
      )
    : [];
  const customDirty =
    JSON.stringify(customFields) !==
    JSON.stringify(vehicle?.custom_fields ?? {});
  const hasChanges = dirtyKeys.length > 0 || customDirty;
  const totalChanged = dirtyKeys.length + (customDirty ? 1 : 0);

  const mutation = useMutation({
    mutationFn: () => {
      const payload = {};
      dirtyKeys.forEach((k) => (payload[k] = form[k]));
      if (customDirty) payload["custom_fields"] = customFields;
      if (source.trim()) payload["_source"] = source.trim();
      return updateVehicle(vehicle.build_key, payload);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dnr-queue"] });
      qc.invalidateQueries({ queryKey: ["dnr-stats"] });
      toast(
        `Saved — ${totalChanged} change${totalChanged !== 1 ? "s" : ""} applied`,
        "success",
      );
      onSaved?.();
    },
    onError: (err) =>
      toast(err.response?.data?.error ?? "Save failed", "error"),
  });

  useEffect(() => {
    const h = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        if (hasChanges && !mutation.isPending) mutation.mutate();
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [hasChanges, mutation]);

  const toggleSection = (id) => setOpen((p) => ({ ...p, [id]: !p[id] }));
  const pct = pctOf(form);

  /* ── Empty state ── */
  if (!vehicle)
    return (
      <div className="flex flex-col min-h-0 flex-1 items-center justify-center text-center p-12 bg-bg-base">
        <div className="w-20 h-20 rounded-3xl bg-accent/8 border border-accent/15 flex items-center justify-center mb-5">
          <Database className="w-9 h-9 text-accent/30" />
        </div>
        <p className="text-base font-bold text-txt-secondary mb-1">
          Select a vehicle from the queue
        </p>
        <p className="text-sm text-txt-muted max-w-xs leading-relaxed">
          Or click <span className="font-bold text-txt-secondary">Add</span> to
          load a VIN or create a new record.
        </p>
      </div>
    );

  return (
    <div className="flex flex-col min-h-0 overflow-hidden">
      {/* ── Vehicle header (sticky) ── */}
      <div className="shrink-0 px-5 py-4 border-b border-border-subtle bg-bg-surface/50">
        <div className="flex items-center gap-4">
          {/* Ring + title */}
          <div className="relative shrink-0">
            <Ring pct={pct} size={52} strokeWidth={5} />
            <span
              className="absolute inset-0 flex items-center justify-center text-[10px] font-extrabold"
              style={{ color: pctColor(pct) }}
            >
              {pct}%
            </span>
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="text-base font-extrabold text-txt-primary leading-tight truncate">
              {vehicle.year} {vehicle.make} {vehicle.model}
            </h2>
            {vehicle.trim && (
              <p className="text-xs text-txt-muted mt-0.5">{vehicle.trim}</p>
            )}
            {/* Build key + example VIN */}
            <div className="flex items-center gap-3 mt-1.5 flex-wrap">
              <div className="flex items-center gap-1.5">
                <Fingerprint className="w-3 h-3 text-accent shrink-0" />
                <span className="font-mono text-[10px] text-txt-secondary font-semibold tracking-widest">
                  {vehicle.build_key}
                </span>
              </div>
              {vehicle.example_build_number && (
                <span className="font-mono text-[10px] text-txt-muted/60 tracking-widest">
                  e.g. {vehicle.example_build_number}
                </span>
              )}
              {hasChanges && (
                <span className="text-[10px] font-bold text-accent bg-accent/10 border border-accent/20 rounded px-1.5 py-0.5">
                  {totalChanged} unsaved
                </span>
              )}
            </div>
          </div>
          {/* Propagate */}
          <button
            onClick={() => setShowProp(true)}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold text-emerald-500 bg-emerald-500/10 border border-emerald-500/25 rounded-xl hover:bg-emerald-500/20 transition-all shrink-0"
          >
            <Share2 className="w-3.5 h-3.5" />
            Propagate
          </button>
        </div>
      </div>

      {/* ── Scrollable accordions ── */}
      <div className="flex-1 overflow-y-auto overscroll-contain min-h-0 px-5 py-4 space-y-3">
        {SECTIONS.map((s) => (
          <SectionAccordion
            key={s.id}
            section={s}
            form={form}
            dirtyKeys={dirtyKeys}
            open={!!open[s.id]}
            onToggle={() => toggleSection(s.id)}
            onChange={(k, v) => setForm((p) => ({ ...p, [k]: v }))}
          />
        ))}
        <BuildNumberPanel
          vehicle={vehicle}
          open={!!open.buildnum}
          onToggle={() => toggleSection("buildnum")}
        />
        <CustomAccordion
          fields={customFields}
          onChange={setCF}
          dirty={customDirty}
          open={!!open.custom}
          onToggle={() => toggleSection("custom")}
        />
      </div>

      {/* ── Sticky action bar ── */}
      <div className="shrink-0 flex items-center gap-3 px-5 py-3 border-t border-border-subtle bg-bg-surface/60">
        <BookOpen className="w-3.5 h-3.5 text-txt-muted shrink-0" />
        <input
          value={source}
          onChange={(e) => setSource(e.target.value)}
          placeholder="Data source (e.g. Gates 2024 catalog, p.142)"
          className="flex-1 text-xs px-3 py-2 bg-bg-elevated border border-border-subtle rounded-xl placeholder:text-txt-muted/50 focus:outline-none focus:border-accent transition-all"
        />
        <button
          onClick={() => mutation.mutate()}
          disabled={!hasChanges || mutation.isPending}
          className="flex items-center gap-2 px-5 py-2 bg-accent hover:bg-accent-hover disabled:opacity-40 text-white text-sm font-bold rounded-xl transition-all shadow-glow-sm shrink-0"
        >
          {mutation.isPending ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Save className="w-4 h-4" />
          )}
          Save{" "}
          {hasChanges && (
            <span className="opacity-70 text-xs">({totalChanged})</span>
          )}
          <kbd className="text-white/40 text-[10px] font-mono">⌘S</kbd>
        </button>
      </div>

      {showProp && (
        <PropagateModal
          vehicle={vehicle}
          formValues={form}
          source={source}
          onClose={() => setShowProp(false)}
        />
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Propagate Modal
═══════════════════════════════════════════════════════ */
function PropagateModal({ vehicle, formValues, source, onClose }) {
  const toast = useToast();
  const [criteria, setCriteria] = useState("same_model_year");
  const [sel, setSel] = useState(() => {
    const filled = ALL_KEYS.filter((k) => {
      const v = formValues[k];
      return v !== null && v !== undefined && v != "" && v !== "0";
    });
    /* Fork fields (rotor sizes, suspension, steering) are build-number-tier and
       excluded from propagation — pre-select only shared brake-section keys. */
    const brakeKeys =
      SECTIONS.find((s) => s.id === "brakes")?.fields.map((f) => f.key) ?? [];
    return new Set(filled.filter((k) => brakeKeys.includes(k)));
  });
  const [result, setResult] = useState(null);

  const { data: similar } = useQuery({
    queryKey: ["dnr-similar", vehicle.id, criteria],
    queryFn: () => getSimilarVehicles(vehicle.id, criteria).then((r) => r.data),
    staleTime: 60_000,
  });

  const mutation = useMutation({
    mutationFn: () =>
      propagateSpecs({
        source_vehicle_id: vehicle.id,
        fields: [...sel],
        criteria,
        source: source || undefined,
        dry_run: false,
      }).then((r) => r.data),
    onSuccess: (data) => {
      setResult(data);
      toast(`Propagated to ${data.updated_count} vehicles`, "success");
    },
    onError: () => toast("Propagation failed", "error"),
  });

  const filledKeys = ALL_KEYS.filter((k) => {
    const v = formValues[k];
    return v !== null && v !== undefined && v != "" && v !== "0";
  });
  const toggle = (k) =>
    setSel((p) => {
      const n = new Set(p);
      n.has(k) ? n.delete(k) : n.add(k);
      return n;
    });

  if (result)
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 animate-fade-in">
        <div className="w-full max-w-sm bg-bg-card border border-border rounded-2xl shadow-2xl p-8 flex flex-col items-center gap-4 animate-pop">
          <div className="w-14 h-14 rounded-2xl bg-success/10 border border-success/25 flex items-center justify-center">
            <CheckCircle2 className="w-7 h-7 text-success" />
          </div>
          <div className="text-center">
            <p className="text-lg font-bold text-txt-primary">Done!</p>
            <p className="text-sm text-txt-muted mt-1">
              Applied to{" "}
              <span className="font-bold text-success">
                {result.updated_count}
              </span>{" "}
              vehicle{result.updated_count !== 1 ? "s" : ""}
            </p>
          </div>
          {result.affected?.length > 0 && (
            <div className="w-full max-h-40 overflow-y-auto divide-y divide-border-subtle/50 border border-border-subtle rounded-xl text-xs">
              {result.affected.map((v) => (
                <div
                  key={v.vehicle_id}
                  className="flex justify-between px-3 py-2"
                >
                  <span className="text-txt-secondary">
                    {v.year} {v.make} {v.model}
                  </span>
                  <span className="text-txt-muted">
                    {v.applied_fields?.length} fields
                  </span>
                </div>
              ))}
            </div>
          )}
          <button
            onClick={onClose}
            className="w-full py-2.5 bg-accent hover:bg-accent-hover text-white font-semibold rounded-xl text-sm"
          >
            Close
          </button>
        </div>
      </div>
    );

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 animate-fade-in"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="w-full max-w-md bg-bg-card border border-border rounded-2xl shadow-2xl animate-pop overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border-subtle">
          <span className="flex items-center gap-2 font-bold text-txt-primary text-sm">
            <Share2 className="w-4 h-4 text-emerald-500" />
            Propagate Specs
          </span>
          <button
            onClick={onClose}
            className="text-txt-muted hover:text-txt-primary"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* What this does */}
        <div className="mx-6 mt-5 bg-emerald-500/8 border border-emerald-500/20 rounded-xl px-4 py-3 space-y-1">
          <p className="text-xs font-semibold text-emerald-400">
            What this does
          </p>
          <p className="text-xs text-txt-secondary leading-relaxed">
            Copies the specs you just researched on{" "}
            <span className="font-semibold text-txt-primary">
              {vehicle.year} {vehicle.make} {vehicle.model}
            </span>{" "}
            to other vehicles of the same build that are still missing those
            specs.
          </p>
          <p className="text-[11px] text-txt-muted mt-1">
            ✓ Only fills <strong>empty</strong> fields — never overwrites
            existing data.
          </p>
        </div>

        <div className="px-6 py-4 space-y-5">
          {/* Step 1: who to target */}
          <div>
            <p className="text-[11px] font-bold text-txt-muted uppercase tracking-wider mb-2">
              Step 1 — Find similar vehicles by
            </p>
            <div className="grid grid-cols-2 gap-2">
              {[
                {
                  v: "same_model_year",
                  l: "Same year / make / model",
                  sub: "Broadest match",
                },
                {
                  v: "same_engine",
                  l: "+ Same engine size",
                  sub: "More precise",
                },
              ].map((o) => (
                <button
                  key={o.v}
                  onClick={() => setCriteria(o.v)}
                  className={`px-3 py-3 rounded-xl border text-left transition-all ${criteria === o.v ? "bg-accent/10 border-accent/40 text-accent" : "bg-bg-elevated border-border-subtle text-txt-muted hover:border-border"}`}
                >
                  <p className="text-xs font-semibold">{o.l}</p>
                  <p className="text-[10px] opacity-60 mt-0.5">{o.sub}</p>
                </button>
              ))}
            </div>
            <div className="flex items-center gap-1.5 mt-2.5">
              <Zap className="w-3 h-3 text-amber-500 shrink-0" />
              {similar == null ? (
                <span className="text-xs text-txt-muted">Searching…</span>
              ) : similar.similar_count === 0 ? (
                <span className="text-xs text-txt-muted">
                  No matching vehicles found in DB
                </span>
              ) : (
                <span className="text-xs text-txt-muted">
                  Found{" "}
                  <span className="font-bold text-txt-secondary">
                    {similar.similar_count}
                  </span>{" "}
                  vehicle{similar.similar_count !== 1 ? "s" : ""} — will fill
                  only the ones missing the selected specs
                </span>
              )}
            </div>
          </div>

          {/* Step 2: which fields */}
          <div>
            <p className="text-[11px] font-bold text-txt-muted uppercase tracking-wider mb-2">
              Step 2 — Select specs to copy
            </p>
            {filledKeys.length === 0 ? (
              <div className="bg-bg-elevated border border-border-subtle rounded-xl px-4 py-5 text-center">
                <p className="text-xs text-txt-muted">
                  Fill in some specs on this vehicle first, then propagate.
                </p>
              </div>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-1.5 max-h-52 overflow-y-auto pr-1 overscroll-contain">
                  {SECTIONS.flatMap((s) =>
                    s.fields
                      .filter((f) => filledKeys.includes(f.key))
                      .map((f) => {
                        const checked = sel.has(f.key);
                        return (
                          <label
                            key={f.key}
                            className={`flex items-center gap-2 px-2.5 py-2 rounded-lg border text-xs cursor-pointer transition-all ${checked ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400" : "bg-bg-elevated border-border-subtle text-txt-muted hover:border-border"}`}
                          >
                            <input
                              type="checkbox"
                              className="sr-only"
                              checked={checked}
                              onChange={() => toggle(f.key)}
                            />
                            <span
                              className={`w-3.5 h-3.5 rounded border flex items-center justify-center shrink-0 ${checked ? "bg-emerald-500 border-emerald-500" : "border-txt-muted/30"}`}
                            >
                              {checked && (
                                <span className="text-white text-[8px]">✓</span>
                              )}
                            </span>
                            <span className="truncate">{f.label}</span>
                            <span className="ml-auto font-mono text-[9px] opacity-60 shrink-0">
                              {String(formValues[f.key]).slice(0, 8)}
                            </span>
                          </label>
                        );
                      }),
                  )}
                </div>
                <p className="text-[10px] text-txt-muted/60 mt-1.5">
                  {sel.size} field{sel.size !== 1 ? "s" : ""} selected
                </p>
              </>
            )}
          </div>
        </div>

        <div className="flex items-center justify-between gap-3 px-6 py-4 border-t border-border-subtle">
          <button
            onClick={onClose}
            className="text-sm text-txt-muted hover:text-txt-primary"
          >
            Cancel
          </button>
          <button
            disabled={
              sel.size === 0 ||
              mutation.isPending ||
              (similar?.similar_count ?? 0) === 0
            }
            onClick={() => mutation.mutate()}
            className="flex items-center gap-2 px-5 py-2.5 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-40 text-white text-sm font-bold rounded-xl transition-all"
          >
            {mutation.isPending ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Share2 className="w-4 h-4" />
            )}
            Copy to {similar?.similar_count ?? "?"} vehicles
          </button>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════
   Page
═══════════════════════════════════════════════════════ */
export default function DNRPage() {
  const { user, logout } = useAuth();
  const [selected, setSelected] = useState(null);
  const [showAdd, setShowAdd] = useState(false);
  const qc = useQueryClient();

  const { data: stats } = useQuery({
    queryKey: ["dnr-stats"],
    queryFn: () => getDNRStats().then((r) => r.data),
    refetchInterval: 60_000,
    staleTime: 30_000,
  });

  return (
    <div className="flex flex-col bg-bg-base" style={{ height: "100dvh" }}>
      <Navbar user={user} logout={logout} />

      {/* ── Compact page header ── */}
      <div className="shrink-0 flex items-center gap-3 px-4 sm:px-6 py-2.5 border-b border-border-subtle bg-bg-surface/40">
        <div className="flex items-center gap-2 shrink-0">
          <div className="w-7 h-7 rounded-lg bg-accent/15 border border-accent/25 flex items-center justify-center">
            <Database className="w-3.5 h-3.5 text-accent" />
          </div>
          <span className="text-sm font-extrabold text-txt-primary">
            DNR Data Center
          </span>
        </div>
        {/* Stats */}
        <div className="flex items-center gap-2 overflow-x-auto flex-1">
          {[
            {
              label: "Avg",
              value: stats?.avg_completeness,
              unit: "%",
              color: pctColor(stats?.avg_completeness ?? 0),
            },
            { label: "Total", value: stats?.total_vehicles, color: "#4f8ef7" },
            { label: "Done", value: stats?.fully_complete, color: "#10b981" },
            {
              label: "Today",
              value: stats?.fields_filled_today,
              color: "#f59e0b",
            },
          ].map(({ label, value, unit = "", color }) => (
            <div
              key={label}
              className="flex items-baseline gap-1 bg-bg-elevated border border-border-subtle rounded-lg px-2.5 py-1 shrink-0"
            >
              {value != null ? (
                <span
                  className="text-xs font-extrabold tabular-nums"
                  style={{ color }}
                >
                  {value.toLocaleString()}
                  {unit}
                </span>
              ) : (
                <div className="h-3 w-6 bg-bg-card rounded animate-pulse" />
              )}
              <span className="text-[10px] text-txt-muted">{label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* ── Split panel ── */}
      <div className="flex-1 min-h-0 grid grid-cols-1 xl:grid-cols-[320px_1fr]">
        <WorkQueue
          selectedId={selected?.id}
          onSelect={setSelected}
          onAddClick={() => setShowAdd(true)}
        />
        <VehicleEditor
          vehicle={selected}
          onSaved={() => {
            qc.invalidateQueries({ queryKey: ["dnr-queue"] });
            qc.invalidateQueries({ queryKey: ["dnr-stats"] });
          }}
        />
      </div>

      {showAdd && (
        <AddVehicleModal
          onClose={() => setShowAdd(false)}
          onAdded={(v) => {
            setSelected(v);
            qc.invalidateQueries({ queryKey: ["dnr-queue"] });
          }}
        />
      )}
    </div>
  );
}
