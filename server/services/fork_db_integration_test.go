package services

import (
	"path/filepath"
	"testing"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"main/models"
)

// TestForkDB_SQLite exercises the real gorm-backed ForkStore + engine against an actual SQL
// database (SQLite, pure-Go driver) — the layer the in-memory ForkStore tests don't cover:
// AutoMigrate of the fork tables, the GORM column/index tags, and the store's queries.
//
// (Postgres-only paths — e.g. the backfill's DISTINCT ON — are not exercised here and are
// covered by the in-app behavior instead.)
func TestForkDB_SQLite(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "fork.db")
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Close the connection so Windows can remove the temp file during cleanup.
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	// Same migration + seed the app runs on boot.
	if err := db.AutoMigrate(&models.ForkField{}, &models.FieldRange{}, &models.FieldPoint{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := SeedDefaultForkFields(db); err != nil {
		t.Fatalf("SeedDefaultForkFields: %v", err)
	}

	const dbk = "ZZ_FORK_DB"
	const field = "brake_code"

	engine := NewForkEngine(NewGormForkStore(db))
	point := func(serial int64, val string) PointInput {
		return PointInput{BuildKey: dbk, Serial: serial, FieldKey: field, Value: val,
			Source: "dbtest", Verified: true}
	}
	mustResolve := func(serial int64) Resolved {
		m, err := engine.Resolve(dbk, serial, []string{field})
		if err != nil {
			t.Fatalf("Resolve(%d): %v", serial, err)
		}
		return m[field]
	}

	// Registry lookup against the real seeded table (the renamed-key cleanup ran too).
	if !IsForkFieldKey(db, "", "", field) {
		t.Fatal("brake_code should be a registered fork field after seeding")
	}
	if IsForkFieldKey(db, "", "", "front_suspension") {
		t.Fatal("renamed-away key front_suspension should not be registered")
	}
	if !IsForkFieldKey(db, "", "", "front_spring_type") {
		t.Fatal("front_spring_type should be a registered fork field")
	}

	// Two-point rule, persisted through GORM.
	if o, err := engine.RecordPoint(point(1, "JP9")); err != nil || o != OutcomePending {
		t.Fatalf("point #1 = (%v,%v), want pending", o, err)
	}
	if o, err := engine.RecordPoint(point(100, "JP9")); err != nil || o != OutcomeRangeCreated {
		t.Fatalf("point #2 = (%v,%v), want range_created", o, err)
	}
	if v := mustResolve(50).Value; v != "JP9" {
		t.Fatalf("resolve 50 = %q, want JP9 (interpolated)", v)
	}

	// Reinforce.
	if o, _ := engine.RecordPoint(point(50, "JP9")); o != OutcomeReinforced {
		t.Fatalf("reinforce = %v, want reinforced", o)
	}
	if obs := mustResolve(50).Observations; obs != 3 {
		t.Fatalf("observations = %d, want 3", obs)
	}

	// A carve (the 3-way split) exercised against the unique index on (build_key,field,serial_start).
	engine.RecordPoint(point(60, "JP6")) // first exception → pending
	if o, _ := engine.RecordPoint(point(70, "JP6")); o != OutcomeForked {
		t.Fatalf("second exception = %v, want forked", o)
	}
	for serial, want := range map[int64]string{40: "JP9", 60: "JP6", 70: "JP6", 90: "JP9"} {
		if v := mustResolve(serial).Value; v != want {
			t.Errorf("after carve, resolve %d = %q, want %q", serial, v, want)
		}
	}

	// Manual range wins + trims, persisted.
	if o, err := engine.RecordRange(RangeInput{
		BuildKey: dbk, SerialStart: 50, SerialEnd: ptr(100), FieldKey: field,
		Value: "JP1", Source: "dnr", Verified: true,
	}); err != nil || o != OutcomeManualRange {
		t.Fatalf("manual range = (%v,%v), want manual_range", o, err)
	}
	if r := mustResolve(75); r.Value != "JP1" || r.Origin != "manual" {
		t.Fatalf("resolve 75 = %+v, want JP1/manual", r)
	}
	if v := mustResolve(25).Value; v != "JP9" {
		t.Fatalf("resolve 25 = %q, want JP9 (trimmed left remnant)", v)
	}

	// Build-key view + pending visibility round-trip through the DB.
	views, err := engine.ResolveByBuildKey(dbk, []string{field})
	if err != nil {
		t.Fatalf("ResolveByBuildKey: %v", err)
	}
	if len(views[field]) == 0 {
		t.Fatal("expected ranges in the build-key view")
	}
	pending, err := PendingPointsFor(db, dbk)
	if err != nil {
		t.Fatalf("PendingPointsFor: %v", err)
	}
	t.Logf("DB(SQLite) OK — %d ranges, pending=%v", len(views[field]), pending)
}
