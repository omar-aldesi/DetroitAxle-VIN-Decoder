import { Car, Cpu, GitFork, Disc } from "lucide-react";

export const SPEC_SECTIONS = [
  {
    id: "identity",
    label: "Identity",
    Icon: Car,
    iconCls: "text-accent",
    defaultOpen: true,
    fields: [
      { label: "Year", jsonKey: "year", col: "year" },
      { label: "Make", jsonKey: "make", col: "make" },
      { label: "Model", jsonKey: "model", col: "model" },
      { label: "Trim", jsonKey: "trim", col: "trim" },
      { label: "Series", jsonKey: "series", col: "series" },
      { label: "Body Type", jsonKey: "body_type", col: "body_type" },
      { label: "Drive Type", jsonKey: "drive_type", col: "drive_type" },
      { label: "Country", jsonKey: "country", col: "country" },
    ],
  },
  {
    id: "engine",
    label: "Engine",
    Icon: Cpu,
    iconCls: "text-amber-400",
    defaultOpen: false,
    fields: [
      { label: "Cylinders", jsonKey: "cylinders", col: "cylinders" },
      {
        label: "Displacement (L)",
        jsonKey: "displacement_l",
        col: "displacement_l",
      },
      { label: "Fuel Type", jsonKey: "fuel_type", col: "fuel_type" },
    ],
  },
  {
    id: "transmission",
    label: "Transmission",
    Icon: GitFork,
    iconCls: "text-purple-400",
    defaultOpen: false,
    fields: [
      { label: "Type", jsonKey: "transmission_type", col: "transmission_type" },
      { label: "Speeds", jsonKey: "speeds", col: "speeds" },
    ],
  },
  {
    id: "brakes",
    label: "Brakes",
    Icon: Disc,
    iconCls: "text-red-400",
    defaultOpen: false,
    fields: [
      { label: "ABS", jsonKey: "abs", col: "abs" },
      {
        label: "Front Brake Type",
        jsonKey: "front_brake_type",
        col: "front_brake_type",
      },
      {
        label: "Rear Brake Type",
        jsonKey: "rear_brake_type",
        col: "rear_brake_type",
      },
      { label: "GVWR (lbs)", jsonKey: "gvwr_lbs", col: "gvwr_lbs" },
    ],
  },
];

/* Build-number-tier fields — these vary by individual VIN within the build group,
   so they live in the BuildNumberSpecs panel (resolved from the fork engine), not
   the shared spec sections above. Keys match the Vehicle columns / fork registry. */
export const FORK_FIELDS = [
  { key: "brake_code", label: "Brake Code", mono: true },
  { key: "front_rotor_size", label: "Front Rotor" },
  { key: "rear_rotor_size", label: "Rear Rotor" },
  { key: "front_spring_type", label: "Front Suspension" },
  { key: "rear_spring_type", label: "Rear Suspension" },
  { key: "steering_type", label: "Steering" },
];

/* Friendly confidence labels — agents shouldn't have to learn engine jargon. */
export const FORK_CONFIDENCE = {
  observed: {
    label: "Confirmed",
    color: "#10b981",
    desc: "Seen on two or more VINs across this range",
  },
  manual: {
    label: "Set by team",
    color: "#4f8ef7",
    desc: "Entered as a known range by the DNR team",
  },
  assumed: {
    label: "Assumed",
    color: "#f59e0b",
    desc: "Inferred between known VINs — verify if critical",
  },
  extrapolated: {
    label: "Estimated",
    color: "#f97316",
    desc: "Beyond the VINs we've confirmed — treat with caution",
  },
};
