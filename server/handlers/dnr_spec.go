package handlers

import (
	"math"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"main/models"
)

// Build-key-tier spec fields for DNR research. Fork-tier fields are excluded.
var dnrSpecFields = []struct {
	Key      string
	Category string
}{
	{"trim", "identity"},
	{"series", "identity"},
	{"body_type", "identity"},
	{"doors", "identity"},
	{"drive_type", "identity"},
	{"country", "identity"},
	{"cylinders", "engine"},
	{"displacement_l", "engine"},
	{"fuel_type", "engine"},
	{"engine_configuration", "engine"},
	{"transmission_type", "transmission"},
	{"speeds", "transmission"},
	{"abs", "brakes"},
	{"brake_system_type", "brakes"},
	{"front_brake_type", "brakes"},
	{"rear_brake_type", "brakes"},
	{"gvwr_lbs", "brakes"},
}

// fieldValue extracts a spec field value from a Vehicle as a string.
func fieldValue(v models.Vehicle, key string) string {
	switch key {
	case "trim":
		return v.Trim
	case "series":
		return v.Series
	case "body_type":
		return v.BodyType
	case "doors":
		return v.Doors
	case "drive_type":
		return v.DriveType
	case "country":
		return v.Country
	case "cylinders":
		return v.Cylinders
	case "displacement_l":
		return v.DisplacementL
	case "fuel_type":
		return v.FuelType
	case "engine_configuration":
		return v.EngineConfiguration
	case "transmission_type":
		return v.TransmissionType
	case "speeds":
		if v.Speeds == 0 {
			return ""
		}
		return strconv.Itoa(v.Speeds)
	case "abs":
		return v.ABS
	case "brake_system_type":
		return v.BrakeSystemType
	case "front_brake_type":
		return v.FrontBrakeType
	case "rear_brake_type":
		return v.RearBrakeType
	case "gvwr_lbs":
		return v.GVWR
	}
	return ""
}

func missingFields(v models.Vehicle, categoryFilter string) []string {
	out := []string{}
	for _, f := range dnrSpecFields {
		if categoryFilter != "" && categoryFilter != "all" && f.Category != categoryFilter {
			continue
		}
		val := strings.TrimSpace(fieldValue(v, f.Key))
		if val == "" || val == "0" {
			out = append(out, f.Key)
		}
	}
	return out
}

func completeness(v models.Vehicle) float64 {
	total := len(dnrSpecFields)
	filled := 0
	for _, f := range dnrSpecFields {
		val := strings.TrimSpace(fieldValue(v, f.Key))
		if val != "" && val != "0" {
			filled++
		}
	}
	return math.Round(float64(filled)/float64(total)*1000) / 10 // one decimal
}

type DNRHandler struct {
	DB *gorm.DB
}
