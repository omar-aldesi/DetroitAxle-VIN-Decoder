export const GM_WMI2 = ["1G", "2G", "3G"];
export const GM_WMI3 = ["KL4", "KL8", "KL1", "W0L"];
export const GM_MAKES = new Set([
  "chevrolet",
  "gmc",
  "buick",
  "cadillac",
  "pontiac",
  "saturn",
  "oldsmobile",
  "hummer",
  "opel",
  "vauxhall",
]);

export function isGMVehicle(vehicle, activeVin) {
  if (GM_MAKES.has((vehicle.make || "").toLowerCase())) return true;
  const vin = (
    activeVin ||
    vehicle.viewed_vin ||
    vehicle.known_vins?.[0] ||
    vehicle.example_build_number ||
    ""
  ).toUpperCase();
  if (vin.length >= 3) {
    if (GM_WMI2.some((p) => vin.startsWith(p))) return true;
    if (GM_WMI3.some((p) => vin.startsWith(p))) return true;
  }
  return false;
}

// Title-case alphabetic runs; keep numbers, slashes, punctuation intact.
// Also strips a trailing dash/space left over from codes like "PACKAGE OPTION-".
export function formatGMText(raw) {
  return String(raw)
    .replace(
      /[A-Za-z]+/g,
      (w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase(),
    )
    .replace(/[\s-]+$/, "")
    .trim();
}

// Friendly labels for the GM summary rows (major attributes + vehicle info).
export const GM_FRIENDLY = {
  Productiondate: "Production Date",
  CatalogCode: "Catalog Code",
  MakeCode: "Make Code",
  ModelCode: "Model Code",
  Vehicle: "Vehicle",
  Engine: "Engine",
  "Model String": "Model String",
  Transmission: "Transmission",
};

// Split a GM spec description into individual RPO entries.
//   "AE8-ADJUSTER FRT ST POWER, 8 WAY"  -> [{ code:"AE8", text:"Adjuster Frt St Power, 8 Way" }]
//   "1SZ-PACKAGE OPTION-;PCW-CONTROL…"  -> two entries (semicolon-separated)
export function parseRPO(desc) {
  return String(desc)
    .split(";")
    .map((seg) => seg.trim())
    .filter(Boolean)
    .map((seg) => {
      const m = seg.match(/^([A-Z0-9]{2,4})-(.*)$/);
      if (m) return { code: m[1], text: formatGMText(m[2]) };
      return { code: null, text: formatGMText(seg) };
    });
}

export function gmVinFor(vehicle, activeVin) {
  if ((activeVin ?? "").length === 17) return activeVin.toUpperCase();
  return (
    vehicle.viewed_vin ||
    vehicle.known_vins?.[0] ||
    vehicle.example_build_number ||
    ""
  );
}
