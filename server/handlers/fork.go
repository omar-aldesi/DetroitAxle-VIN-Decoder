package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"main/helpers"
	"main/models"
	"main/services"
)

type ForkHandler struct {
	DB *gorm.DB
}

// GetForkData returns the build-number range data for a VIN or build key.
//
//	GET /api/fork/:vin
//
// For a full 17-char VIN it also resolves each fork field's value for that specific
// build number. For a 10-char build key it returns every field's full range list only.
func (h *ForkHandler) GetForkData(c *gin.Context) {
	vin := strings.TrimSpace(strings.ToUpper(c.Param("vin")))

	var buildKey string
	var serialPtr *int64
	switch len(vin) {
	case 17:
		if !helpers.VinValidator(vin) {
			helpers.Fail(c, http.StatusBadRequest, "invalid VIN format")
			return
		}
		buildKey = helpers.ExtractBuildKey(vin)
		if s, ok := services.SerialFromVIN(vin); ok {
			serialPtr = &s
		}
	case 10:
		buildKey = vin
	default:
		helpers.Fail(c, http.StatusBadRequest, "VIN must be 10 or 17 characters long")
		return
	}

	var veh models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&veh).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found")
		return
	}

	keys, err := services.ForkFieldKeysFor(h.DB, veh.Make, veh.Model)
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to load fork fields")
		return
	}

	engine := services.NewForkEngine(services.NewGormForkStore(h.DB))

	fields, err := engine.ResolveByBuildKey(buildKey, keys)
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to resolve ranges")
		return
	}

	resp := gin.H{
		"build_key":   buildKey,
		"fork_fields": keys,
		"fields":      fields, // field -> [ranges]
	}

	if serialPtr != nil {
		resolved, err := engine.Resolve(buildKey, *serialPtr, keys)
		if err != nil {
			helpers.Fail(c, http.StatusInternalServerError, "failed to resolve serial")
			return
		}
		resp["serial"] = *serialPtr
		resp["resolved"] = resolved // field -> value for this exact VIN
	}

	helpers.OK(c, resp)
}
