package handlers

import (
	"fmt"
	dto "main/dto"
	"main/helpers"
	"main/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	DB *gorm.DB
}

// POST /category/
func (h *CategoryHandler) AddCategory(c *gin.Context) {
	var payload struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	category := models.PartCategory{
		Name: payload.Name,
	}

	if err := h.DB.Create(&category).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, http.StatusConflict, "category with this name already exists")
			return
		}
		helpers.Fail(c, http.StatusInternalServerError, "failed to create category")
		return
	}

	helpers.OK(c, dto.CategoryFromModel(&category))
}

// PATCH /category/:category_id
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Param("category_id"), 10, 64)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid category ID")
		return
	}

	var category models.PartCategory
	if err := h.DB.First(&category, categoryID).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "category not found")
		return
	}

	var payload struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	category.Name = payload.Name

	if err := h.DB.Save(&category).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, http.StatusConflict, "category with this name already exists")
			return
		}
		helpers.Fail(c, http.StatusInternalServerError, "failed to update category")
		return
	}
	helpers.OK(c, dto.CategoryFromModel(&category))
}

// DELETE /category/:category_id
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Param("category_id"), 10, 64)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid category ID")
		return
	}

	var category models.PartCategory
	if err := h.DB.First(&category, categoryID).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "category not found")
		return
	}

	var count int64
	h.DB.Model(&models.AgentNote{}).Where("part_category_id = ?", categoryID).Count(&count)
	if count > 0 {
		helpers.Fail(c, http.StatusConflict, fmt.Sprintf("cannot delete category, it is referenced by %d note(s)", count))
		return
	}

	if err := h.DB.Delete(&category).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to delete category")
		return
	}

	helpers.OK(c, gin.H{"message": "category deleted successfully"})
}

// GET /category/
func (h *CategoryHandler) GetCategories(c *gin.Context) {
	var categories []models.PartCategory
	if err := h.DB.Find(&categories).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch categories")
		return
	}

	result := make([]dto.CategoryResponse, len(categories))
	for i := range categories {
		result[i] = dto.CategoryFromModel(&categories[i])
	}
	helpers.OK(c, result)
}
