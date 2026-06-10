package models

import "time"

// KnownVIN records a full 17-character VIN sighted for a build key. Same build key
// can have many units (check digit + serial differ); we keep every distinct VIN seen.
type KnownVIN struct {
	ID         uint      `gorm:"primaryKey"`
	BuildKey   string    `gorm:"column:build_key;not null;size:20;uniqueIndex:idx_known_vin_build_vin"`
	VIN        string    `gorm:"column:vin;not null;size:17;uniqueIndex:idx_known_vin_build_vin"`
	CheckDigit string    `gorm:"column:check_digit;not null;size:1"` // VIN position 9
	Serial     int64     `gorm:"column:serial;not null"`             // VIN positions 12–17
	SeenCount  int       `gorm:"column:seen_count;not null;default:1"`
	LastSeenAt time.Time `gorm:"column:last_seen_at;not null"`
	CreatedAt  time.Time
}
