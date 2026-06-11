package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"main/helpers"
	"main/models"
)

func TestRecordKnownVIN_Upsert(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.KnownVIN{}); err != nil {
		t.Fatal(err)
	}

	vin := "2GKALUEK5E6140075"
	if !helpers.VinValidator(vin) {
		t.Fatalf("test VIN %q must pass validator", vin)
	}
	bk := helpers.ExtractBuildKey(vin)

	if err := RecordKnownVIN(db, bk, vin); err != nil {
		t.Fatalf("first record: %v", err)
	}
	if err := RecordKnownVIN(db, bk, vin); err != nil {
		t.Fatalf("second record: %v", err)
	}

	var row models.KnownVIN
	if err := db.Where("vin = ?", vin).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.SeenCount != 2 {
		t.Fatalf("seen_count = %d, want 2", row.SeenCount)
	}
	if row.CheckDigit == "" {
		t.Fatal("check_digit should be set")
	}

	vins, err := KnownVINsForBuildKey(db, bk)
	if err != nil {
		t.Fatal(err)
	}
	if len(vins) != 1 || vins[0] != vin {
		t.Fatalf("known vins = %v, want [%s]", vins, vin)
	}
}
