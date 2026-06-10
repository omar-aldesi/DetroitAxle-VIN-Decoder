package handlers

import (
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	dto "main/dto"
	"main/helpers"
	"main/models"
	"main/services"
)

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

	if s := strings.TrimSpace(q.Search); s != "" {
		like := "%" + s + "%"
		db = db.Where("make ILIKE ? OR model ILIKE ? OR build_key ILIKE ? OR trim ILIKE ?", like, like, like, like)
	}

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

	buildKeys := make([]string, len(vehicles))
	for i, v := range vehicles {
		buildKeys[i] = v.BuildKey
	}
	knownMap, err := services.KnownVINsByBuildKeys(h.DB, buildKeys)
	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to load known VINs")
		return
	}

	items := make([]qItem, len(vehicles))
	for i, v := range vehicles {
		mf := missingFields(v, "")
		resp := dto.VehicleFromModel(v)
		resp.KnownVINs = knownMap[v.BuildKey]
		items[i] = qItem{
			VehicleResponse: resp,
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
