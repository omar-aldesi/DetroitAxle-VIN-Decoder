package handlers

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	dto "main/dto"
	"main/auth"
	"main/helpers"
	"main/models"
)

// GET /parts?q=&category=&brand=&page=1&page_size=25
func (h *PartsHandler) ListParts(c *gin.Context) {
	search := strings.TrimSpace(c.Query("q"))
	category := strings.TrimSpace(c.Query("category"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	db := h.DB.Model(&models.CatalogPart{})

	// Only search indexed, short columns — searching description on 100K rows
	// with ILIKE causes full table scans. Agents search by part number or name.
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("part_number ILIKE ? OR name ILIKE ?", like, like)
	}
	if category != "" {
		db = db.Where("LOWER(category) = LOWER(?)", category)
	}

	var total int64
	db.Count(&total)

	var parts []models.CatalogPart
	if err := db.Preload("FitmentRules").
		Order("category ASC, part_number ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&parts).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch parts")
		return
	}

	summaries := make([]dto.PartSummary, len(parts))
	for i, p := range parts {
		summaries[i] = dto.PartSummaryFromModel(p)
	}

	helpers.OK(c, helpers.PaginatedData{
		Items:      summaries,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

// POST /parts/:id/clone — duplicate a part with all its fitment rules
func (h *PartsHandler) ClonePart(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !canEditParts(user) {
		helpers.Fail(c, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		PartNumber string `json:"part_number" binding:"required"`
		Name       string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var source models.CatalogPart
	if err := h.DB.Preload("FitmentRules").First(&source, c.Param("id")).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "part not found")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = source.Name
	}

	clone := models.CatalogPart{
		PartNumber:   strings.TrimSpace(req.PartNumber),
		Name:         name,
		Category:     source.Category,
		Description:  source.Description,
		InternalNote: source.InternalNote,
	}
	if err := h.DB.Create(&clone).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, http.StatusConflict, "part number already exists")
			return
		}
		helpers.Fail(c, http.StatusInternalServerError, "failed to clone part")
		return
	}

	// Copy all fitment rules
	for _, r := range source.FitmentRules {
		newRule := models.PartFitmentRule{
			PartID:           clone.ID,
			YearMin:          r.YearMin,
			YearMax:          r.YearMax,
			Make:             r.Make,
			VehicleModel:     r.VehicleModel,
			Trim:             r.Trim,
			Cylinders:        r.Cylinders,
			DisplacementL:    r.DisplacementL,
			FuelType:         r.FuelType,
			DriveType:        r.DriveType,
			BodyType:         r.BodyType,
			TransmissionType: r.TransmissionType,
			Callouts:         r.Callouts,
			Note:             r.Note,
		}
		h.DB.Create(&newRule)
	}

	h.DB.Preload("FitmentRules").First(&clone, clone.ID)
	helpers.OK(c, dto.PartFromModel(clone))
}

// GET /parts/categories
func (h *PartsHandler) ListCategories(c *gin.Context) {
	var cats []string
	h.DB.Model(&models.CatalogPart{}).
		Distinct("category").
		Where("category != ''").
		Order("category ASC").
		Pluck("category", &cats)
	helpers.OK(c, cats)
}

// GET /parts/:id
func (h *PartsHandler) GetPart(c *gin.Context) {
	var part models.CatalogPart
	if err := h.DB.Preload("FitmentRules").First(&part, c.Param("id")).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "part not found")
		return
	}
	helpers.OK(c, dto.PartFromModel(part))
}

// POST /parts
func (h *PartsHandler) CreatePart(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !canEditParts(user) {
		helpers.Fail(c, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		PartNumber   string `json:"part_number" binding:"required"`
		Name         string `json:"name"         binding:"required"`
		Category     string `json:"category"`
		Description  string `json:"description"`
		InternalNote string `json:"internal_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	part := models.CatalogPart{
		PartNumber:   strings.TrimSpace(req.PartNumber),
		Name:         strings.TrimSpace(req.Name),
		Category:     strings.TrimSpace(req.Category),
		Description:  strings.TrimSpace(req.Description),
		InternalNote: strings.TrimSpace(req.InternalNote),
	}
	if err := h.DB.Create(&part).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, http.StatusConflict, "part number already exists")
			return
		}
		helpers.Fail(c, http.StatusInternalServerError, "failed to create part")
		return
	}
	helpers.OK(c, dto.PartFromModel(part))
}

// PATCH /parts/:id
func (h *PartsHandler) UpdatePart(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !canEditParts(user) {
		helpers.Fail(c, http.StatusForbidden, "insufficient permissions")
		return
	}
	var part models.CatalogPart
	if err := h.DB.First(&part, c.Param("id")).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "part not found")
		return
	}
	var req struct {
		PartNumber   *string `json:"part_number"`
		Name         *string `json:"name"`
		Category     *string `json:"category"`
		Description  *string `json:"description"`
		InternalNote *string `json:"internal_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	updates := map[string]any{}
	if req.PartNumber != nil {
		updates["part_number"] = strings.TrimSpace(*req.PartNumber)
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Category != nil {
		updates["category"] = strings.TrimSpace(*req.Category)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.InternalNote != nil {
		updates["internal_note"] = strings.TrimSpace(*req.InternalNote)
	}

	if err := h.DB.Model(&part).Updates(updates).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to update part")
		return
	}
	h.DB.Preload("FitmentRules").First(&part, part.ID)
	helpers.OK(c, dto.PartFromModel(part))
}

// DELETE /parts/:id
func (h *PartsHandler) DeletePart(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !canEditParts(user) {
		helpers.Fail(c, http.StatusForbidden, "insufficient permissions")
		return
	}
	var part models.CatalogPart
	if err := h.DB.First(&part, c.Param("id")).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "part not found")
		return
	}
	// Soft-delete rules first (GORM soft-delete via gorm.Model.DeletedAt)
	h.DB.Where("part_id = ?", part.ID).Delete(&models.PartFitmentRule{})
	if err := h.DB.Delete(&part).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to delete part")
		return
	}
	helpers.OK(c, gin.H{"message": "part deleted"})
}
