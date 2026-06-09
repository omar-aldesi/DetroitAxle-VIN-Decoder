package handlers

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"main/auth"
	dto "main/DTO"
	"main/helpers"
	"main/models"
)

// ── Spec field catalogue (build-key-tier fields the DNR team researches) ──────
//
// NOTE: the build-NUMBER-tier fields (brake_code, front/rear_rotor_size,
// front/rear_spring_type, steering_type) are intentionally NOT here — they vary
// per individual VIN and are managed by the fork/range engine, not column edits.
// Keeping them out also stops Propagate from spreading per-VIN data across build keys.
var dnrSpecFields = []struct {
	Key      string
	Category string
}{
	{"trim", "identity"},
	{"series", "identity"},
	{"body_type", "identity"},
	{"doors", "identity"},
	{"drive_type", "identity"},
	{"country", "identity"},
	{"cylinders", "engine"},
	{"displacement_l", "engine"},
	{"fuel_type", "engine"},
	{"engine_configuration", "engine"},
	{"transmission_type", "transmission"},
	{"speeds", "transmission"},
	{"abs", "brakes"},
	{"brake_system_type", "brakes"},
	{"front_brake_type", "brakes"},
	{"rear_brake_type", "brakes"},
	{"gvwr_lbs", "brakes"},
}

// fieldValue extracts a spec field value from a Vehicle as a string.
func fieldValue(v models.Vehicle, key string) string {
	switch key {
	case "trim":
		return v.Trim
	case "series":
		return v.Series
	case "body_type":
		return v.BodyType
	case "doors":
		return v.Doors
	case "drive_type":
		return v.DriveType
	case "country":
		return v.Country
	case "cylinders":
		return v.Cylinders
	case "displacement_l":
		return v.DisplacementL
	case "fuel_type":
		return v.FuelType
	case "engine_configuration":
		return v.EngineConfiguration
	case "transmission_type":
		return v.TransmissionType
	case "speeds":
		if v.Speeds == 0 {
			return ""
		}
		return strconv.Itoa(v.Speeds)
	case "abs":
		return v.ABS
	case "brake_system_type":
		return v.BrakeSystemType
	case "front_brake_type":
		return v.FrontBrakeType
	case "rear_brake_type":
		return v.RearBrakeType
	case "gvwr_lbs":
		return v.GVWR
	}
	return ""
}

func missingFields(v models.Vehicle, categoryFilter string) []string {
	out := []string{}
	for _, f := range dnrSpecFields {
		if categoryFilter != "" && categoryFilter != "all" && f.Category != categoryFilter {
			continue
		}
		val := strings.TrimSpace(fieldValue(v, f.Key))
		if val == "" || val == "0" {
			out = append(out, f.Key)
		}
	}
	return out
}

func completeness(v models.Vehicle) float64 {
	total := len(dnrSpecFields)
	filled := 0
	for _, f := range dnrSpecFields {
		val := strings.TrimSpace(fieldValue(v, f.Key))
		if val != "" && val != "0" {
			filled++
		}
	}
	return math.Round(float64(filled)/float64(total)*1000) / 10 // one decimal
}

// ── Handler ────────────────────────────────────────────────────────────

type DNRHandler struct {
	DB *gorm.DB
}

