package models

import (
	"time"

	"gorm.io/datatypes" // Required for JSON handling
	"gorm.io/gorm"
)

type NoteType string

const (
	NoteTypeFreeText     NoteType = "free_text"
	NoteTypePart         NoteType = "part_number"
	NoteTypeListingError NoteType = "listing_error"
)

type PartCategory struct {
	gorm.Model
	Name  string      `gorm:"uniqueIndex;not null;size:100"`
	Notes []AgentNote `gorm:"foreignKey:PartCategoryID"`
}

type FieldPermission struct {
	gorm.Model
	Role      string `gorm:"not null;size:20;uniqueIndex:idx_role_field"`
	FieldName string `gorm:"not null;size:100;uniqueIndex:idx_role_field"`
	CanEdit   bool   `gorm:"not null;default:false"`
}

type VehicleFieldHistory struct {
	gorm.Model
	VehicleID uint   `gorm:"not null;index"`
	UserID    uint   `gorm:"not null;index"`
	Username  string `gorm:"not null;size:50"`
	FieldName string `gorm:"not null;size:100"`
	OldValue  string `gorm:"size:500"`
	NewValue  string `gorm:"size:500"`
	IsTrusted bool   `gorm:"not null;default:false"`

	IsVerified bool `gorm:"not null;default:false"`

	VerifierID *uint  `gorm:"index"`
	Source     string `gorm:"column:source;size:500"` // where this data came from (DNR use)

	// Build-number fork system: which serial this edit was made against (nil = the
	// edit was build-key-wide), and which tier the field belongs to.
	OriginSerial *int64 `gorm:"column:origin_serial"`
	Tier         string `gorm:"column:tier;size:20"` // "build_key" | "build_number" | ""

	Vehicle *Vehicle `gorm:"foreignKey:VehicleID"`
}

type AgentNote struct {
	gorm.Model

	// Vehicle this note belongs to
	VehicleID uint `gorm:"not null;index"`

	// Author
	UserID   uint   `gorm:"not null;index"`
	Username string `gorm:"not null;size:50"`

	// Content
	NoteType   NoteType `gorm:"not null;size:20"`
	FreeText   *string  `gorm:"size:1000"`
	PartNumber *string  `gorm:"size:100;index"`

	// Only for part-number notes
	PartCategoryID *uint
	PartCategory   *PartCategory

	User User `gorm:"foreignKey:UserID"`

	IsResolved  bool    `gorm:"not null;default:false"`
	ResolveNote *string `gorm:"size:1000"`

	// Build-number the note was entered against (nil = entered build-key-wide). Used to
	// compute whether the note applies to the VIN being viewed (see services.NoteScope).
	OriginSerial *int64 `gorm:"column:origin_serial"`
}
type Vehicle struct {
	ID                 uint   `gorm:"primaryKey;autoIncrement"`
	BuildKey           string `gorm:"column:build_key;unique;notNull"`
	ExampleBuildNumber string `gorm:"column:example_build_number"`

	// --- Basic Info ---
	Year      int    `gorm:"column:year;notNull"`
	Make      string `gorm:"column:make;notNull"`
	Model     string `gorm:"column:model;notNull"`
	Trim      string `gorm:"column:trim"`
	Series    string `gorm:"column:series"`
	BodyType  string `gorm:"column:body_type"`
	DriveType string `gorm:"column:drive_type"`
	Country   string `gorm:"column:country"`

	Cylinders     string `gorm:"column:cylinders"`
	DisplacementL string `gorm:"column:displacement_l"`
	FuelType      string `gorm:"column:fuel_type"`

	TransmissionType string `gorm:"column:transmission_type"`
	Speeds           int    `gorm:"column:speeds"`

	GVWR string `gorm:"column:gvwr_lbs"`

	ABS string `gorm:"column:abs"`

	FrontBrakeType string `gorm:"column:front_brake_type"`
	RearBrakeType  string `gorm:"column:rear_brake_type"`

	// Build-NUMBER-tier fields (brake code, rotor sizes, suspension, steering) are not
	// stored here — they're managed entirely by the fork/range engine (field_range /
	// field_point) and resolved per-VIN via services.ForkEngine.Resolve.

	CustomFields datatypes.JSONMap `gorm:"column:custom_fields;type:jsonb"`

	BrakeSystemType     string `gorm:"column:brake_system_type"`
	Doors               string `gorm:"column:doors"`
	EngineConfiguration string `gorm:"column:engine_configuration"`

	// GMChecked is true once we've attempted GM Parts Giant enrichment for this
	// build key (whether GM had data or not). Lets us backfill GM build-key-stable
	// fields exactly once for GM cars, and filter GM cars still pending lookup.
	//
	// NOTE: GM's per-VIN RPO data (brake/option codes) is NEVER stored — it is
	// VIN-specific and fetched live via GET /api/gm/decode/:vin. Only VDS-encoded,
	// build-key-stable GM fields (trim, series, engine) are persisted.
	GMChecked bool `gorm:"column:gm_checked;default:false;not null"`

	// --- Notes ---
	Notes []AgentNote `gorm:"foreignKey:VehicleID"`

	History []VehicleFieldHistory `gorm:"foreignKey:VehicleID"`

	CreatedAt time.Time `gorm:"column:created_at;notNull;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;notNull;default:CURRENT_TIMESTAMP"`
}
