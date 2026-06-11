package services

import (
	"main/models"

	"gorm.io/gorm"
)

// Note scope values describe whether a note applies to the VIN being viewed.
const (
	NoteScopeApplies = "applies" // confidently applies to this VIN
	NoteScopeWarn    = "warn"    // shown, but flagged (different range / unknown / unscoped)
)

// ForkBoundaries returns the distinct range start serials across ALL fork fields for a
// build key — i.e. every serial at which some build-number field is known to change.
func ForkBoundaries(db *gorm.DB, buildKey string) ([]int64, error) {
	var starts []int64
	err := db.Model(&models.FieldRange{}).
		Where("build_key = ?", buildKey).
		Distinct().
		Order("serial_start ASC").
		Pluck("serial_start", &starts).Error
	return starts, err
}

// NoteScope decides whether a note applies to the VIN being viewed. A note on the exact
// same serial always applies. Otherwise it applies only when both serials are known, we
// have range data, and NO fork boundary lies between the note's serial and the viewed
// serial. Anything else is shown with a warning.
func NoteScope(boundaries []int64, noteSerial, viewSerial *int64) string {
	if noteSerial == nil || viewSerial == nil {
		return NoteScopeWarn // unscoped, or viewing a build key with no specific serial
	}
	if *noteSerial == *viewSerial {
		return NoteScopeApplies // same VIN — applies regardless of range data
	}
	if len(boundaries) == 0 {
		return NoteScopeWarn // no range data — we don't know the range
	}
	lo, hi := *noteSerial, *viewSerial
	if lo > hi {
		lo, hi = hi, lo
	}
	for _, b := range boundaries {
		if b > lo && b <= hi {
			return NoteScopeWarn // a build-number change lies between the two VINs
		}
	}
	return NoteScopeApplies
}
