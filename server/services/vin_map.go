package services

import (
	"fmt"
	"log"
	"main/helpers"
	"main/models"
	"strconv"
	"strings"
)

// mapToVehicle merges auto.dev (primary) and NHTSA (supplementary) data onto v.
// Layer priority (highest → lowest):  auto.dev  →  NHTSA  →  VIN-derived  →  existing DB value
//
// GM Parts Giant data is NOT applied here — it is fetched live on demand and
// never persisted (RPO codes are per-VIN, not per build key).
func mapToVehicle(vin string, r *autoDevResponse, nhtsa *nhtsaResult, v *models.Vehicle) error {
	// auto.dev omits vehicle.year for some VIN types (e.g. incomplete/chassis
	// cabs), so we can't hard-require it. NHTSA's ModelYear is reliable, and the
	// VIN itself encodes the model year in position 10 as a last resort.
	year := r.Vehicle.Year
	if year == 0 && nhtsa != nil {
		if y, err := strconv.Atoi(strings.TrimSpace(nhtsa.ModelYear)); err == nil {
			year = y
		}
	}
	if year == 0 {
		year = modelYearFromVIN(vin)
	}
	if year == 0 {
		return fmt.Errorf("could not determine model year for VIN %s", vin)
	}

	// auto.dev's top-level make/model are present even when the nested vehicle
	// object is sparse; fall back to the nested object, then to NHTSA.
	make_ := coalesce(r.Make, r.Vehicle.Make, nhtsaStr(nhtsa, func(n *nhtsaResult) string { return n.Make }))
	model := coalesce(r.Model, r.Vehicle.Model, nhtsaStr(nhtsa, func(n *nhtsaResult) string { return n.Model }))
	if make_ == "" || model == "" {
		return fmt.Errorf("could not determine make/model for VIN %s", vin)
	}

	buildKey := helpers.ExtractBuildKey(vin)

	v.BuildKey = buildKey
	v.ExampleBuildNumber = strings.TrimSpace(r.VIN)
	v.Year = year
	v.Make = normalizeMake(make_) // preserve acronyms like GMC, BMW
	v.Model = model
	v.Trim = strings.TrimSpace(r.Trim)
	v.BodyType = strings.TrimSpace(r.Body)
	v.DriveType = normalizeDriveType(r.Drive)
	v.Country = strings.TrimSpace(r.Origin)
	v.TransmissionType = normalizeTransmission(r.Trans)

	v.Cylinders, v.DisplacementL, v.FuelType = parseEngineString(r.Engine)

	if nhtsa != nil {
		// Log NHTSA decode warnings without failing the whole request.
		if nhtsa.ErrorCode != "" && nhtsa.ErrorCode != "0" {
			log.Printf("[VIN decode] NHTSA warning vin=%s code=%s: %s",
				vin, nhtsa.ErrorCode, truncate(nhtsa.ErrorText, 120))
		}

		// Engine gaps
		if v.Cylinders == "" && nhtsa.EngineCylinders != "" {
			v.Cylinders = strings.TrimSpace(nhtsa.EngineCylinders)
		}
		if v.DisplacementL == "" && nhtsa.DisplacementL != "" {
			v.DisplacementL = strings.TrimSpace(nhtsa.DisplacementL)
		}
		if v.EngineConfiguration == "" && nhtsa.EngineConfiguration != "" {
			v.EngineConfiguration = strings.TrimSpace(nhtsa.EngineConfiguration)
		}

		// Body
		if v.Doors == "" && nhtsa.Doors != "" {
			v.Doors = strings.TrimSpace(nhtsa.Doors)
		}
		if v.BodyType == "" && nhtsa.BodyClass != "" {
			v.BodyType = strings.TrimSpace(nhtsa.BodyClass)
		}
		if v.BrakeSystemType == "" && nhtsa.BrakeSystemType != "" {
			v.BrakeSystemType = strings.TrimSpace(nhtsa.BrakeSystemType)
		}

		// Weight — NHTSA is usually reliable; GM will overwrite with exact value if available
		if nhtsa.GVWR != "" {
			v.GVWR = strings.TrimSpace(nhtsa.GVWR)
		}

		// Trim — NHTSA often has the full trim string when auto.dev is blank
		if v.Trim == "" && nhtsa.Trim != "" {
			v.Trim = strings.TrimSpace(nhtsa.Trim)
		}

		// Series — fill if auto.dev didn't provide one
		if v.Series == "" && nhtsa.Series != "" {
			v.Series = strings.TrimSpace(nhtsa.Series)
		}

		// Drive type — fill if auto.dev was blank
		if v.DriveType == "" && nhtsa.DriveType != "" {
			v.DriveType = normalizeDriveType(nhtsa.DriveType)
		}

		// Transmission speeds
		if v.Speeds == 0 && nhtsa.TransmissionSpeeds != "" {
			if n, err := strconv.Atoi(strings.TrimSpace(nhtsa.TransmissionSpeeds)); err == nil && n > 0 {
				v.Speeds = n
			}
		}

		// Transmission type — fill if auto.dev was blank
		if v.TransmissionType == "" && nhtsa.TransmissionStyle != "" {
			v.TransmissionType = normalizeTransmission(nhtsa.TransmissionStyle)
		}

		// Fuel type — NHTSA FuelTypePrimary+Secondary is more precise than auto.dev's
		// engine-string inference (catches flex-fuel, hybrid, hydrogen, etc.).
		// Only upgrade if auto.dev left us with the generic "Gasoline" default.
		if nhtsa.FuelTypePrimary != "" {
			if ft := normalizeNHTSAFuelType(nhtsa.FuelTypePrimary, nhtsa.FuelTypeSecondary); ft != "" {
				if v.FuelType == "" || v.FuelType == "Gasoline" {
					v.FuelType = ft
				}
			}
		}

		// ABS — NHTSA is the standard source for this
		if nhtsa.ABS != "" {
			v.ABS = strings.TrimSpace(nhtsa.ABS)
		}

		// Country — fall back to NHTSA plant country when auto.dev Origin is blank
		if v.Country == "" && nhtsa.PlantCountry != "" {
			v.Country = cleanNHTSACountry(nhtsa.PlantCountry)
		}
	}

	return nil
}
