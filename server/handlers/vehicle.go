package handlers

import (
	"fmt"
	"log"
	dto "main/DTO"
	"main/auth"
	"main/helpers"
	"main/models"
	"main/services"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type VehicleHandler struct {
	DB *gorm.DB
}

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
	resp := dto.VehicleFromModel(*vehicle)
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
			return db.Order("created_at DESC")
		}).
		First(&vehicle)

	if result.Error == nil {
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

	// New decode succeeded — increment and return
	incrementVinUsage(h.DB, user.ID)
	h.respondWithVehicle(c, &vehicle, vin)
}

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
	// Fork-field edits to feed the range engine (only verified/trusted ones — rule #2).
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
			if isTrusted {
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

	// 6b. Feed verified (trusted) fork-field edits into the build-number range engine.
	// Untrusted agent edits are NOT fed here — they reach the engine only once verified
	// (see HistoryHandler.VerifyEntry). Non-fatal: a failure never blocks the edit.
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

func (h *VehicleHandler) ListVehicles(c *gin.Context) {
	type ListVehiclesQuery struct {
		helpers.PaginationQuery
		Search string `form:"q"`
	}
	var query ListVehiclesQuery
	var total int64

	if err := c.ShouldBindQuery(&query); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid query params")
		return
	}

	db := h.DB.Model(&models.Vehicle{})
	if q := strings.TrimSpace(query.Search); q != "" {
		normalized := strings.ToLower(q)
		normalized = strings.ReplaceAll(normalized, "-", " ")
		normalized = strings.ReplaceAll(normalized, "_", " ")
		tokens := strings.Fields(normalized)
		for _, token := range tokens {
			like := "%" + token + "%"
			db = db.Where(
				`CAST(year AS TEXT) LIKE ? OR make ILIKE ? OR REPLACE(model, '-', '') ILIKE ? OR model ILIKE ?`,
				like, like, like, like,
			)
		}
	}

	db.Count(&total)

	// intermediate struct for GORM to scan into
	type vehicleWithCount struct {
		models.Vehicle
		NotesCount int64 `gorm:"column:notes_count"`
	}

	var vehicles []vehicleWithCount
	err := db.Select("vehicles.*, (SELECT COUNT(*) FROM agent_notes WHERE agent_notes.vehicle_id = vehicles.id) as notes_count").
		Scopes(helpers.Paginate(query.PaginationQuery)).
		Find(&vehicles).Error
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch vehicles")
		return
	}

	// map to DTOs
	result := make([]dto.VehicleResponseWithNoteCount, len(vehicles))
	for i, v := range vehicles {
		result[i] = dto.VehicleResponseWithNoteCount{
			VehicleResponse: dto.VehicleFromModel(v.Vehicle),
			NotesCount:      v.NotesCount,
		}
	}

	helpers.OK(c, helpers.PaginatedData{
		Items:      result,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalCount: total,
		TotalPages: int(math.Ceil(float64(total) / float64(query.PageSize))),
	})
}
