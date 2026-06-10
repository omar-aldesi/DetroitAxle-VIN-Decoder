package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	dto "main/dto"
	"main/helpers"
	"main/models"
)

// POST /dnr/vehicles — create a vehicle stub with minimal info (manual entry)
func (h *DNRHandler) CreateVehicle(c *gin.Context) {
	var req struct {
		Year     int    `json:"year"      binding:"required"`
		Make     string `json:"make"      binding:"required"`
		Model    string `json:"model"     binding:"required"`
		Trim     string `json:"trim"`
		BuildKey string `json:"build_key"` // optional — generated if omitted
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Derive a build key if not provided: YEAR + first 6 chars of MAKE+MODEL uppercased
	bk := strings.TrimSpace(strings.ToUpper(req.BuildKey))
	if bk == "" {
		base := strings.ToUpper(strings.ReplaceAll(req.Make+req.Model, " ", ""))
		if len(base) > 6 {
			base = base[:6]
		}
		bk = fmt.Sprintf("%d%s", req.Year, base)
	}

	v := models.Vehicle{
		BuildKey: bk,
		Year:     req.Year,
		Make:     strings.TrimSpace(req.Make),
		Model:    strings.TrimSpace(req.Model),
		Trim:     strings.TrimSpace(req.Trim),
	}

	if err := h.DB.Where(models.Vehicle{BuildKey: bk}).FirstOrCreate(&v).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, http.StatusConflict, "a vehicle with this build key already exists")
			return
		}
		helpers.Fail(c, http.StatusInternalServerError, "failed to create vehicle")
		return
	}

	helpers.OK(c, dto.VehicleFromModel(v))
}
