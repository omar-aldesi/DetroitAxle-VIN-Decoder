import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, X, Loader2, Edit3, ShieldCheck } from "lucide-react";
import { updateVehicle } from "../../api/vehicles";
import { useToast } from "../../contexts/ToastContext";
import { useAuth } from "../../contexts/AuthContext";
import FieldHistoryIndicator from "../Fieldhistoryindicator.jsx";

export function SpecRow({ label, value, col, vin, history }) {
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
