package handlers

import (
	dto "main/dto"
	"main/helpers"
	"main/models"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct {
	DB *gorm.DB
}

func (h *AdminHandler) AdminStatus(c *gin.Context) {
	var vehiclesCount int64
	var userCount int64
	var notesCount int64
	var notesTodayCount int64
	var categoriesCount int64
	var freeTextCount int64
	var partNumberCount int64

	type Totals struct {
		TotalFreeNotes int64 `json:"total_free_notes"`
		TotalPartNotes int64 `json:"total_part_notes"`
		TotalVinUsage  int64 `json:"total_vin_usage"`
		TotalUpdates   int64 `json:"total_updates"`
	}
	var totals Totals

	if err := h.DB.Model(&models.Vehicle{}).Count(&vehiclesCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.AgentNote{}).Count(&notesCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.AgentNote{}).Where("created_at >= ?", time.Now().Add(-24*time.Hour)).Count(&notesTodayCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.PartCategory{}).Count(&categoriesCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.AgentNote{}).Where("note_type = ?", models.NoteTypeFreeText).Count(&freeTextCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.AgentNote{}).Where("note_type = ?", models.NoteTypePart).Count(&partNumberCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	if err := h.DB.Model(&models.User{}).
		Select("SUM(free_notes_count) as total_free_notes, SUM(part_notes_count) as total_part_notes, SUM(vin_usage_count) as total_vin_usage, SUM(updates_count) as total_updates").
		Scan(&totals).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	var listingErrorCount int64
	if err := h.DB.Model(&models.AgentNote{}).Where("note_type = ?", models.NoteTypeListingError).Count(&listingErrorCount).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	helpers.OK(c, gin.H{
		"vehicles_count":       vehiclesCount,
		"user_count":           userCount,
		"notes_count":          notesCount,
		"notes_today_count":    notesTodayCount,
		"categories_count":     categoriesCount,
		"free_notes_count":     freeTextCount,
		"part_notes_count":     partNumberCount,
		"listing_errors_count": listingErrorCount,
		"total_free_notes":     totals.TotalFreeNotes,
		"total_part_notes":     totals.TotalPartNotes,
		"total_vin_usage":      totals.TotalVinUsage,
		"total_updates":        totals.TotalUpdates,
	})
}
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	var pagination helpers.PaginationQuery
	var total int64

	if err := c.ShouldBindQuery(&pagination); err != nil {
		helpers.Fail(c, 400, "invalid pagination params")
		return
	}

	h.DB.Model(&models.User{}).Count(&total)

	if err := h.DB.Offset((pagination.Page - 1) * pagination.PageSize).
		Limit(pagination.PageSize).
		Find(&users).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	result := make([]dto.UserResponse, len(users))
	for i := range users {
		result[i] = dto.UserFromModel(&users[i])
	}

	helpers.OK(c, helpers.PaginatedData{
		Items:      result,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalCount: total,
		TotalPages: int(math.Ceil(float64(total) / float64(pagination.PageSize))),
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	// 1. Find existing user
	var user models.User
	if err := h.DB.First(&user, c.Param("id")).Error; err != nil {
		helpers.Fail(c, 404, "user not found")
		return
	}

	// 2. Bind incoming JSON into a generic map to capture only sent fields
	var updates map[string]any
	if err := c.BindJSON(&updates); err != nil {
		helpers.Fail(c, 400, err.Error())
		return
	}

	// 3. If password is provided, hash it and remap to hashed_password
	if rawPassword, ok := updates["password"].(string); ok && rawPassword != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
		if err != nil {
			helpers.Fail(c, 500, "failed to hash password")
			return
		}
		updates["hashed_password"] = string(hashed)
		delete(updates, "password")
	}

	// 4. Strip fields that should never be updated via this endpoint
	for _, protected := range []string{"id", "created_at", "deleted_at"} {
		delete(updates, protected)
	}

	// 5. Use Model + Updates for a true partial update (only sent fields)
	if err := h.DB.Model(&user).Updates(updates).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	// 6. Re-fetch to return the fresh record
	h.DB.First(&user, user.ID)
	helpers.OK(c, dto.UserFromModel(&user))
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	var user models.User
	if err := h.DB.First(&user, c.Param("id")).Error; err != nil {
		helpers.Fail(c, 404, "user not found")
		return
	}

	if err := h.DB.Delete(&user).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	helpers.OK(c, gin.H{"message": "user deleted"})
}

// NotesChartPoint represents a single data point for the notes chart
type NotesChartPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// RecentNoteResponse is the flattened response for recent notes
type RecentNoteResponse struct {
	ID           uint      `json:"id"`
	VehicleID    uint      `json:"vehicle_id"`
	Username     string    `json:"username"`
	NoteType     string    `json:"note_type"`
	FreeText     *string   `json:"free_text"`
	PartNumber   *string   `json:"part_number"`
	PartCategory *string   `json:"part_category"`
	CreatedAt    time.Time `json:"created_at"`
}

// GET /admin/stats/notes-chart?days=14
func (h *AdminHandler) GetNotesChart(c *gin.Context) {
	daysParam := c.DefaultQuery("days", "14")
	days, err := strconv.Atoi(daysParam)
	if err != nil || days <= 0 || days > 365 {
		helpers.Fail(c, http.StatusBadRequest, "invalid 'days' parameter (1–365)")
		return
	}

	since := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -days+1)

	type row struct {
		Date  string `gorm:"column:note_date"`
		Count int64  `gorm:"column:note_count"`
	}

	var rows []row
	err = h.DB.Raw(`
		SELECT DATE(created_at)::text AS note_date, COUNT(*) AS note_count
		FROM agent_notes
		WHERE created_at >= ? AND deleted_at IS NULL
		GROUP BY DATE(created_at)
		ORDER BY note_date ASC
	`, since).Scan(&rows).Error

	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to query notes chart")
		return
	}

	// Build a full date-range map so days with 0 notes are included
	dateMap := make(map[string]int64, days)

	for i := 0; i < days; i++ {
		d := since.AddDate(0, 0, i).Format("2006-01-02")
		dateMap[d] = 0
	}
	for _, r := range rows {
		dateMap[r.Date] = r.Count
	}

	result := make([]NotesChartPoint, 0, days)
	for i := 0; i < days; i++ {
		d := since.AddDate(0, 0, i).Format("2006-01-02")
		result = append(result, NotesChartPoint{Date: d, Count: dateMap[d]})
	}

	helpers.OK(c, result)
}

