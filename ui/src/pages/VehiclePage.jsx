import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  Fingerprint,
  RefreshCw,
  AlertCircle,
  WifiOff,
  ChevronsUpDown,
  Car,
  DoorOpen,
  Fuel,
  Cog,
  Gauge,
  Activity,
} from "lucide-react";
import { getVehicle } from "../api/vehicles";
import ThemeToggle from "../components/ThemeToggle";
import NotesPanel from "../components/NotesPanel";
import { SPEC_SECTIONS } from "../components/vehicle/constants";
import { CopyBtn } from "../components/vehicle/CopyBtn";
import { SpecSection } from "../components/vehicle/SpecSection";
import { CustomFieldsSection } from "../components/vehicle/CustomFieldsSection";
import { BuildNumberSpecs } from "../components/vehicle/BuildNumberSpecs";
import { GMLiveSection } from "../components/vehicle/GMLiveSection";
import { HeroBadge } from "../components/vehicle/HeroBadge";
import { QuickVinSearch } from "../components/vehicle/QuickVinSearch";

export default function VehiclePage() {
  const { vin } = useParams();
  const navigate = useNavigate();
  const handleBack = () => {
    if (window.history.length > 1) {
      navigate(-1);
    } else {
      navigate("/");
    }
  };
  /* null = use section defaults, true = all open, false = all closed */
  const [expandAll, setExpandAll] = useState(null);

  const {
    data: vehicle,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["vehicle", vin],
    queryFn: () => getVehicle(vin).then((r) => r.data),
    enabled: !!vin,
    refetchOnWindowFocus: false,
    retry: false,
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-bg-base flex items-center justify-center">
        <div className="text-center">
          <div className="w-12 h-12 border-2 border-accent/20 border-t-accent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-txt-secondary text-sm font-medium">
            Decoding VIN…
          </p>
          <p className="font-mono text-xs text-txt-muted mt-1.5 tracking-widest">
            {vin}
          </p>
        </div>
      </div>
    );
  }

  if (isError) {
    const errMsg =
      error?.response?.data?.error ??
      error?.message ??
      "An unexpected error occurred";
    const isNetwork = !error?.response;
    return (
      <div className="min-h-screen bg-bg-base flex flex-col">
        <header className="border-b border-border-subtle px-6 h-14 flex items-center">
          <button
            onClick={handleBack}
            className="flex items-center gap-2 text-sm text-txt-muted hover:text-txt-primary transition-colors"
          >
            <ArrowLeft className="w-4 h-4" /> Back
          </button>
        </header>
        <div className="flex-1 flex items-center justify-center px-4 py-12">
          <div className="w-full max-w-md">
            <div className="bg-bg-card border border-danger/20 rounded-2xl p-8 shadow-card text-center">
              <div className="w-16 h-16 rounded-2xl bg-danger/10 border border-danger/20 flex items-center justify-center mx-auto mb-5">
                {isNetwork ? (
                  <WifiOff className="w-8 h-8 text-danger/80" />
                ) : (
                  <AlertCircle className="w-8 h-8 text-danger/80" />
                )}
              </div>
              <h2 className="text-xl font-bold text-txt-primary mb-2">
                Decode Failed
              </h2>
              <div className="inline-flex items-center gap-2 bg-bg-elevated border border-border-subtle rounded-lg px-3 py-1.5 mb-4">
                <Fingerprint className="w-3.5 h-3.5 text-accent" />
                <span className="font-mono text-xs text-txt-muted tracking-widest">
                  {vin}
                </span>
              </div>
              <div className="bg-bg-elevated border border-border-subtle rounded-xl px-4 py-3 mb-6 text-left">
                <p className="text-xs font-semibold text-txt-muted uppercase tracking-wider mb-1">
                  {isNetwork ? "Network Error" : "Server Error"}
                </p>
                <p className="text-sm text-txt-secondary leading-relaxed">
                  {errMsg}
                </p>
              </div>
              <div className="flex gap-3">
                <button
                  onClick={handleBack}
                  className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-bg-elevated border border-border-subtle rounded-xl text-sm text-txt-primary hover:border-border transition-all"
                >
                  <ArrowLeft className="w-4 h-4" /> Go back
                </button>
                <button
                  onClick={() => refetch()}
                  className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-accent/10 border border-accent/30 rounded-xl text-sm text-accent hover:bg-accent/20 transition-all"
                >
                  <RefreshCw className="w-4 h-4" /> Retry
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  const title = `${vehicle.year} ${vehicle.make} ${vehicle.model}`;

  return (
    <div className="min-h-screen bg-bg-base">
      {}
      <header className="sticky top-0 z-40 bg-bg-base/90 backdrop-blur-md border-b border-border-subtle">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 h-14 flex items-center gap-3">
          <button
            onClick={handleBack}
            className="flex items-center gap-1.5 text-txt-muted hover:text-txt-primary text-sm transition-colors shrink-0"
          >
            <ArrowLeft className="w-4 h-4" />
            <span className="hidden sm:inline">Back</span>
          </button>
          <div className="h-4 w-px bg-border-subtle" />
          <Fingerprint className="w-3.5 h-3.5 text-accent shrink-0" />
          <span className="font-mono text-xs text-txt-muted tracking-widest hidden md:block">
            {vin}
          </span>
          <CopyBtn text={vin} label="VIN" />
          <div className="flex-1 min-w-0 ml-1">
            <p className="text-sm font-semibold text-txt-primary truncate">
              {title}
            </p>
          </div>
          <QuickVinSearch />
          <ThemeToggle />
        </div>
      </header>

      {}
      <div className="max-w-screen-xl mx-auto px-4 sm:px-6 py-7">
        {/* Hero */}
        <div className="mb-7 animate-slide-up">
          <p className="text-xs font-semibold text-txt-muted uppercase tracking-widest mb-1">
            {vehicle.make}
          </p>
          <h1 className="text-4xl sm:text-5xl font-extrabold text-txt-primary leading-none">
            {vehicle.year}&nbsp;
            <span className="text-txt-secondary font-bold">
              {vehicle.model}
            </span>
          </h1>
          {(vehicle.trim || vehicle.series) && (
            <p className="mt-2 text-txt-secondary">
              {[vehicle.trim, vehicle.series].filter(Boolean).join(" · ")}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2.5 mt-5">
            <HeroBadge label={vehicle.body_type} icon={Car} />
            {vehicle.doors && (
              <HeroBadge label={`${vehicle.doors}-door`} icon={DoorOpen} />
            )}
            <HeroBadge label={vehicle.fuel_type} variant="fuel" icon={Fuel} />
            <HeroBadge label={vehicle.drive_type} variant="drive" icon={Cog} />
            {vehicle.cylinders > 0 && (
              <HeroBadge
                label={`${vehicle.cylinders}-cyl`}
                variant="engine"
                icon={Gauge}
              />
            )}
            {vehicle.displacement_l > 0 && (
              <HeroBadge
                label={`${vehicle.displacement_l}L`}
                variant="engine"
                icon={Activity}
              />
            )}
          </div>
          {/* VIN chip with copy */}
          <div className="mt-4 inline-flex items-center gap-2 bg-bg-card border border-border-subtle rounded-lg px-3 py-1.5">
            <Fingerprint className="w-3.5 h-3.5 text-accent" />
            <span className="font-mono text-xs text-txt-muted tracking-widest">
              {vin}
            </span>
            <CopyBtn text={vin} label="VIN" />
          </div>
        </div>

        {/* Two-column grid */}
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_420px] gap-5">
          {}
          <div>
            {/* Expand / Collapse all bar */}
            <div className="flex items-center justify-between mb-3">
              <span className="text-[13px] font-semibold text-txt-secondary uppercase tracking-wider">
                Specifications
              </span>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => setExpandAll(true)}
                  className="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-txt-muted hover:text-txt-primary rounded-lg hover:bg-bg-elevated transition-all"
                >
                  <ChevronsUpDown className="w-3.5 h-3.5" />
                  Expand all
                </button>
                <span className="text-border opacity-60 select-none">·</span>
                <button
                  onClick={() => setExpandAll(false)}
                  className="px-2.5 py-1.5 text-xs font-medium text-txt-muted hover:text-txt-primary rounded-lg hover:bg-bg-elevated transition-all"
                >
                  Collapse all
                </button>
              </div>
            </div>

            <div className="space-y-3">
              {SPEC_SECTIONS.map((section) => (
                <SpecSection
                  key={section.id}
                  section={section}
                  vehicle={vehicle}
                  vin={vin}
                  expandAll={expandAll}
                />
              ))}
              <BuildNumberSpecs vehicle={vehicle} pageVin={vin} />
              <CustomFieldsSection
                customFields={vehicle.custom_fields}
                vin={vin}
              />
              <GMLiveSection vehicle={vehicle} activeVin={vin} />
            </div>
          </div>

          {/* Notes + Compatible Parts column */}
          <div className="xl:sticky xl:top-[3.75rem] xl:self-start xl:max-h-[calc(100vh-5rem)] xl:overflow-y-auto pb-4 space-y-4">
            <NotesPanel vehicle={vehicle} vin={vin} />
            {/* <CompatibleParts vin={vin} />*/}
          </div>
        </div>
      </div>
    </div>
  );
}
