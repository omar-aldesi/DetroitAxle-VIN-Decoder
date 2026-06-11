import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Search,
  Plus,
  SlidersHorizontal,
  ChevronUp,
  ChevronDown,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Fingerprint,
} from "lucide-react";
import { getDNRQueue } from "../../api/dnr";
import { Ring } from "./Ring";
import { pctColor } from "./helpers";
import {
  CAT_META,
  CAT_KEYS,
  MISSING_OPTS,
  FUEL_OPTS,
  DRIVE_OPTS,
  TRANS_OPTS,
} from "./constants";

export function WorkQueue({ selectedId, onSelect, onAddClick }) {
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
      {}
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

      {}
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

      {}
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
