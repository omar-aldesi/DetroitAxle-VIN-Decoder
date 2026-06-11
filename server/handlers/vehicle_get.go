package handlers

import (
	"fmt"
	"log"
	"main/auth"
	dto "main/dto"
	"main/helpers"
	"main/models"
	"main/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *VehicleHandler) GetVehicleById(c *gin.Context) {
	id := c.Param("id")
	var vehicle models.Vehicle
	result := h.DB.First(&vehicle, id)
	if result.Error != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found")
		return
	}
	helpers.OK(c, dto.VehicleFromModel(vehicle))
}

// respondWithVehicle maps the vehicle to a DTO and annotates each note with whether it
// applies to the VIN being viewed (build-number scope), then returns it.
func (h *VehicleHandler) respondWithVehicle(c *gin.Context, vehicle *models.Vehicle, vin string) {
	vin = strings.TrimSpace(strings.ToUpper(vin))
	resp := dto.VehicleFromModel(*vehicle)

	if len(vin) == 17 {
		resp.ViewedVIN = vin
	}
	if known, err := services.KnownVINsForBuildKey(h.DB, vehicle.BuildKey); err != nil {
		log.Printf("known vins: failed to load for %s: %v", vehicle.BuildKey, err)
	} else {
		resp.KnownVINs = known
	}

	if len(resp.Notes) > 0 {
		var viewSerial *int64
		if s, ok := services.SerialFromVIN(vin); ok {
			viewSerial = &s
		}
		boundaries, err := services.ForkBoundaries(h.DB, vehicle.BuildKey)
		if err != nil {
			log.Printf("note scope: failed to load boundaries for %s: %v", vehicle.BuildKey, err)
		} else {
			for i := range resp.Notes {
				resp.Notes[i].Scope = services.NoteScope(boundaries, resp.Notes[i].OriginSerial, viewSerial)
			}
		}
	}
	helpers.OK(c, resp)
}

func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	vin := c.Param("vin")
	user := auth.CurrentUser(c)
	fmt.Println("Current User count: ", user.VinUsageCount)

	if vin == "" {
		helpers.Fail(c, http.StatusBadRequest, "vin is required")
		return
	}

	vinLen := len(vin)
	if vinLen != 10 && vinLen != 17 {
		helpers.Fail(c, http.StatusBadRequest, "VIN must be 10 or 17 characters long")
		return
	}
	if !helpers.VinValidator(vin) {
		helpers.Fail(c, http.StatusBadRequest, "invalid VIN format")
		return
	}

	var buildKey string
	if vinLen == 10 {
		buildKey = vin
	} else {
		buildKey = helpers.ExtractBuildKey(vin)
	}

	// Extracted helper — fire and forget
	incrementVinUsage := func(db *gorm.DB, userID uint) {
		go func() {
			result := db.Model(&models.User{Model: gorm.Model{ID: userID}}).
				Update("vin_usage_count", gorm.Expr("vin_usage_count + 1"))
			if result.Error != nil {
				fmt.Printf("failed to update vin usage count for user %d: %v\n", userID, result.Error)
			}
		}()
	}

	var vehicle models.Vehicle
	result := h.DB.
		Where("build_key = ?", buildKey).
		Preload("Notes", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("Notes.PartCategory").
		Preload("Notes.User").
		Preload("History", func(db *gorm.DB) *gorm.DB {
			return db.Where("tier != ?", "build_number").Order("created_at DESC")
		}).
		First(&vehicle)

	if result.Error == nil {
		// Remember this unit's VIN (check digit + serial) for the build key.
		if vinLen == 17 {
			if err := services.RecordKnownVIN(h.DB, buildKey, vin); err != nil {
				log.Printf("known vin record failed for %s (non-fatal): %v", vin, err)
			}
		}
		// Vehicle found. For GM cars not yet enriched, backfill GM build-key-stable
		// fields once (uses the searched VIN, or the stored example VIN for a
		// 10-char build-key lookup). Non-fatal — display still works if GM is down.
		if !vehicle.GMChecked {
			fullVIN := vin
			if len(fullVIN) != 17 {
				fullVIN = vehicle.ExampleBuildNumber
			}
			if len(fullVIN) == 17 && services.IsGMBrandVIN(fullVIN) {
				if err := services.EnrichExistingWithGM(h.DB, fullVIN, &vehicle); err != nil {
					log.Printf("GM backfill failed for %s (non-fatal): %v", fullVIN, err)
				}
			}
		}
		incrementVinUsage(h.DB, user.ID)
		h.respondWithVehicle(c, &vehicle, vin)
		return
	}

	if result.Error != gorm.ErrRecordNotFound {
		helpers.Fail(c, http.StatusInternalServerError, "database error")
		return
	}

	// Not found — only full VINs can be decoded
	if vinLen != 17 {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found")
		return
	}

	if err := services.DecodeVINAndSave(h.DB, vin, &vehicle); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := services.RecordKnownVIN(h.DB, vehicle.BuildKey, vin); err != nil {
		log.Printf("known vin record failed for %s (non-fatal): %v", vin, err)
	}

	// New decode succeeded — increment and return
	incrementVinUsage(h.DB, user.ID)
	h.respondWithVehicle(c, &vehicle, vin)
}
