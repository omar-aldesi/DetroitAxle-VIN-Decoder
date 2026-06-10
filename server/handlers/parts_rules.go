package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "main/dto"
	"main/auth"
	"main/helpers"
	"main/models"
)

// POST /parts/:id/rules
func (h *PartsHandler) AddRule(c *gin.Context) {
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
	rule, err := bindRule(c)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule.PartID = part.ID
	if err := h.DB.Create(&rule).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to create rule")
		return
	}
	helpers.OK(c, dto.RuleFromModel(rule))
}

// PATCH /parts/:id/rules/:rule_id
func (h *PartsHandler) UpdateRule(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !canEditParts(user) {
		helpers.Fail(c, http.StatusForbidden, "insufficient permissions")
		return
	}
	var rule models.PartFitmentRule
	if err := h.DB.Where("id = ? AND part_id = ?", c.Param("rule_id"), c.Param("id")).
		First(&rule).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "rule not found")
		return
	}
	updated, err := bindRule(c)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	updated.Model.ID = rule.Model.ID
	updated.PartID = rule.PartID
	if err := h.DB.Save(&updated).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to update rule")
		return
	}
	helpers.OK(c, dto.RuleFromModel(updated))
}

// DELETE /parts/:id/rules/:rule_id
func (h *PartsHandler) DeleteRule(c *gin.Context) {
	user := auth.CurrentUser(c)
	if !canEditParts(user) {
		helpers.Fail(c, http.StatusForbidden, "insufficient permissions")
		return
	}
	var rule models.PartFitmentRule
	if err := h.DB.Where("id = ? AND part_id = ?", c.Param("rule_id"), c.Param("id")).
		First(&rule).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "rule not found")
		return
	}
	h.DB.Delete(&rule)
	helpers.OK(c, gin.H{"message": "rule deleted"})
}
