import { ALL_KEYS, TOTAL } from "./constants";

export function isFilled(val) {
  return val !== null && val !== undefined && val !== "" && Number(val) !== 0;
}
export function pctOf(v) {
  if (!v) return 0;
  return Math.round(
    (ALL_KEYS.filter((k) => isFilled(v[k])).length / TOTAL) * 100,
  );
}
export function pctColor(p) {
  if (p >= 90) return "#10b981";
  if (p >= 60) return "#4f8ef7";
  if (p >= 30) return "#f59e0b";
  return "#ef4444";
}
