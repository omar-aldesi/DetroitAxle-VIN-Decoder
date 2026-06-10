package services

import "strings"

// parseEngineString extracts cylinders, displacement, and fuel type from
// auto.dev's freeform engine string. Handles both common formats:
//
//	"5.3L V8 OHV 16V FFV"        (displacement+L, V8)
//	"6.0, 8 Cylinder Engine"     (bare float, "N Cylinder")
//
// Any field this can't parse is left empty for NHTSA to fill.
func parseEngineString(s string) (cylinders, displacement, fuelType string) {
	upper := strings.ToUpper(strings.TrimSpace(s))
	if upper == "" {
		return
	}
	fields := strings.Fields(upper)

	// Displacement, pass 1: a token ending in "L" whose stem is numeric ("5.3L").
	for _, p := range fields {
		if strings.HasSuffix(p, "L") {
			stem := strings.TrimSuffix(p, "L")
			if isNumericFloat(stem) {
				displacement = stem
				break
			}
		}
	}
	// Displacement, pass 2: a bare decimal token ("6.0", "6.0,"). Must contain a
	// dot so we never mistake the cylinder count ("8") for displacement.
	if displacement == "" {
		for _, p := range fields {
			stem := strings.Trim(p, ",;")
			if strings.Contains(stem, ".") && isNumericFloat(stem) {
				displacement = stem
				break
			}
		}
	}

	// Cylinders, pass 1: "V8", "I4", "H6", "W12".
	for _, p := range fields {
		if len(p) >= 2 {
			switch p[0] {
			case 'V', 'I', 'H', 'W':
				if rest := p[1:]; isAllDigits(rest) {
					cylinders = rest
				}
			}
		}
		if cylinders != "" {
			break
		}
	}
	// Cylinders, pass 2: "8 CYLINDER" — the digit token immediately before "CYLINDER".
	if cylinders == "" {
		for i, p := range fields {
			if strings.HasPrefix(p, "CYLINDER") && i > 0 {
				if prev := strings.Trim(fields[i-1], ",;"); isAllDigits(prev) {
					cylinders = prev
					break
				}
			}
		}
	}

	// Fuel type from engine string keywords (rough pass; NHTSA refines later).
	switch {
	case strings.Contains(upper, "FFV") || strings.Contains(upper, "FLEX"):
		fuelType = "Flex Fuel"
	case strings.Contains(upper, "DIESEL"):
		fuelType = "Diesel"
	case strings.Contains(upper, "ELECTRIC"):
		fuelType = "Electric"
	case strings.Contains(upper, "HYBRID"):
		fuelType = "Hybrid"
	default:
		fuelType = "Gasoline"
	}

	return
}

// isNumericFloat reports whether s is a non-empty run of digits with at most one dot.
func isNumericFloat(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// isAllDigits reports whether s is a non-empty run of digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
