package services

import (
	"testing"

	"main/models"
)

func gmTerrainAttrs() *GMVINAttributes {
	return &GMVINAttributes{
		VinInfos: []gmVinInfo{{
			MajorAttribute: []gmNameDesc{
				{Name: "Engine", Desc: "4 Cyl 2.4L SIDI, DOHC, E85 MAX, ALUM"},
				{Name: "Model String", Desc: "Silverado 1500 'LTZ' Crew Cab"},
				{Name: "Transmission", Desc: "6-Speed Automatic Transmission, HMD, 6L-80"},
			},
			Specification: []gmNameDesc{
				{Name: "BRK", Desc: "JD9-BRAKE VAC POWER, 17\" DISC/DISC"},
				{Name: "GVW", Desc: "C5Z-GVW RATING 7,200 LBS"},
			},
		}},
	}
}

func TestEnrichExistingWithGM_NoOpGuards(t *testing.T) {
	// Each of these must return nil WITHOUT touching the (nil) DB or network,
	// so a nil *gorm.DB proves we returned before using it.
	cases := []struct {
		name string
		vin  string
		v    models.Vehicle
	}{
		{"already checked", "1GCEK14K0RE100000", models.Vehicle{GMChecked: true}},
		{"not a GM VIN", "JTDBR32E430100000", models.Vehicle{}},
		{"build key (not 17)", "1GCEK14K0R", models.Vehicle{}},
	}
	for _, c := range cases {
		if err := EnrichExistingWithGM(nil, c.vin, &c.v); err != nil {
			t.Errorf("%s: expected nil, got %v", c.name, err)
		}
	}
}

func TestParseGMTrim(t *testing.T) {
	cases := map[string]string{
		"Terrain 'SLT' SUV":             "SLT",
		"Silverado 1500 'LTZ' Crew Cab": "LTZ",
		"Silverado 1500 Crew Cab":       "", // no quotes → no trim
		"":                              "",
	}
	for in, want := range cases {
		if got := parseGMTrim(in); got != want {
			t.Errorf("parseGMTrim(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestApplyStableFields_FillsStableOnly(t *testing.T) {
	var v models.Vehicle
	gmTerrainAttrs().ApplyStableFields(&v)

	if v.Series != "1500" {
		t.Errorf("Series = %q, want 1500", v.Series)
	}
	if v.Trim != "LTZ" {
		t.Errorf("Trim = %q, want LTZ", v.Trim)
	}
	if v.Cylinders != "4" {
		t.Errorf("Cylinders = %q, want 4", v.Cylinders)
	}
	if v.DisplacementL != "2.4" {
		t.Errorf("DisplacementL = %q, want 2.4", v.DisplacementL)
	}
	if v.FuelType != "Flex Fuel" {
		t.Errorf("FuelType = %q, want Flex Fuel (E85)", v.FuelType)
	}

	// Transmission is NOT VDS-encoded → must never be persisted from GM,
	// even though the majorAttribute contains "6-Speed …".
	if v.Speeds != 0 {
		t.Errorf("Speeds = %d, want 0 (transmission is per-unit, not build-key-stable)", v.Speeds)
	}

	// Per-VIN fields must NEVER be set from GM, even though BRK/GVW are present.
	if v.GVWR != "" {
		t.Errorf("GVWR leaked from GM: %q", v.GVWR)
	}
}

func TestApplyStableFields_NeverOverwrites(t *testing.T) {
	v := models.Vehicle{
		Series:        "EXISTING",
		Trim:          "KEEP_ME",
		Cylinders:     "8",
		DisplacementL: "5.3",
		FuelType:      "Diesel", // not the generic default → must be preserved
	}
	gmTerrainAttrs().ApplyStableFields(&v)

	if v.Series != "EXISTING" || v.Trim != "KEEP_ME" || v.Cylinders != "8" ||
		v.DisplacementL != "5.3" || v.FuelType != "Diesel" {
		t.Errorf("ApplyStableFields overwrote an existing value: %+v", v)
	}
}

func TestApplyStableFields_NilSafe(t *testing.T) {
	var a *GMVINAttributes
	var v models.Vehicle
	a.ApplyStableFields(&v) // must not panic
	(&GMVINAttributes{}).ApplyStableFields(&v)
}

func TestFormatAsJSON_FiltersMetadataKeepsRPO(t *testing.T) {
	attrs := &GMVINAttributes{
		VinInfos: []gmVinInfo{
			{
				VehicleInfo: "2014 Light Truck Terrain",
				RedirectURL: "https://example.com/x",
				MajorAttribute: []gmNameDesc{
					{Name: "Engine", Desc: "4 Cyl 2.4L SIDI, DOHC"},
				},
				Specification: []gmNameDesc{
					{Name: "CATALOG", Desc: "TL1-"},                         // metadata → drop
					{Name: "MD", Desc: "L-"},                                // metadata → drop
					{Name: "YEAR_FROM", Desc: "2014-"},                      // metadata → drop
					{Name: "AFT", Desc: "AE8-ADJUSTER FRT ST POWER, 8 WAY"}, // real → keep
					{Name: "ABS", Desc: "STANDARD"},                         // bare word → keep
					{Name: "EMPTY", Desc: ""},                               // empty → drop
				},
			},
		},
	}

	out := attrs.FormatAsJSON()
	if out == nil {
		t.Fatal("FormatAsJSON returned nil")
	}

	specs, ok := out["specifications"].([]map[string]string)
	if !ok {
		t.Fatalf("specifications wrong type: %T", out["specifications"])
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs kept (AE8 + STANDARD), got %d: %+v", len(specs), specs)
	}

	gotDescs := map[string]bool{}
	for _, s := range specs {
		gotDescs[s["desc"]] = true
	}
	if !gotDescs["AE8-ADJUSTER FRT ST POWER, 8 WAY"] {
		t.Error("real RPO row was dropped")
	}
	if !gotDescs["STANDARD"] {
		t.Error("bare-word description 'STANDARD' was incorrectly dropped")
	}

	if out["vehicle_info"] != "2014 Light Truck Terrain" {
		t.Errorf("vehicle_info = %v", out["vehicle_info"])
	}
	if out["redirect_url"] != "https://example.com/x" {
		t.Errorf("redirect_url = %v", out["redirect_url"])
	}
}

func TestFormatAsJSON_NilSafe(t *testing.T) {
	var a *GMVINAttributes
	if a.FormatAsJSON() != nil {
		t.Error("nil receiver should return nil")
	}
	empty := &GMVINAttributes{}
	if empty.FormatAsJSON() != nil {
		t.Error("empty VinInfos should return nil")
	}
}

func TestIsGMBrandVIN(t *testing.T) {
	cases := map[string]bool{
		"1GCEK14K0RE100000": true,  // 1G
		"2GKALUEK5E6140075": true,  // 2G
		"3GNEK13T0YG100000": true,  // 3G
		"KL4CJBSB6JB100000": true,  // KL4
		"W0LABC1234567890X": true,  // W0L
		"54DBDJ1B5LS483704": false, // upfitter WMI (still a Chevrolet, but not a GM WMI)
		"JTDBR32E430100000": false, // Toyota
		"":                  false,
	}
	for vin, want := range cases {
		if got := IsGMBrandVIN(vin); got != want {
			t.Errorf("IsGMBrandVIN(%q) = %v, want %v", vin, got, want)
		}
	}
}
