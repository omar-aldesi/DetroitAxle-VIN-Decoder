import { Car, Cpu, GitFork, Disc } from "lucide-react";

export const SECTIONS = [
  {
    id: "identity",
    label: "Identity",
    Icon: Car,
    color: "#4f8ef7",
    fields: [
      { key: "trim", label: "Trim", type: "text" },
      { key: "series", label: "Series", type: "text" },
      { key: "body_type", label: "Body Type", type: "text" },
      { key: "doors", label: "Doors", type: "text", placeholder: "2, 4" },
      {
        key: "drive_type",
        label: "Drive Type",
        type: "select",
        options: ["FWD", "RWD", "AWD", "4WD"],
      },
      { key: "country", label: "Country", type: "text" },
    ],
  },
  {
    id: "engine",
    label: "Engine",
    Icon: Cpu,
    color: "#f59e0b",
    fields: [
      { key: "cylinders", label: "Cylinders", type: "text", placeholder: "8" },
      {
        key: "displacement_l",
        label: "Displacement (L)",
        type: "text",
        placeholder: "5.3",
      },
      {
        key: "fuel_type",
        label: "Fuel Type",
        type: "select",
        options: ["Gasoline", "Diesel", "Electric", "Hybrid", "Flex Fuel"],
      },
      {
        key: "engine_configuration",
        label: "Configuration",
        type: "text",
        placeholder: "V-type, In-line",
      },
    ],
  },
  {
    id: "transmission",
    label: "Transmission",
    Icon: GitFork,
    color: "#8b5cf6",
    fields: [
      {
        key: "transmission_type",
        label: "Type",
        type: "select",
        options: ["Automatic", "Manual", "CVT", "DCT"],
      },
      { key: "speeds", label: "Speeds", type: "text", placeholder: "6, 8, 10" },
    ],
  },
  {
    id: "brakes",
    label: "Brakes",
    Icon: Disc,
    color: "#ef4444",
    fields: [
      {
        key: "abs",
        label: "ABS",
        type: "select",
        options: ["4-Wheel ABS", "2-Wheel ABS", "None"],
      },
      { key: "brake_system_type", label: "System Type", type: "text" },
      {
        key: "front_brake_type",
        label: "Front Type",
        type: "select",
        options: ["Disc", "Drum"],
      },
      {
        key: "rear_brake_type",
        label: "Rear Type",
        type: "select",
        options: ["Disc", "Drum"],
      },
      {
        key: "gvwr_lbs",
        label: "GVWR (lbs)",
        type: "text",
        placeholder: "7200",
      },
    ],
  },
];

/* Build-number-tier fields — entered per VIN / per range via the Build-Number panel,
   not as shared column edits. Keys match the Vehicle columns / fork registry. */
export const FORK_FIELDS = [
  { key: "brake_code", label: "Brake Code", placeholder: "JL9, J55…" },
  { key: "front_rotor_size", label: "Front Rotor", placeholder: "325mm" },
  { key: "rear_rotor_size", label: "Rear Rotor", placeholder: "298mm" },
  {
    key: "front_spring_type",
    label: "Front Suspension",
    placeholder: "Coil, Torsion Bar",
  },
  {
    key: "rear_spring_type",
    label: "Rear Suspension",
    placeholder: "Coil, Leaf",
  },
  { key: "steering_type", label: "Steering", placeholder: "Rack & Pinion" },
];
export const FORK_LABEL = Object.fromEntries(FORK_FIELDS.map((f) => [f.key, f.label]));

export function primaryKnownVin(vehicle) {
  if (!vehicle) return "";
  const known = vehicle.known_vins ?? [];
  if (known.length > 0) return known[0];
  return vehicle.viewed_vin || vehicle.example_build_number || "";
}

export const ALL_KEYS = SECTIONS.flatMap((s) => s.fields.map((f) => f.key));
export const TOTAL = ALL_KEYS.length;

// Keys by category — used for missing-category dots in queue
export const CAT_KEYS = {
  brakes:
    SECTIONS.find((s) => s.id === "brakes")?.fields.map((f) => f.key) ?? [],
  engine:
    SECTIONS.find((s) => s.id === "engine")?.fields.map((f) => f.key) ?? [],
  transmission:
    SECTIONS.find((s) => s.id === "transmission")?.fields.map((f) => f.key) ??
    [],
};
export const CAT_META = [
  { id: "brakes", color: "#ef4444", label: "Brakes" },
  { id: "engine", color: "#f59e0b", label: "Engine" },
  { id: "transmission", color: "#8b5cf6", label: "Transmission" },
];

export const MISSING_OPTS = [
  { value: "", label: "All vehicles" },
  { value: "brakes", label: "Missing brakes" },
  { value: "engine", label: "Missing engine" },
  { value: "transmission", label: "Missing transmission" },
];
export const FUEL_OPTS = [
  "",
  "Gasoline",
  "Diesel",
  "Electric",
  "Hybrid",
  "Flex Fuel",
];
export const DRIVE_OPTS = ["", "FWD", "RWD", "AWD", "4WD"];
export const TRANS_OPTS = ["", "Automatic", "Manual", "CVT", "DCT"];
