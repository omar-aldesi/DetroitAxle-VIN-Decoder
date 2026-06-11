package services

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"main/helpers"
	"main/models"
)

// RecordKnownVIN upserts a sighting of a full VIN for its build key. No-op for non-17-char
// input or invalid VINs. Increments seen_count when the same VIN is looked up again.
func RecordKnownVIN(db *gorm.DB, buildKey, vin string) error {
	vin = strings.TrimSpace(strings.ToUpper(vin))
	if len(vin) != 17 || !helpers.VinValidator(vin) {
		return nil
	}
	if buildKey == "" {
		buildKey = helpers.ExtractBuildKey(vin)
	}
	serial, ok := SerialFromVIN(vin)
	if !ok {
		return nil
	}
	checkDigit := string(vin[8])
	now := time.Now().UTC()

	var row models.KnownVIN
	err := db.Where("build_key = ? AND vin = ?", buildKey, vin).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&models.KnownVIN{
			BuildKey:   buildKey,
			VIN:        vin,
			CheckDigit: checkDigit,
			Serial:     serial,
			SeenCount:  1,
			LastSeenAt: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&row).Updates(map[string]any{
		"seen_count":   gorm.Expr("seen_count + 1"),
		"last_seen_at": now,
	}).Error
}

// KnownVINsForBuildKey returns full VINs seen for a build key, most recent first.
func KnownVINsForBuildKey(db *gorm.DB, buildKey string) ([]string, error) {
	var vins []string
	err := db.Model(&models.KnownVIN{}).
		Where("build_key = ?", buildKey).
		Order("last_seen_at DESC").
		Pluck("vin", &vins).Error
	return vins, err
}

// KnownVINsByBuildKeys returns VIN lists keyed by build_key (most recent first per key).
func KnownVINsByBuildKeys(db *gorm.DB, buildKeys []string) (map[string][]string, error) {
	out := make(map[string][]string, len(buildKeys))
	if len(buildKeys) == 0 {
		return out, nil
	}
	var rows []models.KnownVIN
	if err := db.Where("build_key IN ?", buildKeys).
		Order("last_seen_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.BuildKey] = append(out[r.BuildKey], r.VIN)
	}
	return out, nil
}