// GET /dnr/queue
func (h *DNRHandler) GetQueue(c *gin.Context) {
	type Q struct {
		helpers.PaginationQuery
		Missing       string `form:"missing"` // brakes | suspension | engine | transmission | identity
		Search        string `form:"q"`
		Make          string `form:"make"`
		Model         string `form:"model"`
		Year          int    `form:"year"`
		YearMin       int    `form:"year_min"`
		YearMax       int    `form:"year_max"`
		Cylinders     string `form:"cylinders"`
		DisplacementL string `form:"displacement_l"`
		FuelType      string `form:"fuel_type"`
		DriveType     string `form:"drive_type"`
		BodyType      string `form:"body_type"`
		TransType     string `form:"transmission_type"`
	}
	var q Q
	if err := c.ShouldBindQuery(&q); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid query params")
		return
	}

	db := h.DB.Model(&models.Vehicle{})

	// ── Text search ──────────────────────────────────────────────────
	if s := strings.TrimSpace(q.Search); s != "" {
		like := "%" + s + "%"
		db = db.Where("make ILIKE ? OR model ILIKE ? OR build_key ILIKE ? OR trim ILIKE ?", like, like, like, like)
	}

	// ── Exact / range filters ────────────────────────────────────────
	if m := strings.TrimSpace(q.Make); m != "" {
		db = db.Where("LOWER(make) = LOWER(?)", m)
	}
	if m := strings.TrimSpace(q.Model); m != "" {
		db = db.Where("model ILIKE ?", "%"+m+"%")
	}
	if q.Year > 0 {
		db = db.Where("year = ?", q.Year)
	}
	if q.YearMin > 0 {
		db = db.Where("year >= ?", q.YearMin)
	}
	if q.YearMax > 0 {
		db = db.Where("year <= ?", q.YearMax)
	}
	if c_ := strings.TrimSpace(q.Cylinders); c_ != "" {
		db = db.Where("cylinders = ?", c_)
	}
	if d := strings.TrimSpace(q.DisplacementL); d != "" {
		db = db.Where("displacement_l = ?", d)
	}
	if f := strings.TrimSpace(q.FuelType); f != "" {
		db = db.Where("LOWER(fuel_type) = LOWER(?)", f)
	}
	if d := strings.TrimSpace(q.DriveType); d != "" {
		db = db.Where("LOWER(drive_type) = LOWER(?)", d)
	}
	if b := strings.TrimSpace(q.BodyType); b != "" {
		db = db.Where("LOWER(body_type) = LOWER(?)", b)
	}
	if t := strings.TrimSpace(q.TransType); t != "" {
		db = db.Where("LOWER(transmission_type) = LOWER(?)", t)
	}

	// ── Missing-category filter ──────────────────────────────────────
	// (brakes/suspension fork fields are excluded — they're build-number-tier now)
	switch q.Missing {
	case "brakes":
		db = db.Where("(front_brake_type IS NULL OR front_brake_type = '' OR rear_brake_type IS NULL OR rear_brake_type = '' OR gvwr_lbs IS NULL OR gvwr_lbs = '')")
	case "engine":
		db = db.Where("(cylinders IS NULL OR cylinders = '' OR displacement_l IS NULL OR displacement_l = '')")
	case "transmission":
		db = db.Where("(transmission_type IS NULL OR transmission_type = '' OR speeds = 0 OR speeds IS NULL)")
	}

	var total int64
	db.Count(&total)

	var vehicles []models.Vehicle
	if err := db.Find(&vehicles).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch vehicles")
		return
	}

	// Compute completeness per vehicle, then sort ascending (least complete first)
	type qItem struct {
		dto.VehicleResponse
		Completeness  float64  `json:"completeness"`
		FilledCount   int      `json:"filled_count"`
		TotalFields   int      `json:"total_fields"`
		MissingFields []string `json:"missing_fields"`
	}

	items := make([]qItem, len(vehicles))
	for i, v := range vehicles {
		mf := missingFields(v, "")
		items[i] = qItem{
			VehicleResponse: dto.VehicleFromModel(v),
			Completeness:    completeness(v),
			FilledCount:     len(dnrSpecFields) - len(mf),
			TotalFields:     len(dnrSpecFields),
			MissingFields:   mf,
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Completeness < items[j].Completeness
	})

	// Manual pagination after sort
	page := q.Page
	pageSize := q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}

	helpers.OK(c, helpers.PaginatedData{
		Items:      items[start:end],
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

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
		"total_vehicles":     total,
		"avg_completeness":   avg,
		"fully_complete":     fullyComplete,
		"brake_complete":     brakeComplete,
		"suspension_complete": suspComplete,
		"fields_filled_today": filledToday,
		"dnr_fills_today":    dnrToday,
		"spec_fields_total":  len(dnrSpecFields),
	})
}

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
		VehicleID    uint     `json:"vehicle_id"`
		BuildKey     string   `json:"build_key"`
		Year         int      `json:"year"`
		Make         string   `json:"make"`
		Model        string   `json:"model"`
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
			// NOTE (fork system): propagation writes the transitional columns and records
			// history, but intentionally does NOT feed the build-number range engine. A
			// propagated value is a cross-build-key *guess* (same make/model/year), not a
			// real per-VIN sighting, so it must never become authoritative fork data. The
			// engine is fed only by direct VIN edits and explicit manual ranges.
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
			VehicleID:    target.ID,
			BuildKey:     target.BuildKey,
			Year:         target.Year,
			Make:         target.Make,
			Model:        target.Model,
			AppliedFields: applied,
		})
	}

	helpers.OK(c, gin.H{
		"dry_run":       req.DryRun,
		"source_vehicle": gin.H{"id": source.ID, "build_key": source.BuildKey},
		"updated_count": updatedCount,
		"affected":      results,
	})
}
