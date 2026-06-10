import { useState } from "react";
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
