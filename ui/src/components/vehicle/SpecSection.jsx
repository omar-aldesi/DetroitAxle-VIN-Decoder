import { useState, useEffect } from "react";
import { ChevronUp, ChevronDown } from "lucide-react";
import { SpecRow } from "./SpecRow";

export function SpecSection({ section, vehicle, vin, expandAll }) {
  const [open, setOpen] = useState(section.defaultOpen);
  const { Icon, iconCls, label, fields } = section;

  /* Sync with expand-all control */
  useEffect(() => {
    if (expandAll !== null) setOpen(expandAll);
  }, [expandAll]);

  const filled = fields.filter(({ jsonKey }) => {
    const v = vehicle[jsonKey];
    return v !== null && v !== undefined && v !== "" && v !== 0;
  }).length;
  const allFilled = filled === fields.length;

  return (
    <div className="section-card animate-fade-in">
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between"
      >
        <span className="section-title">
          <Icon className={`w-4 h-4 ${iconCls}`} />
          {label}
          {/* Completeness bar */}
          <span className="ml-2 flex items-center gap-1 shrink-0">
            <span className="flex gap-px">
              {fields.map((_, i) => (
                <span
                  key={i}
                  className={`w-1 h-2.5 rounded-sm transition-colors ${
                    i < filled
                      ? allFilled
                        ? "bg-success/50"
                        : "bg-accent/40"
                      : "bg-border-subtle"
                  }`}
                />
              ))}
            </span>
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
          {fields.map(({ label: lbl, jsonKey, col }) => (
            <SpecRow
              key={col}
              label={lbl}
              value={vehicle[jsonKey]}
              col={col}
              vin={vin}
              history={vehicle.history ?? []}
            />
          ))}
        </div>
      )}
    </div>
  );
}
