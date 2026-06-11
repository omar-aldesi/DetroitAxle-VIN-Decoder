package services

import (
	"main/models"

	"gorm.io/gorm"
)

// DefaultForkFields are the build-number-tier fields tracked out of the box (global scope).
//
// These keys MUST match the field names the edit/verify path receives (the request payload
// keys from the frontend's FORK_FIELDS, e.g. "brake_code"), since UpdateVehicle and
// VerifyEntry route by these exact names when feeding the engine. They are NOT columns on
// `vehicles` — build-number-tier values live only in field_range/field_point.
var DefaultForkFields = []string{
	"brake_code",
	"front_rotor_size",
	"rear_rotor_size",
	"front_spring_type", // front suspension
	"rear_spring_type",  // rear suspension
	"steering_type",     // steering
}

// renamedAwayForkFields are old default keys that were renamed to match Vehicle column
// names; remove the global entries so the edit path matches the registry.
var renamedAwayForkFields = []string{"front_suspension", "rear_suspension", "steering"}

// SeedDefaultForkFields inserts the default fork-field registry entries if missing.
// Idempotent — safe to call on every boot.
func SeedDefaultForkFields(db *gorm.DB) error {
	// Drop superseded default keys (safe: no range data was ever created under them, since
	// they never matched a column name in the edit path).
	if err := db.Where("key IN ? AND scope_make = '' AND scope_model = ''", renamedAwayForkFields).
		Delete(&models.ForkField{}).Error; err != nil {
		return err
	}

	for _, key := range DefaultForkFields {
		var count int64
		if err := db.Model(&models.ForkField{}).
			Where("key = ? AND scope_make = '' AND scope_model = ''", key).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&models.ForkField{
			Key: key, ScopeMake: "", ScopeModel: "",
			ValueType: "string", Enabled: true,
			Description: "default build-number fork field",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
