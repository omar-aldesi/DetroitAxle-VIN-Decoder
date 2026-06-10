import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Layers,
  Plus,
  ChevronUp,
  ChevronDown,
  Loader2,
  Check,
  X,
} from "lucide-react";
import { updateVehicle } from "../../api/vehicles";
import { useToast } from "../../contexts/ToastContext";

export function CustomFieldsSection({ customFields, vin }) {
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
