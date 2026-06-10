import os
import re

root = os.path.join(os.path.dirname(__file__), "..")
src = os.path.join(root, "src", "pages", "DNRPage.jsx")
out_dir = os.path.join(root, "src", "components", "dnr")
os.makedirs(out_dir, exist_ok=True)

with open(src, "r", encoding="utf-8") as f:
    lines = f.readlines()

# 1-based inclusive line ranges
RANGES = {
    "constants.js": (48, 185),
    "helpers.js": (193, 207),
    "Ring.jsx": (210, 243),
    "AddVehicleModal.jsx": (247, 482),
    "WorkQueue.jsx": (496, 812),
    "SectionAccordion.jsx": (816, 936),
    "CustomAccordion.jsx": (940, 1047),
    "BuildNumberPanel.jsx": (1051, 1589),
    "VehicleEditor.jsx": (1593, 1829),
    "PropagateModal.jsx": (1833, 2097),
}

COMMON_IMPORTS = """import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
"""

for fname, (start, end) in RANGES.items():
    chunk = "".join(lines[start - 1 : end])
    chunk = re.sub(r"/\* [═─][\s\S]*?\*/\n*", "", chunk)
    chunk = re.sub(r"/\*\*[^*]*\*/\n?", "", chunk)
    path = os.path.join(out_dir, fname)
    if fname.endswith(".js"):
        chunk = chunk.replace("const ", "export const ")
        chunk = chunk.replace("function primaryKnownVin", "export function primaryKnownVin")
        chunk = chunk.replace("function isFilled", "export function isFilled")
        chunk = chunk.replace("function pctOf", "export function pctOf")
        chunk = chunk.replace("function pctColor", "export function pctColor")
    else:
        if not chunk.strip().startswith("import"):
            chunk = COMMON_IMPORTS + chunk
        chunk = chunk.replace("function ", "export function ")
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write(chunk.strip() + "\n")
    print("wrote", fname)

page = """import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Database } from "lucide-react";
import Navbar from "../components/NavBar";
import { useAuth } from "../contexts/AuthContext";
import { getDNRStats } from "../api/dnr";
import { pctColor } from "../components/dnr/helpers";
import { AddVehicleModal } from "../components/dnr/AddVehicleModal";
import { WorkQueue } from "../components/dnr/WorkQueue";
import { VehicleEditor } from "../components/dnr/VehicleEditor";

export default function DNRPage() {
  const { user, logout } = useAuth();
  const [selected, setSelected] = useState(null);
  const [showAdd, setShowAdd] = useState(false);
  const qc = useQueryClient();

  const { data: stats } = useQuery({
    queryKey: ["dnr-stats"],
    queryFn: () => getDNRStats().then((r) => r.data),
    refetchInterval: 60_000,
    staleTime: 30_000,
  });

  return (
    <div className="flex flex-col bg-bg-base" style={{ height: "100dvh" }}>
      <Navbar user={user} logout={logout} />

      <div className="shrink-0 flex items-center gap-3 px-4 sm:px-6 py-2.5 border-b border-border-subtle bg-bg-surface/40">
        <div className="flex items-center gap-2 shrink-0">
          <div className="w-7 h-7 rounded-lg bg-accent/15 border border-accent/25 flex items-center justify-center">
            <Database className="w-3.5 h-3.5 text-accent" />
          </div>
          <span className="text-sm font-extrabold text-txt-primary">
            DNR Data Center
          </span>
        </div>
        <div className="flex items-center gap-2 overflow-x-auto flex-1">
          {[
            {
              label: "Avg",
              value: stats?.avg_completeness,
              unit: "%",
              color: pctColor(stats?.avg_completeness ?? 0),
            },
            { label: "Total", value: stats?.total_vehicles, color: "#4f8ef7" },
            { label: "Done", value: stats?.fully_complete, color: "#10b981" },
            {
              label: "Today",
              value: stats?.fields_filled_today,
              color: "#f59e0b",
            },
          ].map(({ label, value, unit = "", color }) => (
            <div
              key={label}
              className="flex items-baseline gap-1 bg-bg-elevated border border-border-subtle rounded-lg px-2.5 py-1 shrink-0"
            >
              {value != null ? (
                <span
                  className="text-xs font-extrabold tabular-nums"
                  style={{ color }}
                >
                  {value.toLocaleString()}
                  {unit}
                </span>
              ) : (
                <div className="h-3 w-6 bg-bg-card rounded animate-pulse" />
              )}
              <span className="text-[10px] text-txt-muted">{label}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="flex-1 min-h-0 grid grid-cols-1 xl:grid-cols-[320px_1fr]">
        <WorkQueue
          selectedId={selected?.id}
          onSelect={setSelected}
          onAddClick={() => setShowAdd(true)}
        />
        <VehicleEditor
          vehicle={selected}
          onSaved={() => {
            qc.invalidateQueries({ queryKey: ["dnr-queue"] });
            qc.invalidateQueries({ queryKey: ["dnr-stats"] });
          }}
        />
      </div>

      {showAdd && (
        <AddVehicleModal
          onClose={() => setShowAdd(false)}
          onAdded={(v) => {
            setSelected(v);
            qc.invalidateQueries({ queryKey: ["dnr-queue"] });
          }}
        />
      )}
    </div>
  );
}
"""

with open(src, "w", encoding="utf-8", newline="\n") as out:
    out.write(page)
print("rewrote DNRPage.jsx")
