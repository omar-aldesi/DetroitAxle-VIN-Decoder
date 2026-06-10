import { Layers } from "lucide-react";

export function RecordMeta({ createdAt, updatedAt }) {
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
