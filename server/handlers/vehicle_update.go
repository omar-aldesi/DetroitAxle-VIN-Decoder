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
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (h *VehicleHandler) UpdateVehicle(c *gin.Context) {
	user := auth.CurrentUser(c)
	vin := c.Param("vin")
	if vin == "" {
		helpers.Fail(c, http.StatusBadRequest, "vin is required")
		return
	}

	var buildKey string
	if len(vin) == 10 {
		buildKey = vin
	} else if len(vin) == 17 {
		buildKey = helpers.ExtractBuildKey(vin)
	} else {
		helpers.Fail(c, http.StatusBadRequest, "invalid VIN")
		return
	}

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(payload) == 0 {
		helpers.Fail(c, http.StatusBadRequest, "no fields provided for update")
		return
	}

	// Extract optional source annotation (DNR use) before processing
	source, _ := payload["_source"].(string)

	// Remove read-only / meta fields
	delete(payload, "_source")
	delete(payload, "id")
	delete(payload, "build_key")
	delete(payload, "notes")
	delete(payload, "history")
	delete(payload, "created_at")
	delete(payload, "updated_at")

	// 1. Fetch current vehicle (need old values for history diff)
	var current models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&current).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found for the given VIN")
		return
	}

	if len(vin) == 17 {
		if err := services.RecordKnownVIN(h.DB, buildKey, vin); err != nil {
			log.Printf("known vin record failed for %s (non-fatal): %v", vin, err)
		}
	}

	// 2. Check field permissions for this user's role
	var permissions []models.FieldPermission
	if err := h.DB.Where("role = ?", user.Role).Find(&permissions).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to load permissions")
		return
	}

	denied := make(map[string]bool, len(permissions))
	for _, p := range permissions {
		if !p.CanEdit {
			denied[p.FieldName] = true
		}
	}

	deniedFields := []string{}
	for fieldName := range payload {
		if denied[fieldName] {
			deniedFields = append(deniedFields, fieldName)
		}
	}
	if len(deniedFields) > 0 {
		helpers.Fail(c, http.StatusForbidden, "you do not have permission to edit these fields")
		return
	}
	// 3. Cast nested maps to JSONMap
	for k, v := range payload {
		if m, ok := v.(map[string]any); ok {
			payload[k] = datatypes.JSONMap(m)
		}
	}

	// 4. Diff old vs new values and build history entries
	currentMap := helpers.StructToMap(current)
	historyEntries := []models.VehicleFieldHistory{}
	isTrusted := user.Role != "agent" || user.IsTrusted

	// Build-number context: which serial this edit targets (nil = build-key-wide).
	var serialPtr *int64
	if s, ok := services.SerialFromVIN(vin); ok {
		serialPtr = &s
	}
	// Fork-field edits to feed the range engine. DNR/admin write directly; agents
	// (trusted or not) reach the engine only after verification (VerifyEntry).
	feedsForkOnEdit := user.Role == "dnr" || user.Role == "admin"
	type forkFeed struct{ field, value string }
	var forkFeeds []forkFeed

	for fieldName, newVal := range payload {
		oldVal := currentMap[fieldName]
		oldStr := fmt.Sprintf("%v", oldVal)
		newStr := fmt.Sprintf("%v", newVal)
		if oldStr == newStr {
			continue
		}

		tier := "build_key"
		var entrySerial *int64
		if services.IsForkFieldKey(h.DB, current.Make, current.Model, fieldName) {
			tier = "build_number"
			entrySerial = serialPtr
			if feedsForkOnEdit {
				forkFeeds = append(forkFeeds, forkFeed{fieldName, newStr})
			}
		}

		historyEntries = append(historyEntries, models.VehicleFieldHistory{
			VehicleID:    uint(current.ID),
			UserID:       user.ID,
			Username:     user.Username,
			FieldName:    fieldName,
			OldValue:     oldStr,
			NewValue:     newStr,
			IsTrusted:    isTrusted,
			Source:       source,
			OriginSerial: entrySerial,
			Tier:         tier,
		})
	}

	// 5. Apply update
	payload["updated_at"] = time.Now()
	if err := h.DB.Model(&models.Vehicle{}).Where("build_key = ?", buildKey).Updates(payload).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 6. Insert history entries
	if len(historyEntries) > 0 {
		if err := h.DB.Create(&historyEntries).Error; err != nil {
			// Non-fatal — update succeeded, just log it
			log.Printf("failed to write field history for vehicle %s: %v", buildKey, err)
		}
	}

	// 6b. Feed DNR/admin fork-field edits into the build-number range engine immediately.
	// Agent edits (trusted or not) are NOT fed here — they reach the engine only once
	// verified (see HistoryHandler.VerifyEntry). Non-fatal: a failure never blocks the edit.
	if len(forkFeeds) > 0 {
		feedSource := source
		if feedSource == "" {
			feedSource = user.Role
		}
		for _, f := range forkFeeds {
			if _, err := services.FeedForkField(h.DB, buildKey, current.Make, current.Model,
				f.field, f.value, serialPtr, feedSource, &user.ID); err != nil {
				log.Printf("fork feed failed for %s.%s: %v", buildKey, f.field, err)
			}
		}
	}

	var updated models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&updated).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch updated vehicle")
		return
	}

	userID := user.ID

	go func() {
		if err := h.DB.Model(&models.User{}).
			Where("id = ?", userID).
			Update("updates_count", gorm.Expr("updates_count + 1")).Error; err != nil {
			log.Printf("failed to update updates count for user %d: %v", userID, err)
		}
	}()

	helpers.OK(c, dto.VehicleFromModel(updated))
}
