package handlers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	dto "main/DTO"
	"main/auth"
	"main/helpers"
	"main/models"
	"main/services"
)

type HistoryHandler struct {
	DB *gorm.DB
}

// GET /history/?include_vehicle=true&filter=all&search=&page=1&page_size=20
func (h *HistoryHandler) ListHistory(c *gin.Context) {
	includeVehicle := c.Query("include_vehicle") == "true"
	filter := c.DefaultQuery("filter", "all") // "all" | "trusted" | "not-trusted"
	search := strings.TrimSpace(c.Query("search"))

	var pagination helpers.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid pagination params")
		return
	}

	// Latest unverified record per (vehicle_id, field_name)
	subQuery := h.DB.Model(&models.VehicleFieldHistory{}).
		Select("MAX(id)").
		Where("is_verified = ?", false).
		Group("vehicle_id, field_name")

	// applySearch adds a vehicle JOIN + WHERE when a search term is present.
	// Uses ILIKE for case-insensitive matching on PostgreSQL.
	applySearch := func(q *gorm.DB) *gorm.DB {
		if search == "" {
			return q
		}
		like := "%" + search + "%"
		return q.
			Select("vehicle_field_histories.*").
			Joins("LEFT JOIN vehicles ON vehicles.id = vehicle_field_histories.vehicle_id").
			Where(
				h.DB.Where("vehicle_field_histories.field_name ILIKE ?", like).
					Or("vehicle_field_histories.new_value ILIKE ?", like).
					Or("vehicle_field_histories.old_value ILIKE ?", like).
					Or("vehicle_field_histories.username ILIKE ?", like).
					Or("vehicles.make ILIKE ?", like).
					Or("vehicles.model ILIKE ?", like).
					Or("vehicles.build_key ILIKE ?", like).
					Or("vehicles.trim ILIKE ?", like),
			)
	}

	base := func() *gorm.DB {
		return applySearch(
			h.DB.Model(&models.VehicleFieldHistory{}).
				Where("vehicle_field_histories.id IN (?)", subQuery),
		)
	}

	var notTrustedCount, trustedCount int64
	base().Where("vehicle_field_histories.is_trusted = ?", false).Count(&notTrustedCount)
	base().Where("vehicle_field_histories.is_trusted = ?", true).Count(&trustedCount)

	query := base()
	switch filter {
	case "trusted":
		query = query.Where("vehicle_field_histories.is_trusted = ?", true)
	case "not-trusted":
		query = query.Where("vehicle_field_histories.is_trusted = ?", false)
	}

	var total int64
	query.Count(&total)

	query = query.Order("vehicle_field_histories.created_at DESC")
	if includeVehicle {
		query = query.Preload("Vehicle")
	}

	var history []models.VehicleFieldHistory
	if err := query.Scopes(helpers.Paginate(pagination)).Find(&history).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch history")
		return
	}

	result := make([]dto.FullHistoryResponse, len(history))
	for i := range history {
		var vehicleResp *dto.VehicleResponse
		if includeVehicle && history[i].Vehicle != nil {
			v := dto.VehicleFromModel(*history[i].Vehicle)
			vehicleResp = &v
		}
		result[i] = dto.FullHistoryFromModel(&history[i], vehicleResp)
	}

	helpers.OK(c, gin.H{
		"items":             result,
		"page":              pagination.Page,
		"page_size":         pagination.PageSize,
		"total_count":       total,
		"total_pages":       int(math.Ceil(float64(total) / float64(pagination.PageSize))),
		"not_trusted_count": notTrustedCount,
		"trusted_count":     trustedCount,
	})
}

// DELETE /history/:id
func (h *HistoryHandler) DeleteEntry(c *gin.Context) {
	user := auth.CurrentUser(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var entry models.VehicleFieldHistory
	if err := h.DB.First(&entry, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			helpers.Fail(c, http.StatusNotFound, "entry not found")
		} else {
			helpers.Fail(c, http.StatusInternalServerError, "failed to fetch entry")
		}
		return
	}

	// Only admin or the person who made the change can delete it
	if user.Role != "admin" && entry.UserID != user.ID {
		helpers.Fail(c, http.StatusForbidden, "not authorized to delete this entry")
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// Revert the vehicle field to its previous value.
		// Use raw Exec instead of Updates(map) so GORM never skips the value
		// when OldValue is an empty string or other zero value — which is the
		// exact case when an agent filled a previously-blank field and then
		// wants to undo that change.
		sql := fmt.Sprintf(
			`UPDATE vehicles SET %s = $1, updated_at = NOW() WHERE id = $2`,
			entry.FieldName,
		)
		if err := tx.Exec(sql, entry.OldValue, entry.VehicleID).Error; err != nil {
			return err
		}
		// Remove the history entry
		return tx.Delete(&entry).Error
	})
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to delete entry")
		return
	}

	helpers.OK(c, gin.H{"message": "entry deleted", "reverted_field": entry.FieldName, "reverted_to": entry.OldValue})
}

