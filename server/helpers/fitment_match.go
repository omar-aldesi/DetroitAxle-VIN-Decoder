package helpers

import (
	"math"
	"strconv"
	"strings"
)

func ciEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// normModelStr strips all non-alphanumeric characters so that "F-350" and
// "F350" produce the same token, then lowercases.
// "F-350 Super Duty" → "f350 super duty"
// "Super Duty F-350 DRW" → "super duty f350 drw"

func normModelStr(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune(' ')
			// hyphens, slashes, dots → nothing (join the surrounding chars)
		}
	}
	// collapse multiple spaces
	return strings.Join(strings.Fields(b.String()), " ")
}

// containsWord checks that needle appears as a whole space-delimited word
// in haystack, preventing "f35" from matching "f350".

func containsWord(haystack, needle string) bool {
	haystack = " " + haystack + " "
	return strings.Contains(haystack, " "+needle+" ")
}

// matchesModelTokens returns true when every space-delimited token of
// ruleVal appears as a whole word in vehicleVal (after normModelStr).
// "F-350 Super Duty" matches "Super Duty F-350 DRW" because the tokens
// ["f350","super","duty"] all appear word-for-word in the vehicle string.

func matchesModelTokens(vehicleVal, ruleVal string) bool {
	if ruleVal == "" {
		return true
	}
	vNorm := normModelStr(vehicleVal)
	for _, tok := range strings.Fields(normModelStr(ruleVal)) {
		if tok == "" {
			continue
		}
		if !containsWord(vNorm, tok) {
			return false
		}
	}
	return true
}

// matchesTrimList handles a comma-separated list of acceptable trim values.
// "King Ranch, Lariat, XL, XLT" matches a vehicle whose trim contains "XLT".
// Matching is token-based (same as matchesModelTokens) per list entry.

func matchesTrimList(vehicleTrim, ruleTrim string) bool {
	if ruleTrim == "" {
		return true
	}
	vNorm := normModelStr(vehicleTrim)
	for _, opt := range strings.Split(ruleTrim, ",") {
		opt = strings.TrimSpace(opt)
		if opt == "" {
			continue
		}
		allMatch := true
		for _, tok := range strings.Fields(normModelStr(opt)) {
			if !containsWord(vNorm, tok) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// normDriveType maps common drive-type strings to a canonical short form.
// Handles both human-entered values ("4WD") and decoded strings from
// auto.dev/NHTSA ("FOUR WHEEL DRIVE", "FOUR-WHEEL DRIVE", "4x4", etc.).

func normDriveType(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch {
	case strings.Contains(s, "4WD") || strings.Contains(s, "4X4") ||
		strings.Contains(s, "FOUR") && strings.Contains(s, "WHEEL"):
		return "4WD"
	case strings.Contains(s, "AWD") ||
		strings.Contains(s, "ALL") && strings.Contains(s, "WHEEL"):
		return "AWD"
	case strings.Contains(s, "FWD") ||
		strings.Contains(s, "FRONT") && strings.Contains(s, "WHEEL"):
		return "FWD"
	case strings.Contains(s, "RWD") ||
		strings.Contains(s, "REAR") && strings.Contains(s, "WHEEL"):
		return "RWD"
	default:
		return s
	}
}

func displMatches(vehicleDispl, ruleDispl string) bool {
	if ruleDispl == "" {
		return true // any displacement
	}
	if vehicleDispl == "" {
		return false // rule requires displacement, vehicle has none
	}
	rv, errR := strconv.ParseFloat(strings.TrimSpace(ruleDispl), 64)
	vv, errV := strconv.ParseFloat(strings.TrimSpace(vehicleDispl), 64)
	if errR != nil || errV != nil {
		return ciEqual(vehicleDispl, ruleDispl)
	}
	// Round to one decimal to avoid float precision drift (3.5000000001 == 3.5)
	return math.Round(rv*10) == math.Round(vv*10)
}

// EvaluateRule evaluates a single PartFitmentRule against a Vehicle.
//
// Algorithm:
//  1. All non-empty required fields are checked; any failure → FitNone.
//  2. Callouts are checked:
//     - field value matches → ok
//     - field exists but wrong value → FitNone (explicit mismatch)
//     - field missing from vehicle → FitWithNote (can't verify)
//  3. If all required pass and no callout mismatch:
//     - any unverified callout → FitWithNote with notes
//     - otherwise → FitExact
