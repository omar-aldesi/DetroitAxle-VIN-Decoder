import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Database,
  Fingerprint,
  Share2,
  BookOpen,
  Loader2,
  Save,
} from "lucide-react";
import { useToast } from "../../contexts/ToastContext";
import { updateVehicle } from "../../api/vehicles";
import { SectionAccordion } from "./SectionAccordion";
import { CustomAccordion } from "./CustomAccordion";
import { BuildNumberPanel } from "./BuildNumberPanel";
import { PropagateModal } from "./PropagateModal";
import { Ring } from "./Ring";
import { SECTIONS, ALL_KEYS, primaryKnownVin } from "./constants";
import { pctOf, pctColor } from "./helpers";

export function VehicleEditor({ vehicle, onSaved }) {
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
      {}
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
              {(vehicle.known_vins?.length > 0 ||
                vehicle.example_build_number) && (
                <span
                  className="font-mono text-[10px] text-txt-muted/60 tracking-widest truncate max-w-[220px]"
                  title={
                    vehicle.known_vins?.length
                      ? vehicle.known_vins.join(", ")
                      : vehicle.example_build_number
                  }
                >
                  {vehicle.known_vins?.length > 0
                    ? `${vehicle.known_vins.length} VIN${vehicle.known_vins.length !== 1 ? "s" : ""} · ${primaryKnownVin(vehicle)}`
                    : `e.g. ${vehicle.example_build_number}`}
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

      {}
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

      {}
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
