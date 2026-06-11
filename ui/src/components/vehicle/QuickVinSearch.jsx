import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Search, ArrowRight } from "lucide-react";

export function QuickVinSearch() {
  const [raw, setRaw] = useState("");
  const navigate = useNavigate();
  const isReady = raw.length === 17;

  const go = () => {
    if (!isReady) return;
    navigate(`/v/${raw}`);
    setRaw("");
  };

  return (
    <div className="hidden sm:flex items-center gap-1.5">
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3 h-3 text-txt-muted/60 pointer-events-none" />
        <input
          value={raw}
          onChange={(e) =>
            setRaw(
              e.target.value
                .replace(/[^a-zA-Z0-9]/g, "")
                .toUpperCase()
                .slice(0, 17),
            )
          }
          onKeyDown={(e) => e.key === "Enter" && go()}
          placeholder="Quick VIN lookup…"
          className={`w-36 lg:w-44 bg-bg-elevated border rounded-lg pl-7 pr-2 py-1.5 text-xs font-mono
                      text-txt-primary placeholder:font-sans placeholder:text-txt-muted/50
                      focus:outline-none transition-all
                      ${isReady ? "border-accent" : "border-border-subtle focus:border-accent/50"}`}
        />
      </div>
      <button
        onClick={go}
        disabled={!isReady}
        title="Go to VIN"
        className="p-1.5 rounded-lg bg-accent/10 text-accent disabled:opacity-30 disabled:cursor-not-allowed hover:bg-accent/20 transition-all"
      >
        <ArrowRight className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}
