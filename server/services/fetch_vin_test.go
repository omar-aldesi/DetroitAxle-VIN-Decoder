package services

import (
	"encoding/json"
	"testing"

	"main/models"
)

func TestModelYearFromVIN(t *testing.T) {
	cases := map[string]int{
		"54DBDJ1B5LS483704": 2020, // pos10 = L → 1990|2020; window picks 2020 (the originally failing VIN)
		"2GKALUEK5E6140075": 2014, // pos10 = E → 1984|2014; window picks 2014
		"1GCEK14K0RE100000": 2024, // pos10 = R → 1994|2024; window picks 2024 (most recent plausible)
		"short":             0,    // too short
	}
	for vin, want := range cases {
		if got := modelYearFromVIN(vin); got != want {
			t.Errorf("modelYearFromVIN(%q) = %d, want %d", vin, got, want)
		}
	}
}

func TestParseEngineString(t *testing.T) {
	type out struct{ cyl, disp, fuel string }
	cases := map[string]out{
		"5.3L V8 OHV 16V FFV":    {"8", "5.3", "Flex Fuel"},
		"6.0, 8 Cylinder Engine": {"8", "6.0", "Gasoline"},
		"3.6L V6 24V GDI DOHC":   {"6", "3.6", "Gasoline"},
		"2.0L I4 DIESEL":         {"4", "2.0", "Diesel"},
		"":                       {"", "", ""},
	}
	for in, want := range cases {
		cyl, disp, fuel := parseEngineString(in)
		if cyl != want.cyl || disp != want.disp || fuel != want.fuel {
			t.Errorf("parseEngineString(%q) = (%q,%q,%q), want (%q,%q,%q)",
				in, cyl, disp, fuel, want.cyl, want.disp, want.fuel)
		}
	}
}

func TestNormalizeMake(t *testing.T) {
	cases := map[string]string{
		"GMC":        "GMC",
		"BMW":        "BMW",
		"Chevrolet":  "Chevrolet",
		"CHEVROLET":  "Chevrolet",
		"  Buick  ":  "Buick",
	}
	for in, want := range cases {
		if got := normalizeMake(in); got != want {
			t.Errorf("normalizeMake(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMapToVehicle_IncompleteVIN reproduces the originally failing case:
// a Chevy 3500 LCF chassis cab where auto.dev omits vehicle.year entirely.
func TestMapToVehicle_IncompleteVIN(t *testing.T) {
	// Real auto.dev payload (trimmed) — note: NO year anywhere.
	autoJSON := `{
		"vin": "54DBDJ1B5LS483704",
		"vinValid": true,
		"origin": "United States",
		"make": "Chevrolet",
		"model": "3500 LCF Gas",
		"trim": "2WD Crew Cab 176\"",
		"body": "Incomplete - Chassis Cab (Single Cab)",
		"engine": "6.0, 8 Cylinder Engine",
		"drive": "Rear Wheel Drive",
		"transmission": "Automatic",
		"vehicle": { "vin": "54DBDJ1B5LS483704", "make": "Chevrolet", "model": "3500 LCF Gas" }
	}`
	// Real NHTSA Results[0] (trimmed).
	nhtsaJSON := `{
		"ModelYear": "2020",
		"EngineCylinders": "8",
		"DisplacementL": "6.0",
		"EngineConfiguration": "V-Shaped",
		"FuelTypePrimary": "Gasoline",
		"DriveType": "4x2",
		"BodyClass": "Incomplete - Chassis Cab (Single Cab)",
		"BrakeSystemType": "Hydraulic",
		"GVWR": "Class 3: 10,001 - 14,000 lb (4,536 - 6,350 kg)",
		"PlantCountry": "UNITED STATES (USA)",
		"ErrorCode": "4,14"
	}`

	var auto autoDevResponse
	if err := json.Unmarshal([]byte(autoJSON), &auto); err != nil {
		t.Fatalf("auto unmarshal: %v", err)
	}
	var nh nhtsaResult
	if err := json.Unmarshal([]byte(nhtsaJSON), &nh); err != nil {
		t.Fatalf("nhtsa unmarshal: %v", err)
	}

	var v models.Vehicle
	if err := mapToVehicle("54DBDJ1B5LS483704", &auto, &nh, &v); err != nil {
		t.Fatalf("mapToVehicle returned error: %v", err)
	}

	if v.Year != 2020 {
		t.Errorf("Year = %d, want 2020", v.Year)
	}
	if v.Make != "Chevrolet" {
		t.Errorf("Make = %q, want Chevrolet", v.Make)
	}
	if v.Model != "3500 LCF Gas" {
		t.Errorf("Model = %q, want '3500 LCF Gas'", v.Model)
	}
	if v.Cylinders != "8" {
		t.Errorf("Cylinders = %q, want 8", v.Cylinders)
	}
	if v.DisplacementL != "6.0" {
		t.Errorf("DisplacementL = %q, want 6.0", v.DisplacementL)
	}
	if v.DriveType != "RWD" {
		t.Errorf("DriveType = %q, want RWD (auto.dev wins over NHTSA 4x2)", v.DriveType)
	}
	if v.Country != "United States" {
		t.Errorf("Country = %q, want 'United States'", v.Country)
	}
	if v.BuildKey == "" {
		t.Error("BuildKey is empty")
	}
}

// TestMapToVehicle_NormalVIN confirms auto.dev's own year wins when present.
func TestMapToVehicle_NormalVIN(t *testing.T) {
	autoJSON := `{
		"vin": "2GKALUEK5E6140075",
		"vinValid": true,
		"origin": "Canada",
		"make": "GMC",
		"model": "Terrain",
		"trim": "Denali",
		"body": "SUV",
		"engine": "3.6L V6 24V GDI DOHC",
		"drive": "All Wheel Drive",
		"transmission": "6-Speed Automatic",
		"vehicle": { "vin": "2GKALUEK5E6140075", "year": 2014, "make": "GMC", "model": "Terrain" }
	}`
	var auto autoDevResponse
	if err := json.Unmarshal([]byte(autoJSON), &auto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var v models.Vehicle
	if err := mapToVehicle("2GKALUEK5E6140075", &auto, nil, &v); err != nil {
		t.Fatalf("mapToVehicle error: %v", err)
	}
	if v.Year != 2014 {
		t.Errorf("Year = %d, want 2014", v.Year)
	}
	if v.Make != "GMC" {
		t.Errorf("Make = %q, want GMC (acronym preserved)", v.Make)
	}
	if v.DriveType != "AWD" {
		t.Errorf("DriveType = %q, want AWD", v.DriveType)
	}
}
