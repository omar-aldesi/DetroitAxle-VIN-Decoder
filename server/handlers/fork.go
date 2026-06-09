package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"main/auth"
	"main/helpers"
	"main/models"
	"main/services"
)

type ForkHandler struct {
	DB *gorm.DB
}

// isTrustedUser reports whether a user may write authoritative fork data. Same notion the
// edit path uses: any non-agent, or an explicitly trusted agent.
func isTrustedUser(u *models.User) bool {
	return u != nil && (u.Role != "agent" || u.IsTrusted)
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

	pending, err := services.PendingPointsFor(h.DB, buildKey)
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to load pending points")
		return
	}

	resp := gin.H{
		"build_key":   buildKey,
		"fork_fields": keys,
		"fields":      fields,  // field -> [ranges]
		"pending":     pending, // field -> [serials] still awaiting a second sighting
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

// ─── write endpoints (trusted users only) ───────────────────────────────────

type forkPointRequest struct {
	FieldKey string `json:"field_key" binding:"required"`
	Value    string `json:"value" binding:"required"`
}

// RecordForkPoint records a verified single VIN sighting (two-point rule applies).
//
//	POST /api/fork/:vin/point   { field_key, value }
func (h *ForkHandler) RecordForkPoint(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !isTrustedUser(user) {
		helpers.Fail(c, http.StatusForbidden, "only trusted users can record fork data")
		return
	}

	vin := strings.TrimSpace(strings.ToUpper(c.Param("vin")))
	if len(vin) != 17 || !helpers.VinValidator(vin) {
		helpers.Fail(c, http.StatusBadRequest, "a full 17-character VIN is required to record a point")
		return
	}
	serial, ok := services.SerialFromVIN(vin)
	if !ok {
		helpers.Fail(c, http.StatusBadRequest, "VIN has a non-numeric serial; not forkable")
		return
	}

	var req forkPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "field_key and value are required")
		return
	}

	veh, err := h.vehicleByVIN(vin)
	if err != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found")
		return
	}

	engine := services.NewForkEngine(services.NewGormForkStore(h.DB))
	outcome, err := engine.RecordPoint(services.PointInput{
		BuildKey: veh.BuildKey, Serial: serial, FieldKey: req.FieldKey, Value: req.Value,
		Source: user.Role, Verified: true, ActorID: &user.ID, Make: veh.Make, Model: veh.Model,
	})
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if outcome == services.OutcomeNotForkField {
		helpers.Fail(c, http.StatusBadRequest, "'"+req.FieldKey+"' is not a registered fork field")
		return
	}

	helpers.OK(c, gin.H{"outcome": outcome, "build_key": veh.BuildKey, "serial": serial, "field_key": req.FieldKey})
}

type forkRangeRequest struct {
	FieldKey    string `json:"field_key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	SerialStart int64  `json:"serial_start"`
	SerialEnd   *int64 `json:"serial_end"` // nil = open-ended
}

// RecordForkRange records an authoritative manual span (no two-point rule).
//
//	POST /api/fork/:vin/range   { field_key, value, serial_start, serial_end? }
func (h *ForkHandler) RecordForkRange(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !isTrustedUser(user) {
		helpers.Fail(c, http.StatusForbidden, "only trusted users can record fork data")
		return
	}

	vin := strings.TrimSpace(strings.ToUpper(c.Param("vin")))
	var buildKey string
	switch len(vin) {
	case 17:
		if !helpers.VinValidator(vin) {
			helpers.Fail(c, http.StatusBadRequest, "invalid VIN format")
			return
		}
		buildKey = helpers.ExtractBuildKey(vin)
	case 10:
		buildKey = vin
	default:
		helpers.Fail(c, http.StatusBadRequest, "VIN must be 10 or 17 characters long")
		return
	}

	var req forkRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "field_key and value are required")
		return
	}
	if req.SerialStart < 0 {
		helpers.Fail(c, http.StatusBadRequest, "serial_start must be >= 0")
		return
	}
	if req.SerialEnd != nil && *req.SerialEnd < req.SerialStart {
		helpers.Fail(c, http.StatusBadRequest, "serial_end must be >= serial_start")
		return
	}

	var veh models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&veh).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found")
		return
	}

	engine := services.NewForkEngine(services.NewGormForkStore(h.DB))
	outcome, err := engine.RecordRange(services.RangeInput{
		BuildKey: buildKey, SerialStart: req.SerialStart, SerialEnd: req.SerialEnd,
		FieldKey: req.FieldKey, Value: req.Value, Source: user.Role, Verified: true,
		ActorID: &user.ID, Make: veh.Make, Model: veh.Model,
	})
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if outcome == services.OutcomeNotForkField {
		helpers.Fail(c, http.StatusBadRequest, "'"+req.FieldKey+"' is not a registered fork field")
		return
	}

	helpers.OK(c, gin.H{"outcome": outcome, "build_key": buildKey, "field_key": req.FieldKey})
}

// BackfillLegacy seeds the range engine from existing VERIFIED data (rule #2: unverified
// values "don't exist", so they are skipped). For each global fork field it takes the
// latest verified value per vehicle and writes it as a manual base range (build-key-wide,
// source "legacy"). Idempotent — a manual base range simply gets rewritten on re-run.
//
//	POST /api/admin/fork/backfill   (admin only, enforced by the route)
func (h *ForkHandler) BackfillLegacy(c *gin.Context) {
	var keys []string
	if err := h.DB.Model(&models.ForkField{}).
		Where("enabled = ? AND scope_make = '' AND scope_model = ''", true).
		Pluck("key", &keys).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to load fork fields")
		return
	}

	engine := services.NewForkEngine(services.NewGormForkStore(h.DB))
	imported, skipped := 0, 0

	for _, key := range keys {
		// Latest verified value per vehicle for this field (Postgres DISTINCT ON).
		type row struct {
			VehicleID uint
			NewValue  string
		}
		var rows []row
		if err := h.DB.Raw(`
			SELECT DISTINCT ON (vehicle_id) vehicle_id, new_value
			FROM vehicle_field_histories
			WHERE field_name = ? AND is_verified = true
			ORDER BY vehicle_id, created_at DESC
		`, key).Scan(&rows).Error; err != nil {
			helpers.Fail(c, http.StatusInternalServerError, "failed to read history for "+key)
			return
		}

		for _, r := range rows {
			if strings.TrimSpace(r.NewValue) == "" {
				skipped++
				continue
			}
			var veh models.Vehicle
			if err := h.DB.First(&veh, r.VehicleID).Error; err != nil {
				skipped++
				continue
			}
			outcome, err := engine.RecordRange(services.RangeInput{
				BuildKey: veh.BuildKey, SerialStart: 0, SerialEnd: nil, FieldKey: key,
				Value: r.NewValue, Source: "legacy", Verified: true, Make: veh.Make, Model: veh.Model,
			})
			if err == nil && outcome == services.OutcomeManualRange {
				imported++
			} else {
				skipped++
			}
		}
	}

	helpers.OK(c, gin.H{"imported": imported, "skipped": skipped, "fields": keys})
}

func (h *ForkHandler) vehicleByVIN(vin string) (*models.Vehicle, error) {
	buildKey := vin
	if len(vin) == 17 {
		buildKey = helpers.ExtractBuildKey(vin)
	}
	var veh models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&veh).Error; err != nil {
		return nil, err
	}
	return &veh, nil
}
