package helpers

import "main/models"

func StructToMap(v models.Vehicle) map[string]interface{} {
	return map[string]interface{}{
		"year":              v.Year,
		"make":              v.Make,
		"model":             v.Model,
		"trim":              v.Trim,
		"series":            v.Series,
		"body_type":         v.BodyType,
		"drive_type":        v.DriveType,
		"country":           v.Country,
		"cylinders":         v.Cylinders,
		"displacement_l":    v.DisplacementL,
		"fuel_type":         v.FuelType,
		"transmission_type": v.TransmissionType,
		"speeds":            v.Speeds,
		"gvwr_lbs":          v.GVWR,
		"abs":               v.ABS,
		"front_brake_type":  v.FrontBrakeType,
		"rear_brake_type":   v.RearBrakeType,
		"custom_fields":     v.CustomFields,
	}
}
