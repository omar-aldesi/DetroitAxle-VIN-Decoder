package helpers

import (
	"encoding/json"
	"fmt"
	"main/models"
	"strings"
)

func EvaluateRule(vehicle models.Vehicle, rule models.PartFitmentRule) EvalResult {
	none := EvalResult{FitNone, nil, ""}

	// ── Required: year range ──────────────────────────────────────
	if rule.YearMin != nil && vehicle.Year < *rule.YearMin {
		return none
	}
	if rule.YearMax != nil && vehicle.Year > *rule.YearMax {
		return none
	}

	// ── Required: make (exact, case-insensitive) ──────────────────
	if rule.Make != "" && !ciEqual(vehicle.Make, rule.Make) {
		return none
	}

	// ── Required: model — token-based, order-independent ────────
	// "F-350 Super Duty" matches "Super Duty F-350 DRW" because
	// all tokens ["f350","super","duty"] appear as whole words.
	if !matchesModelTokens(vehicle.Model, rule.VehicleModel) {
		return none
	}

	// ── Required: trim — comma-separated list of valid values ────
	// "King Ranch, Lariat, XL, XLT" matches a vehicle trim containing "XLT".
	if !matchesTrimList(vehicle.Trim, rule.Trim) {
		return none
	}

	// ── Required: cylinders (exact, case-insensitive) ────────────
	if rule.Cylinders != "" && !ciEqual(vehicle.Cylinders, rule.Cylinders) {
		return none
	}

	// ── Required: displacement — numeric so "3.5" == "3.50" ──────
	if !displMatches(vehicle.DisplacementL, rule.DisplacementL) {
		return none
	}

	// ── Required: fuel type (exact, case-insensitive) ────────────
	if rule.FuelType != "" && !ciEqual(vehicle.FuelType, rule.FuelType) {
		return none
	}

	// ── Required: drive type — normalised so "FOUR WHEEL DRIVE"
	// and "4WD" are treated as the same value. ────────────────────
	if rule.DriveType != "" && normDriveType(vehicle.DriveType) != normDriveType(rule.DriveType) {
		return none
	}

	// ── Required: body type — token-based ────────────────────────
	if !matchesModelTokens(vehicle.BodyType, rule.BodyType) {
		return none
	}

	// ── Required: transmission type (exact, case-insensitive) ────
	if rule.TransmissionType != "" && !ciEqual(vehicle.TransmissionType, rule.TransmissionType) {
		return none
	}

	// ── Callouts ─────────────────────────────────────────────────
	var callouts []models.FitmentCallout
	if len(rule.Callouts) > 0 {
		_ = json.Unmarshal(rule.Callouts, &callouts)
	}

	var unverifiedNotes []string
	for _, co := range callouts {
		switch evaluateCallout(vehicle, co) {
		case calloutMismatch:
			// Field exists in vehicle record but value doesn't match → hard no
			return none
		case calloutMissing:
			// Field not in vehicle record → can't verify → fits with note
			note := strings.TrimSpace(co.Note)
			if note == "" {
				note = fmt.Sprintf("Verify: %s = %s", co.Field, co.Value)
			}
			unverifiedNotes = append(unverifiedNotes, note)
			// calloutMatch: all good, continue
		}
	}

	if len(unverifiedNotes) > 0 {
		return EvalResult{FitWithNote, unverifiedNotes, rule.Note}
	}
	return EvalResult{FitExact, nil, rule.Note}
}

// BestFitForPart evaluates all of a part's fitment rules against a vehicle
// and returns the best result (FitExact > FitWithNote > FitNone).

func BestFitForPart(vehicle models.Vehicle, rules []models.PartFitmentRule) EvalResult {
	best := EvalResult{FitNone, nil, ""}
	for _, rule := range rules {
		r := EvaluateRule(vehicle, rule)
		if r.Result > best.Result {
			best = r
		}
		if best.Result == FitExact {
			break // can't do better
		}
	}
	return best
}

// FitResultString converts a FitResult to its API string representation.

func FitResultString(r FitResult) string {
	switch r {
	case FitExact:
		return "exact"
	case FitWithNote:
		return "note"
	default:
		return "none"
	}
}
