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

// PendingPointsFor returns the un-absorbed pending sightings per field for a build key
// (the single verified observations still waiting for a second to form a range).
func PendingPointsFor(db *gorm.DB, buildKey string) (map[string][]int64, error) {
	var pts []models.FieldPoint
	if err := db.Where("build_key = ?", buildKey).
		Order("field_key ASC, serial ASC").Find(&pts).Error; err != nil {
		return nil, err
	}
	out := make(map[string][]int64)
	for _, p := range pts {
		out[p.FieldKey] = append(out[p.FieldKey], p.Serial)
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
