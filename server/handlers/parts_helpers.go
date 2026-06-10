package handlers

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"main/models"
)

func canEditParts(user *models.User) bool {
	return user.Role == "admin" || user.Role == "listing" || user.Role == "dnr"
}

func bindRule(c *gin.Context) (models.PartFitmentRule, error) {
	var req struct {
		YearMin          *int                    `json:"year_min"`
		YearMax          *int                    `json:"year_max"`
		Make             string                  `json:"make"`
		Model            string                  `json:"model"`
		Trim             string                  `json:"trim"`
		Cylinders        string                  `json:"cylinders"`
		DisplacementL    string                  `json:"displacement_l"`
		FuelType         string                  `json:"fuel_type"`
		DriveType        string                  `json:"drive_type"`
		BodyType         string                  `json:"body_type"`
		TransmissionType string                  `json:"transmission_type"`
		Callouts         []models.FitmentCallout `json:"callouts"`
		Note             string                  `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return models.PartFitmentRule{}, err
	}
	calloutsJSON, _ := json.Marshal(req.Callouts)
	return models.PartFitmentRule{
		YearMin:          req.YearMin,
		YearMax:          req.YearMax,
		Make:             strings.TrimSpace(req.Make),
		VehicleModel:     strings.TrimSpace(req.Model),
		Trim:             strings.TrimSpace(req.Trim),
		Cylinders:        strings.TrimSpace(req.Cylinders),
		DisplacementL:    strings.TrimSpace(req.DisplacementL),
		FuelType:         strings.TrimSpace(req.FuelType),
		DriveType:        strings.TrimSpace(req.DriveType),
		BodyType:         strings.TrimSpace(req.BodyType),
		TransmissionType: strings.TrimSpace(req.TransmissionType),
		Callouts:         calloutsJSON,
		Note:             strings.TrimSpace(req.Note),
	}, nil
}
