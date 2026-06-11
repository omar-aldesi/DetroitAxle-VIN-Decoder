package services

import (
	"errors"
	"os"
	"testing"

	"main/models"
)

// TestLiveGMDecode exercises the real GM Parts Giant fetch + FormatAsJSON path.
// Guarded behind LIVE_TEST=1.
func TestLiveGMDecode(t *testing.T) {
	if os.Getenv("LIVE_TEST") != "1" {
		t.Skip("set LIVE_TEST=1 to run live API integration test")
	}

	// Standard GM VIN — should return rich build data.
	attrs, err := FetchGMAttributes("2GKALUEK5E6140075")
	if err != nil {
		t.Fatalf("FetchGMAttributes(GMC Terrain): %v", err)
	}
	data := attrs.FormatAsJSON()
	specs, _ := data["specifications"].([]map[string]string)
	t.Logf("Terrain: vehicle_info=%v, specs=%d, major=%v",
		data["vehicle_info"], len(specs), data["major_attributes"])
	if len(specs) == 0 {
		t.Error("expected non-empty specifications for a standard GM VIN")
	}
	// Confirm metadata rows were filtered out.
	for _, s := range specs {
		if s["desc"] == "TL1-" || s["desc"] == "2014-" {
			t.Errorf("metadata row leaked through: %v", s)
		}
	}

	// Stable-field enrichment from the real Terrain data.
	var v models.Vehicle
	attrs.ApplyStableFields(&v)
	t.Logf("Terrain stable fields → trim=%q series=%q cyl=%q disp=%q fuel=%q",
		v.Trim, v.Series, v.Cylinders, v.DisplacementL, v.FuelType)
	if v.Trim != "SLT" {
		t.Errorf("expected Trim=SLT from GM model string, got %q", v.Trim)
	}
	if v.Cylinders != "4" || v.DisplacementL != "2.4" {
		t.Errorf("expected engine cyl=4 disp=2.4 from GM, got cyl=%q disp=%q", v.Cylinders, v.DisplacementL)
	}

	// Upfitter-built Chevrolet — GM Parts Giant has no data → ErrGMNoData.
	_, err = FetchGMAttributes("54DBDJ1B5LS483704")
	if !errors.Is(err, ErrGMNoData) {
		t.Errorf("expected ErrGMNoData for upfitter VIN, got: %v", err)
	}
}

// TestLiveDecode exercises the real auto.dev + NHTSA path (no DB) for a set of
// VINs, including the originally-failing incomplete chassis cab. Guarded behind
// LIVE_TEST=1 so it never runs in normal CI.
//
//	LIVE_TEST=1 AUTO_DEV_TOKEN=... go test ./services/ -run TestLiveDecode -v
func TestLiveDecode(t *testing.T) {
	if os.Getenv("LIVE_TEST") != "1" {
		t.Skip("set LIVE_TEST=1 to run live API integration test")
	}
	token := os.Getenv("AUTO_DEV_TOKEN")
	if token == "" {
		t.Fatal("AUTO_DEV_TOKEN not set")
	}

	vins := []string{
		"54DBDJ1B5LS483704", // Chevy 3500 LCF chassis cab — auto.dev omits year
		"2GKALUEK5E6140075", // 2014 GMC Terrain — normal
	}

	for _, vin := range vins {
		auto, err := fetchAutoDev(vin, token)
		if err != nil {
			t.Errorf("[%s] fetchAutoDev: %v", vin, err)
			continue
		}
		nh, err := fetchNHTSA(vin)
		if err != nil {
			t.Logf("[%s] fetchNHTSA (non-fatal): %v", vin, err)
		}
		var v models.Vehicle
		if err := mapToVehicle(vin, auto, nh, &v); err != nil {
			t.Errorf("[%s] mapToVehicle: %v", vin, err)
			continue
		}
		if v.Year == 0 || v.Make == "" || v.Model == "" {
			t.Errorf("[%s] incomplete: year=%d make=%q model=%q", vin, v.Year, v.Make, v.Model)
		}
		t.Logf("[%s] OK → %d %s %s | trim=%q cyl=%q disp=%q fuel=%q drive=%q country=%q",
			vin, v.Year, v.Make, v.Model, v.Trim, v.Cylinders, v.DisplacementL, v.FuelType, v.DriveType, v.Country)
	}
}
