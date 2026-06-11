package services

import (
	"strconv"
	"strings"

	"main/models"

	"gorm.io/gorm"
)

// SerialFromVIN extracts the build number (positions 12–17) as an integer from a full
// 17-character VIN whose serial is numeric. Returns (0,false) for build-key lookups
// (len != 17) or non-numeric serials (treat that build key as un-forkable).
func SerialFromVIN(vin string) (int64, bool) {
	vin = strings.TrimSpace(strings.ToUpper(vin))
	if len(vin) != 17 {
		return 0, false
	}
	n, err := strconv.ParseInt(vin[11:17], 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// IsForkFieldKey reports whether a field is a registered fork field for this make/model.
// Lets callers (e.g. the edit handler) decide a field's tier without the full engine.
func IsForkFieldKey(db *gorm.DB, make_, model, fieldKey string) bool {
	ok, err := NewGormForkStore(db).IsForkField(make_, model, fieldKey)
	return err == nil && ok
}

// PendingPoint is one verified single sighting still waiting for a second matching VIN to
// confirm a range. The value is included so callers can show what was recorded for the VIN.
type PendingPoint struct {
	Serial int64  `json:"serial"`
	Value  string `json:"value"`
}

// PendingPointsFor returns the un-absorbed pending sightings per field for a build key
// (the single verified observations still waiting for a second to form a range).
func PendingPointsFor(db *gorm.DB, buildKey string) (map[string][]PendingPoint, error) {
	var pts []models.FieldPoint
	if err := db.Where("build_key = ?", buildKey).
		Order("field_key ASC, serial ASC").Find(&pts).Error; err != nil {
		return nil, err
	}
	out := make(map[string][]PendingPoint)
	for _, p := range pts {
		out[p.FieldKey] = append(out[p.FieldKey], PendingPoint{Serial: p.Serial, Value: p.Value})
	}
	return out, nil
}

// ProposedEditsFor returns the latest UNVERIFIED build-number-tier edit per (field, serial)
// for a vehicle — values that have been entered against a specific VIN but have not yet been
// verified, so they aren't in the range engine. They give the vehicle page something to show
// per build number ("pending review") between entry and verification. Once an edit is
// verified it leaves this set and becomes a pending sighting (PendingPointsFor) or a range.
func ProposedEditsFor(db *gorm.DB, vehicleID uint) (map[string][]PendingPoint, error) {
	var rows []models.VehicleFieldHistory
	if err := db.
		Where("vehicle_id = ? AND tier = ? AND is_verified = ? AND origin_serial IS NOT NULL",
			vehicleID, "build_number", false).
		Order("field_name ASC, origin_serial ASC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string][]PendingPoint)
	seen := make(map[string]bool) // field|serial -> kept the latest (highest id) already
	for _, r := range rows {
		if r.OriginSerial == nil {
			continue
		}
		key := r.FieldName + "|" + strconv.FormatInt(*r.OriginSerial, 10)
		if seen[key] {
			continue
		}
		seen[key] = true
		out[r.FieldName] = append(out[r.FieldName], PendingPoint{Serial: *r.OriginSerial, Value: r.NewValue})
	}
	return out, nil
}

// ForkFieldKeysFor returns the enabled fork-field keys that apply to a make/model
// (global + make-scoped + make/model-scoped).
func ForkFieldKeysFor(db *gorm.DB, make_, model string) ([]string, error) {
	var keys []string
	err := db.Model(&models.ForkField{}).
		Where("enabled = ?", true).
		Where("(scope_make = '' OR scope_make = ?)", make_).
		Where("(scope_model = '' OR scope_model = ?)", model).
		Distinct().
		Pluck("key", &keys).Error
	return keys, err
}

// FeedForkField records a VERIFIED fork-field value into the range engine. A VIN sighting
// (serial set) goes through RecordPoint (two-point rule); a build-key-wide value (serial
// nil) is written as a gap-filling base range — it never erases existing per-serial range
// data. No-op + (false,nil) if the field is not a registered fork field. Always call with
// verified/authoritative data only.
func FeedForkField(db *gorm.DB, buildKey, make_, model, fieldKey, value string,
	serial *int64, source string, actorID *uint) (bool, error) {

	store := NewGormForkStore(db)
	isFork, err := store.IsForkField(make_, model, fieldKey)
	if err != nil || !isFork {
		return false, err
	}
	engine := NewForkEngine(store)
	if serial != nil {
		_, err = engine.RecordPoint(PointInput{
			BuildKey: buildKey, Serial: *serial, FieldKey: fieldKey, Value: value,
			Source: source, Verified: true, ActorID: actorID, Make: make_, Model: model,
		})
	} else {
		_, err = engine.RecordBaseRange(RangeInput{
			BuildKey: buildKey, SerialStart: 0, SerialEnd: nil, FieldKey: fieldKey,
			Value: value, Source: source, Verified: true, ActorID: actorID,
			Make: make_, Model: model,
		})
	}
	return true, err
}