// GET /admin/notes/recent?limit=15
func (h *AdminHandler) GetRecentNotes(c *gin.Context) {
	limitParam := c.DefaultQuery("limit", "15")
	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 || limit > 100 {
		helpers.Fail(c, http.StatusBadRequest, "invalid 'limit' parameter (1–100)")
		return
	}

	var notes []models.AgentNote
	err = h.DB.
		Preload("PartCategory").
		Order("created_at DESC").
		Limit(limit).
		Find(&notes).Error

	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to query recent notes")
		return
	}

	result := make([]RecentNoteResponse, 0, len(notes))
	for _, n := range notes {
		resp := RecentNoteResponse{
			ID:         n.ID,
			VehicleID:  n.VehicleID,
			Username:   n.Username,
			NoteType:   string(n.NoteType),
			FreeText:   n.FreeText,
			PartNumber: n.PartNumber,
			CreatedAt:  n.CreatedAt,
		}
		if n.PartCategory != nil {
			resp.PartCategory = &n.PartCategory.Name
		}
		result = append(result, resp)
	}

	helpers.OK(c, result)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	type RegisterRequest struct {
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"    binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Role     string `json:"role"     binding:"required"`
		IsActive bool   `json:"isActive"`
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, 400, err.Error())
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	newUser := models.User{
		Email:          req.Email,
		Username:       req.Username,
		HashedPassword: string(bytes),
		Role:           req.Role,
		IsActive:       req.IsActive,
	}

	if err := h.DB.Create(&newUser).Error; err != nil {
		// Postgres/SQLite unique constraint violation
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, 409, "email or username already taken")
			return
		}
		helpers.Fail(c, 500, "could not create user")
		return
	}

	helpers.OK(c, gin.H{"message": "user created successfully"})
}
