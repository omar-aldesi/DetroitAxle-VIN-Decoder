import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import {
  Plus,
  X,
  Fingerprint,
  AlertTriangle,
  Loader2,
} from "lucide-react";
import { useToast } from "../../contexts/ToastContext";
import { createVehicleManual } from "../../api/dnr";
import { getVehicle } from "../../api/vehicles";

export function AddVehicleModal({ onClose, onAdded }) {
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
