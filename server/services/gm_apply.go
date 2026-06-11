package services

import (
	"errors"
	"main/models"
	"strings"

	"gorm.io/gorm"
)

// --- Persisted enrichment (build-key-stable fields only) ---

// ApplyStableFields writes ONLY the GM values that are encoded in the VIN's
// VDS (positions 4–8) — i.e. provably identical for every vehicle that shares
// this build key. These come from GM's majorAttribute section, never from the
// per-VIN `specification` RPO list.
//
// Hard rules (so a sibling vehicle sharing the build key is never corrupted):
//   - Only VDS-encoded attributes: series, trim, and engine (cylinders/
//     displacement/fuel). Transmission is intentionally EXCLUDED — it is not
//     encoded in the VDS for GM, so it can differ between two units with the
//     same build key.
//   - Fill-only-when-empty: GM never overwrites a value auto.dev / NHTSA / a
//     human already set.
//   - NEVER touch brake code, brake/suspension/steering types, GVWR, country,
//     speeds, or any individual RPO option — those vary per unit (live-only).
func (a *GMVINAttributes) ApplyStableFields(v *models.Vehicle) {
	if a == nil {
		return
	}
	info := a.primaryVinInfo()
	if info == nil {
		return
	}
	major := gmToMap(info.MajorAttribute)

	modelString := major["Model String"]

	// Series — from "Model String" (e.g. "Silverado 1500 Crew Cab" → "1500").
	// A different series is a different VDS, so this is build-key-stable. GM is
	// often the only source, since auto.dev/NHTSA frequently omit it for trucks.
	if v.Series == "" {
		if s := parseGMSeries(modelString); s != "" {
			v.Series = s
		}
	}

	// Trim — GM is the most accurate source for trim level. It wraps the trim in
	// single quotes in the model string (e.g. "Terrain 'SLT' SUV" → "SLT"). For
	// GM the trim is generally encoded in the VDS, so it is build-key-stable.
	// Fill-only-when-empty: we never overwrite a trim auto.dev/NHTSA/a human set.
	if v.Trim == "" {
		if t := parseGMTrim(modelString); t != "" {
			v.Trim = t
		}
	}

	// Engine-derived: cylinders / displacement (VDS-encoded) + fuel hint.
	cyl, disp, fuel := parseGMEngine(major["Engine"])
	if v.Cylinders == "" && cyl != "" {
		v.Cylinders = cyl
	}
	if v.DisplacementL == "" && disp != "" {
		v.DisplacementL = disp
	}
	// Fuel: only upgrade the generic "Gasoline"/empty default (GM "E85 MAX" etc.).
	if fuel != "" && (v.FuelType == "" || v.FuelType == "Gasoline") {
		v.FuelType = fuel
	}
}

// EnrichExistingWithGM backfills GM build-key-stable fields onto an
// already-persisted vehicle (a cache hit on search, or an import that skipped
// an existing record) and marks it gm_checked so we only ever do this once.
//
// It is a no-op when the vehicle is already checked, or when vin isn't a
// GM-brand 17-char VIN. Like ApplyStableFields it only FILLS empty fields, so a
// human's edits and a sibling's values are never overwritten.
func EnrichExistingWithGM(db *gorm.DB, vin string, v *models.Vehicle) error {
	vin = strings.TrimSpace(strings.ToUpper(vin))
	if v.GMChecked || len(vin) != 17 || !IsGMBrandVIN(vin) {
		return nil
	}

	attrs, err := FetchGMAttributes(vin)
	if err != nil {
		if errors.Is(err, ErrGMNoData) {
			// GM has no data for this VIN — mark checked so we don't retry.
			v.GMChecked = true
			return db.Model(&models.Vehicle{}).
				Where("build_key = ?", v.BuildKey).
				Update("gm_checked", true).Error
		}
		return err // transient — leave unchecked so a later access retries
	}

	attrs.ApplyStableFields(v)
	v.GMChecked = true

	return db.Model(&models.Vehicle{}).
		Where("build_key = ?", v.BuildKey).
		Updates(map[string]any{
			"trim":           v.Trim,
			"series":         v.Series,
			"cylinders":      v.Cylinders,
			"displacement_l": v.DisplacementL,
			"fuel_type":      v.FuelType,
			"gm_checked":     true,
		}).Error
}

// gmToMap converts a []gmNameDesc into a name→desc map, trimming both sides so
// stray whitespace in the API response never breaks a lookup like major["Engine"].
func gmToMap(items []gmNameDesc) map[string]string {
	m := make(map[string]string, len(items))
	for _, it := range items {
		m[strings.TrimSpace(it.Name)] = strings.TrimSpace(it.Desc)
	}
	return m
}

// parseGMTrim extracts the trim from a GM model string. GM wraps the trim in
// single quotes, e.g. "Terrain 'SLT' SUV" → "SLT". Returns "" when absent.
func parseGMTrim(modelString string) string {
	i := strings.Index(modelString, "'")
	if i < 0 {
		return ""
	}
	rest := modelString[i+1:]
	j := strings.Index(rest, "'")
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

// parseGMSeries extracts the numeric series designation from a model string.
// "Silverado 1500 Crew Cab" → "1500"; "Sierra 2500HD" → "2500HD".
func parseGMSeries(modelString string) string {
	for _, tok := range strings.Fields(modelString) {
		base := strings.TrimSuffix(strings.TrimSuffix(strings.ToUpper(tok), "HD"), "LD")
		if len(base) >= 3 && len(base) <= 4 && isAllDigits(base) {
			return strings.Trim(tok, "'\"")
		}
	}
	return ""
}

// parseGMEngine pulls cylinders, displacement, and a fuel hint from a GM engine
// string, e.g. "4 Cyl 2.4L SIDI, DOHC, E85 MAX, ALUM".
func parseGMEngine(desc string) (cylinders, displacement, fuelType string) {
	upper := strings.ToUpper(desc)
	for _, tok := range strings.Fields(upper) {
		t := strings.Trim(tok, ",;")
		// Displacement: "2.4L" or bare "2.4".
		if displacement == "" {
			stem := strings.TrimSuffix(t, "L")
			if strings.Contains(stem, ".") && isNumericFloat(stem) {
				displacement = stem
			}
		}
	}
	// Cylinders: "<n> CYL" / "<n> CYLINDER".
	fields := strings.Fields(upper)
	for i, tok := range fields {
		if strings.HasPrefix(tok, "CYL") && i > 0 {
			if prev := strings.Trim(fields[i-1], ",;"); isAllDigits(prev) {
				cylinders = prev
				break
			}
		}
	}
	switch {
	case strings.Contains(upper, "E85") || strings.Contains(upper, "FLEX") || strings.Contains(upper, "FFV"):
		fuelType = "Flex Fuel"
	case strings.Contains(upper, "DIESEL"):
		fuelType = "Diesel"
	case strings.Contains(upper, "ELECTRIC"):
		fuelType = "Electric"
	case strings.Contains(upper, "HYBRID"):
		fuelType = "Hybrid"
	}
	return
}
