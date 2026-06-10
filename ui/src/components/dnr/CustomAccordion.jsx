import { useState } from "react";
import { Layers, ChevronUp, ChevronDown, Trash2, Plus } from "lucide-react";

export function CustomAccordion({ fields, onChange, dirty, open, onToggle }) {
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
