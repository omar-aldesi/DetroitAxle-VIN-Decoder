package models

import "time"

// fork.go — build-number-tier data model.
//
// Build-key-tier facts (year/make/model/engine/…) live on the Vehicle row and are shared
// by every VIN in a build key. Build-number-tier facts (brake code, rotor sizes,
// suspension, steering, …) can change along the serial (VIN positions 12–17), so they live
// in per-field RANGES of serials instead of as single Vehicle columns.
//
// See docs/fork-range-system.md.

// ForkField is the registry of which field keys are build-number-tier (forkable),
// optionally scoped by make/model. Empty ScopeMake/ScopeModel mean "applies to all".
type ForkField struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Key         string    `gorm:"column:key;not null;uniqueIndex:idx_forkfield_scope,priority:1"`
	ScopeMake   string    `gorm:"column:scope_make;not null;default:'';uniqueIndex:idx_forkfield_scope,priority:2"`
	ScopeModel  string    `gorm:"column:scope_model;not null;default:'';uniqueIndex:idx_forkfield_scope,priority:3"`
	ValueType   string    `gorm:"column:value_type;not null;default:string"` // "string" | "number" | "enum"
	Enabled     bool      `gorm:"column:enabled;not null;default:true"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// FieldRange is a confirmed [SerialStart, SerialEnd] = Value span for one fork field within
// one build key. Born from >=2 agreeing verified sightings (Origin "vin") or an explicit
// human entry (Origin "manual"). SerialStart 0 is the build-key-wide base. A nil SerialEnd
// means open-ended (the last range).
type FieldRange struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	BuildKey      string `gorm:"column:build_key;not null;uniqueIndex:idx_fieldrange_key,priority:1;index:idx_fieldrange_lookup,priority:1"`
	FieldKey      string `gorm:"column:field_key;not null;uniqueIndex:idx_fieldrange_key,priority:2;index:idx_fieldrange_lookup,priority:2"`
	SerialStart   int64  `gorm:"column:serial_start;not null;uniqueIndex:idx_fieldrange_key,priority:3;index:idx_fieldrange_lookup,priority:3"`
	SerialEnd     *int64 `gorm:"column:serial_end"` // nil = open-ended
	Value         string `gorm:"column:value;not null"`
	ValueNorm     string `gorm:"column:value_norm;not null"`
	Origin        string `gorm:"column:origin;not null"` // "vin" | "manual"
	Source        string `gorm:"column:source;not null"`
	Observations  int    `gorm:"column:observations;not null;default:1"`
	SerialMinSeen int64  `gorm:"column:serial_min_seen;not null"`
	SerialMaxSeen int64  `gorm:"column:serial_max_seen;not null"`
	BoundaryExact bool   `gorm:"column:boundary_exact;not null;default:false"`
	LastActor     *uint  `gorm:"column:last_actor"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

// FieldPoint is a verified VIN sighting not yet absorbed into a range.
type FieldPoint struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	BuildKey  string    `gorm:"column:build_key;not null;uniqueIndex:idx_fieldpoint_key,priority:1;index:idx_fieldpoint_lookup,priority:1"`
	FieldKey  string    `gorm:"column:field_key;not null;uniqueIndex:idx_fieldpoint_key,priority:2;index:idx_fieldpoint_lookup,priority:2"`
	Serial    int64     `gorm:"column:serial;not null;uniqueIndex:idx_fieldpoint_key,priority:3"`
	Value     string    `gorm:"column:value;not null"`
	ValueNorm string    `gorm:"column:value_norm;not null;index:idx_fieldpoint_lookup,priority:3"`
	Source    string    `gorm:"column:source;not null"`
	Actor     *uint     `gorm:"column:actor"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
