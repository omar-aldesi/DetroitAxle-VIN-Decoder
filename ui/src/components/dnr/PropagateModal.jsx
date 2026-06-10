import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Share2, X, CheckCircle2, Zap, Loader2 } from "lucide-react";
import { useToast } from "../../contexts/ToastContext";
import { getSimilarVehicles, propagateSpecs } from "../../api/dnr";
import { SECTIONS, ALL_KEYS } from "./constants";

export function PropagateModal({ vehicle, formValues, source, onClose }) {
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
