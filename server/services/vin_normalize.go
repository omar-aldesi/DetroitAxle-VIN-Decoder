package services

import "strings"

func normalizeDriveType(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch {
	case strings.Contains(s, "4WD") || strings.Contains(s, "4X4") ||
		(strings.Contains(s, "FOUR") && strings.Contains(s, "WHEEL")):
		return "4WD"
	case strings.Contains(s, "AWD") ||
		(strings.Contains(s, "ALL") && strings.Contains(s, "WHEEL")):
		return "AWD"
	case strings.Contains(s, "FWD") ||
		(strings.Contains(s, "FRONT") && strings.Contains(s, "WHEEL")):
		return "FWD"
	case strings.Contains(s, "RWD") ||
		(strings.Contains(s, "REAR") && strings.Contains(s, "WHEEL")):
		return "RWD"
	default:
		return s
	}
}

func normalizeTransmission(s string) string {
	upper := strings.ToUpper(strings.TrimSpace(s))
	switch {
	case strings.Contains(upper, "MANUAL"):
		return "Manual"
	case strings.Contains(upper, "CVT") || strings.Contains(upper, "CONTINUOUSLY"):
		return "CVT"
	case strings.Contains(upper, "DUAL") || strings.Contains(upper, "DCT") || strings.Contains(upper, "PDK"):
		return "DCT"
	case strings.Contains(upper, "AUTOMATIC"):
		return "Automatic"
	default:
		return strings.TrimSpace(s)
	}
}

// normalizeNHTSAFuelType maps NHTSA's FuelTypePrimary+Secondary to our standard labels.
func normalizeNHTSAFuelType(primary, secondary string) string {
	combined := strings.ToUpper(primary + " " + secondary)
	switch {
	case strings.Contains(combined, "FFV") || strings.Contains(combined, "FLEX") ||
		strings.Contains(combined, "E85"):
		return "Flex Fuel"
	case strings.Contains(combined, "DIESEL"):
		return "Diesel"
	case strings.Contains(combined, "ELECTRIC") && strings.Contains(combined, "GAS"):
		return "Hybrid"
	case strings.Contains(combined, "ELECTRIC") || strings.Contains(combined, "BEV"):
		return "Electric"
	case strings.Contains(combined, "NATURAL GAS") || strings.Contains(combined, "CNG"):
		return "Natural Gas"
	case strings.Contains(combined, "HYDROGEN"):
		return "Hydrogen"
	case strings.Contains(combined, "GASOLINE") || strings.Contains(combined, "GAS"):
		return "Gasoline"
	default:
		return ""
	}
}

// cleanNHTSACountry strips the abbreviation in parentheses and title-cases.
// Example: "UNITED STATES (USA)" → "United States"
func cleanNHTSACountry(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "("); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.Title(strings.ToLower(s))
}
