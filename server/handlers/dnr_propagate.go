package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"main/auth"
	"main/helpers"
	"main/models"
)

// GET /dnr/similar?vehicle_id=123&criteria=same_model_year
// Preview how many vehicles would be affected by propagation.
func (h *DNRHandler) GetSimilar(c *gin.Context) {
	vehicleID := c.Query("vehicle_id")
	criteria := c.DefaultQuery("criteria", "same_model_year")

	var source models.Vehicle
	if err := h.DB.First(&source, vehicleID).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "source vehicle not found")
		return
	}

	db := h.DB.Model(&models.Vehicle{}).
		Where("id != ? AND LOWER(make) = LOWER(?) AND LOWER(model) = LOWER(?) AND year = ?",
			source.ID, source.Make, source.Model, source.Year)

	if criteria == "same_engine" {
		db = db.Where("cylinders = ? AND displacement_l = ?",
			source.Cylinders, source.DisplacementL)
	}

	var count int64
	db.Count(&count)

	helpers.OK(c, gin.H{
		"similar_count": count,
		"criteria":      criteria,
		"source": gin.H{
			"id":    source.ID,
			"year":  source.Year,
			"make":  source.Make,
			"model": source.Model,
		},
	})
}

// POST /dnr/propagate
func (h *DNRHandler) Propagate(c *gin.Context) {
	user := auth.CurrentUser(c)

	var req struct {
		SourceVehicleID uint     `json:"source_vehicle_id" binding:"required"`
		Fields          []string `json:"fields" binding:"required"`
		Criteria        string   `json:"criteria"` // "same_model_year" | "same_engine"
		Source          string   `json:"source"`
		DryRun          bool     `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var source models.Vehicle
	if err := h.DB.First(&source, req.SourceVehicleID).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "source vehicle not found")
		return
	}

	// Build allowed field set
	allowed := map[string]bool{}
	for _, f := range dnrSpecFields {
		allowed[f.Key] = true
	}

	// Find similar vehicles
	db := h.DB.Where(
		"id != ? AND LOWER(make) = LOWER(?) AND LOWER(model) = LOWER(?) AND year = ?",
		source.ID, source.Make, source.Model, source.Year,
	)
	if req.Criteria == "same_engine" {
		db = db.Where("cylinders = ? AND displacement_l = ?",
			source.Cylinders, source.DisplacementL)
	}

	var targets []models.Vehicle
	db.Find(&targets)

	type Result struct {
		VehicleID     uint     `json:"vehicle_id"`
		BuildKey      string   `json:"build_key"`
		Year          int      `json:"year"`
		Make          string   `json:"make"`
		Model         string   `json:"model"`
		AppliedFields []string `json:"applied_fields"`
	}

	results := []Result{}
	updatedCount := 0

	for _, target := range targets {
		payload := map[string]any{}
		applied := []string{}

		for _, fk := range req.Fields {
			if !allowed[fk] {
				continue
			}
			srcVal := strings.TrimSpace(fieldValue(source, fk))
			tgtVal := strings.TrimSpace(fieldValue(target, fk))
			// Only fill fields that are empty in the target
			if tgtVal == "" && srcVal != "" {
				payload[fk] = srcVal
				applied = append(applied, fk)
			}
		}

		if len(applied) == 0 {
			continue
		}

		if !req.DryRun {
			// Propagation does not feed the fork engine (cross-build-key guess, not a VIN sighting).
			err := h.DB.Transaction(func(tx *gorm.DB) error {
				payload["updated_at"] = time.Now()
				if err := tx.Model(&models.Vehicle{}).
					Where("id = ?", target.ID).
					Updates(payload).Error; err != nil {
					return err
				}

				entries := make([]models.VehicleFieldHistory, 0, len(applied))
				for _, fk := range applied {
					entries = append(entries, models.VehicleFieldHistory{
						VehicleID: target.ID,
						UserID:    user.ID,
						Username:  user.Username,
						FieldName: fk,
						OldValue:  fieldValue(target, fk),
						NewValue:  fmt.Sprintf("%v", payload[fk]),
						IsTrusted: true, // DNR propagations are always trusted
						Source:    req.Source,
					})
				}
				return tx.Create(&entries).Error
			})
			if err != nil {
				continue
			}
			updatedCount++
		}

		results = append(results, Result{
			VehicleID:     target.ID,
			BuildKey:      target.BuildKey,
			Year:          target.Year,
			Make:          target.Make,
			Model:         target.Model,
			AppliedFields: applied,
		})
	}

	helpers.OK(c, gin.H{
		"dry_run":        req.DryRun,
		"source_vehicle": gin.H{"id": source.ID, "build_key": source.BuildKey},
		"updated_count":  updatedCount,
		"affected":       results,
	})
}