// PATCH /history/:id/verify
func (h *HistoryHandler) VerifyEntry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	var requestBody struct {
		VerifierID     uint    `json:"verifier_id" binding:"required"`
		CorrectedValue *string `json:"corrected_value"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "verifier_id is required")
		return
	}

	allowedColumns := map[string]bool{
		"year":                 true,
		"make":                 true,
		"model":                true,
		"trim":                 true,
		"series":               true,
		"body_type":            true,
		"drive_type":           true,
		"country":              true,
		"cylinders":            true,
		"displacement_l":       true,
		"fuel_type":            true,
		"transmission_type":    true,
		"speeds":               true,
		"gvwr_lbs":             true,
		"abs":                  true,
		"front_brake_type":     true,
		"rear_brake_type":      true,
		"rear_spring_type":     true,
		"front_spring_type":    true,
		"steering_type":        true,
		"brake_code":           true,
		"front_rotor_size":     true,
		"rear_rotor_size":      true,
		"brake_system_type":    true,
		"doors":                true,
		"engine_configuration": true,
		"example_build_number": true,
	}

	var entry models.VehicleFieldHistory
	if err := h.DB.First(&entry, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			helpers.Fail(c, http.StatusNotFound, "entry not found")
		} else {
			helpers.Fail(c, http.StatusInternalServerError, "failed to fetch entry")
		}
		return
	}

	if requestBody.CorrectedValue != nil {
		if !allowedColumns[entry.FieldName] {
			helpers.Fail(c, http.StatusBadRequest, fmt.Sprintf("field '%s' is not allowed", entry.FieldName))
			return
		}
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// Mark this entry verified
		if err := tx.Model(&models.VehicleFieldHistory{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"is_verified": true,
				"verifier_id": requestBody.VerifierID,
			}).Error; err != nil {
			return err
		}

		// Auto-close any older unverified edits to the same field on the same
		// vehicle — they're superseded by the one we just verified, so they
		// should never resurface in the review queue.
		if err := tx.Model(&models.VehicleFieldHistory{}).
			Where("vehicle_id = ? AND field_name = ? AND id < ? AND is_verified = ?",
				entry.VehicleID, entry.FieldName, id, false).
			Update("is_verified", true).Error; err != nil {
			return err
		}

		if requestBody.CorrectedValue != nil {
			// Use raw Exec so an empty corrected value is applied correctly
			sql := fmt.Sprintf(
				`UPDATE vehicles SET %s = $1, updated_at = NOW() WHERE id = $2`,
				entry.FieldName,
			)
			if err := tx.Exec(sql, *requestBody.CorrectedValue, entry.VehicleID).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to apply updates")
		return
	}

	// Feed the now-verified value into the build-number range engine. Trusted entries were
	// already fed at edit time, so only feed previously-untrusted ones here. Non-fatal.
	if !entry.IsTrusted {
		var veh models.Vehicle
		if err := h.DB.First(&veh, entry.VehicleID).Error; err == nil {
			value := entry.NewValue
			if requestBody.CorrectedValue != nil {
				value = *requestBody.CorrectedValue
			}
			verifier := requestBody.VerifierID
			if _, err := services.FeedForkField(h.DB, veh.BuildKey, veh.Make, veh.Model,
				entry.FieldName, value, entry.OriginSerial, "verified", &verifier); err != nil {
				log.Printf("fork feed on verify failed for %s.%s: %v", veh.BuildKey, entry.FieldName, err)
			}
		}
	}

	helpers.OK(c, gin.H{
		"id":              id,
		"is_verified":     true,
		"verifier_id":     requestBody.VerifierID,
		"corrected_value": requestBody.CorrectedValue,
	})
}
