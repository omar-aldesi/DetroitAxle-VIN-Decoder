import { useState, useEffect, useRef } from "react";
import {
  Car,
  ChevronUp,
  ChevronDown,
  Loader2,
  Search,
  AlertCircle,
} from "lucide-react";
import { fetchGMLive } from "../../api/vehicles";
import {
  isGMVehicle,
  gmVinFor,
  formatGMText,
  GM_FRIENDLY,
  parseRPO,
} from "./gmUtils";

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

export function GMLiveSection({ vehicle, activeVin }) {
  const isGM = isGMVehicle(vehicle, activeVin);
  const [open, setOpen] = useState(true);
  const [vinInput, setVinInput] = useState(() => gmVinFor(vehicle, activeVin));
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState(null);
  const hasFetched = useRef(false);

  useEffect(() => {
    const next = gmVinFor(vehicle, activeVin);
    if (next.length === 17) setVinInput(next);
  }, [vehicle.id, activeVin, vehicle.viewed_vin, vehicle.known_vins]);

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
    const seed = gmVinFor(vehicle, activeVin);
    if (isGM && !hasFetched.current && seed.length === 17) {
      doFetch(seed);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isGM, vehicle.id, activeVin]);

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
