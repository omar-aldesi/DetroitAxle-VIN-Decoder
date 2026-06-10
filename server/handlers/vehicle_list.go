package handlers

import (
	dto "main/dto"
	"main/helpers"
	"main/models"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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
