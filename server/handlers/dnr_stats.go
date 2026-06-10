package handlers

import (
	"math"
	"time"

	"github.com/gin-gonic/gin"

	"main/helpers"
	"main/models"
)

// GET /dnr/stats
func (h *DNRHandler) GetStats(c *gin.Context) {
	var vehicles []models.Vehicle
	h.DB.Find(&vehicles)

	total := len(vehicles)
	var sumComp float64
	brakeComplete, suspComplete, fullyComplete := 0, 0, 0

	for _, v := range vehicles {
		pct := completeness(v)
		sumComp += pct
		if pct >= 90 {
			fullyComplete++
		}
		if v.FrontBrakeType != "" && v.RearBrakeType != "" &&
			v.FrontRotorSize != "" && v.RearRotorSize != "" {
			brakeComplete++
		}
		if v.FrontSpringType != "" && v.RearSpringType != "" {
			suspComplete++
		}
	}

	avg := 0.0
	if total > 0 {
		avg = math.Round(sumComp/float64(total)*10) / 10
	}

	// Fields filled today (all roles, from history)
	since := time.Now().UTC().Truncate(24 * time.Hour)
	var filledToday int64
	h.DB.Model(&models.VehicleFieldHistory{}).
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Count(&filledToday)

	// DNR fields filled today specifically
	var dnrToday int64
	h.DB.Raw(`
		SELECT COUNT(*) FROM vehicle_field_histories vfh
		JOIN users u ON u.id = vfh.user_id
		WHERE vfh.created_at >= ? AND vfh.deleted_at IS NULL AND u.role = 'dnr'
	`, since).Scan(&dnrToday)

	helpers.OK(c, gin.H{
		"total_vehicles":      total,
		"avg_completeness":    avg,
		"fully_complete":      fullyComplete,
		"brake_complete":      brakeComplete,
		"suspension_complete": suspComplete,
		"fields_filled_today": filledToday,
		"dnr_fills_today":     dnrToday,
		"spec_fields_total":   len(dnrSpecFields),
	})
}
