package helpers

import (
	"fmt"
	"main/models"
	"strings"
)

func normalizeKey(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer(" ", "", "-", "", "_", "").Replace(s)
	return s
}

var standardFieldMap = map[string]func(models.Vehicle) string{
	"year":             func(v models.Vehicle) string { return fmt.Sprintf("%d", v.Year) },
	"make":             func(v models.Vehicle) string { return v.Make },
	"model":            func(v models.Vehicle) string { return v.Model },
	"trim":             func(v models.Vehicle) string { return v.Trim },
	"series":           func(v models.Vehicle) string { return v.Series },
	"bodytype":         func(v models.Vehicle) string { return v.BodyType },
	"drivetype":        func(v models.Vehicle) string { return v.DriveType },
	"country":          func(v models.Vehicle) string { return v.Country },
	"cylinders":        func(v models.Vehicle) string { return v.Cylinders },
	"displacementl":    func(v models.Vehicle) string { return v.DisplacementL },
	"fueltype":         func(v models.Vehicle) string { return v.FuelType },
	"transmissiontype": func(v models.Vehicle) string { return v.TransmissionType },
	"speeds":           func(v models.Vehicle) string { return fmt.Sprintf("%d", v.Speeds) },
	"doors":            func(v models.Vehicle) string { return v.Doors },
	"abs":              func(v models.Vehicle) string { return v.ABS },
	"frontbraketype":   func(v models.Vehicle) string { return v.FrontBrakeType },
	"rearbraketype":    func(v models.Vehicle) string { return v.RearBrakeType },
	"frontrotorsize":   func(v models.Vehicle) string { return v.FrontRotorSize },
	"rearrotorsize":    func(v models.Vehicle) string { return v.RearRotorSize },
	"brakecode":        func(v models.Vehicle) string { return v.BrakeCode },
	"brakesystemtype":  func(v models.Vehicle) string { return v.BrakeSystemType },
	"frontspringtype":  func(v models.Vehicle) string { return v.FrontSpringType },
	"rearspringtype":   func(v models.Vehicle) string { return v.RearSpringType },
	"steeringtype":     func(v models.Vehicle) string { return v.SteeringType },
	"gvwr":             func(v models.Vehicle) string { return v.GVWR },
	"gvwrlbs":          func(v models.Vehicle) string { return v.GVWR },
}

// resolveField returns the vehicle's value for a given field path.
// Returns "" if the field doesn't exist or isn't set.

func resolveField(vehicle models.Vehicle, fieldPath string) string {
	norm := normalizeKey(fieldPath)

	// Custom field: "custom_fields.some_key" (also handles "customfields.some_key")
	const cfPrefix = "customfields."
	if strings.HasPrefix(norm, cfPrefix) {
		targetKey := normalizeKey(strings.TrimPrefix(norm, cfPrefix))
		for rawKey, rawVal := range vehicle.CustomFields {
			if normalizeKey(rawKey) == targetKey {
				return fmt.Sprintf("%v", rawVal)
			}
		}
		return "" // not found
	}

	// Standard field lookup
	if fn, ok := standardFieldMap[norm]; ok {
		return fn(vehicle)
	}

	// Fallback: search custom_fields without requiring the "custom_fields." prefix.
	// This lets callout authors write {field: "Lugs"} instead of
	// {field: "custom_fields.Lugs"} — the prefix is optional.
	for rawKey, rawVal := range vehicle.CustomFields {
		if normalizeKey(rawKey) == norm {
			return fmt.Sprintf("%v", rawVal)
		}
	}
	return ""
}

type calloutStatus int

const (
	calloutMatch    calloutStatus = 0 // field found and value matches
	calloutMismatch calloutStatus = 1 // field found but value doesn't match → hard no
	calloutMissing  calloutStatus = 2 // field not present in vehicle → fits with note
)
