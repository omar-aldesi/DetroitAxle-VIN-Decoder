import { ChevronUp, ChevronDown } from "lucide-react";
import { isFilled } from "./helpers";

export function SectionAccordion({
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
