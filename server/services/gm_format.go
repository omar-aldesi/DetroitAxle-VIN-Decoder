package services

import (
	"regexp"
	"strings"
)

// --- Live formatter ---

// rpoCleanRe matches a spec description that is ONLY a routing code with no
// human-readable text, e.g. "TL1-", "L-", "0495-", "2014-". These are GM's
// internal catalog metadata and aren't useful to display. The trailing dash is
// REQUIRED so we never drop a legitimate single-word description like "STANDARD".
var rpoCleanRe = regexp.MustCompile(`^[A-Z0-9]+-$`)

// FormatAsJSON returns GM Parts Giant's full native structure for this VIN,
// ready to be JSON-serialised in an HTTP response. Nothing is dropped except
// pure catalog-routing metadata rows; every real RPO / build-option code is
// preserved so the team can see the exact per-VIN equipment.
func (a *GMVINAttributes) FormatAsJSON() map[string]any {
	if a == nil || len(a.VinInfos) == 0 {
		return nil
	}
	info := a.primaryVinInfo()
	if info == nil {
		return nil
	}

	out := map[string]any{
		"vehicle_information": cleanNameDescList(info.VehicleInformation, false),
		"major_attributes":    cleanNameDescList(info.MajorAttribute, false),
		"specifications":      cleanNameDescList(info.Specification, true),
	}
	if v := strings.TrimSpace(info.VehicleInfo); v != "" {
		out["vehicle_info"] = v
	}
	if v := strings.TrimSpace(info.RedirectURL); v != "" {
		out["redirect_url"] = v
	}
	if v := strings.TrimSpace(info.RequiredInfo); v != "" {
		out["required_info"] = v
	}
	if v := strings.TrimSpace(info.OptionalInfo); v != "" {
		out["optional_info"] = v
	}
	return out
}

// cleanNameDescList trims entries and (when dropMeta is true) removes rows whose
// description is just a routing code with no descriptive text.
func cleanNameDescList(items []gmNameDesc, dropMeta bool) []map[string]string {
	out := make([]map[string]string, 0, len(items))
	for _, it := range items {
		name := strings.TrimSpace(it.Name)
		desc := strings.TrimSpace(it.Desc)
		if desc == "" {
			continue
		}
		if dropMeta && rpoCleanRe.MatchString(desc) {
			continue // pure metadata like "TL1-", "2014-"
		}
		out = append(out, map[string]string{"name": name, "desc": desc})
	}
	return out
}

// primaryVinInfo returns the first vinInfo that has specification data.
func (a *GMVINAttributes) primaryVinInfo() *gmVinInfo {
	for i := range a.VinInfos {
		if len(a.VinInfos[i].Specification) > 0 {
			return &a.VinInfos[i]
		}
	}
	if len(a.VinInfos) > 0 {
		return &a.VinInfos[0]
	}
	return nil
}
