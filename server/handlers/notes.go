package handlers

import (
	"log"
	dto "main/dto"
	"main/auth"
	"main/helpers"
	"main/models"
	"main/services"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotesHandler struct {
	DB *gorm.DB
}

func (h *NotesHandler) AddNote(c *gin.Context) {
	vin := c.Param("vin")
	if len(vin) < 17 && len(vin) != 10 {
		helpers.Fail(c, http.StatusBadRequest, "invalid VIN")
		return
	}
	var buildKey string
	if len(vin) == 10 {
		buildKey = vin
	} else {
		buildKey = helpers.ExtractBuildKey(vin)
	}
	var vehicle models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&vehicle).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found for the given VIN")
		return
	}

	var payload dto.CreateNoteRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := payload.Validate(); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Anchor the note to the build number it was entered against (nil for a 10-char
	// build-key entry), so we can later tell whether it applies to a given VIN.
	var originSerial *int64
	if s, ok := services.SerialFromVIN(vin); ok {
		originSerial = &s
	}

	user := auth.CurrentUser(c)
	note := models.AgentNote{
		VehicleID:      vehicle.ID,
		UserID:         user.ID,
		Username:       user.Username,
		NoteType:       payload.NoteType,
		FreeText:       payload.FreeText,
		PartNumber:     payload.PartNumber,
		PartCategoryID: payload.PartCategoryID,
		OriginSerial:   originSerial,
	}

	if err := h.DB.Create(&note).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to create note")
		return
	}

	if err := h.DB.Preload("PartCategory").First(&note, note.ID).Error; err != nil {
		log.Printf("failed to preload note associations for note %d: %v", note.ID, err)
	}

	go h.incrementNoteCount(user.ID, payload.NoteType)

	helpers.OK(c, dto.NoteFromModel(&note))
}

func (h *NotesHandler) incrementNoteCount(userID uint, noteType models.NoteType) {
	var column string
	switch noteType {
	case models.NoteTypePart:
		column = "part_notes_count"
	case models.NoteTypeFreeText:
		column = "free_notes_count"
	default:
		return
	}

	if err := h.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update(column, gorm.Expr(column+" + 1")).Error; err != nil {
		log.Printf("failed to increment %s for user %d: %v", column, userID, err)
	}
}

// PATCH /vehicles/:vin/notes/:note_id
func (h *NotesHandler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("note_id"), 10, 64)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid note ID")
		return
	}

	var note models.AgentNote
	if err := h.DB.Where("id = ?", noteID).First(&note).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "note not found")
		return
	}

	user := auth.CurrentUser(c)
	if note.UserID != user.ID {
		helpers.Fail(c, http.StatusForbidden, "you can only update your own notes")
		return
	}

	var payload dto.UpdateNoteRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		helpers.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate update against the note's existing type
	switch note.NoteType {
	case models.NoteTypeFreeText:
		if payload.FreeText == nil || *payload.FreeText == "" {
			helpers.Fail(c, http.StatusBadRequest, "free_text is required for free_text notes")
			return
		}
		note.FreeText = payload.FreeText
	case models.NoteTypePart:
		if payload.PartNumber == nil || *payload.PartNumber == "" {
			helpers.Fail(c, http.StatusBadRequest, "part_number is required for part_number notes")
			return
		}
		note.PartNumber = payload.PartNumber
		note.PartCategoryID = payload.PartCategoryID
	}

	if err := h.DB.Save(&note).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to update note")
		return
	}

	h.DB.Preload("PartCategory").First(&note, note.ID)

	helpers.OK(c, dto.NoteFromModel(&note))
}

// DELETE /vehicles/:vin/notes/:note_id
func (h *NotesHandler) DeleteNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("note_id"), 10, 64)
	if err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid note ID")
		return
	}

	var note models.AgentNote
	if err := h.DB.Where("id = ?", noteID).First(&note).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "note not found")
		return
	}

	user := auth.CurrentUser(c)
	if note.UserID != user.ID && user.Role != "admin" {
		helpers.Fail(c, http.StatusForbidden, "you can only delete your own notes")
		return
	}

	if err := h.DB.Delete(&note).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to delete note")
		return
	}

	helpers.OK(c, gin.H{"message": "note deleted successfully"})
}

func (h *NotesHandler) GetListingErrorNotes(c *gin.Context) {
	var notes []models.AgentNote
	var pagination helpers.PaginationQuery
	var total int64
	var totalAll int64
	var resolvedCount int64

	if err := c.ShouldBindQuery(&pagination); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid pagination params")
		return
	}

	// 1. Count of unresolved (used for pagination)
	h.DB.Model(&models.AgentNote{}).
		Where("note_type = ? AND is_resolved = ?", models.NoteTypeListingError, false).
		Count(&total)

	// 2. Count of ALL listing error notes regardless of resolution status
	h.DB.Model(&models.AgentNote{}).
		Where("note_type = ?", models.NoteTypeListingError).
		Count(&totalAll)
	h.DB.Model(&models.AgentNote{}).
		Where("note_type = ? AND is_resolved = ?", models.NoteTypeListingError, true).
		Count(&resolvedCount) // add this

	// 3. Fetch the actual records (unresolved only)
	if err := h.DB.
		Where("note_type = ? AND is_resolved = ?", models.NoteTypeListingError, false).
		Preload("PartCategory").
		Preload("User").
		Order("created_at DESC").
		Scopes(helpers.Paginate(pagination)).
		Find(&notes).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to fetch listing error notes")
		return
	}

	helpers.OK(c, gin.H{
		"items":          dto.NotesFromModel(notes),
		"page":           pagination.Page,
		"page_size":      pagination.PageSize,
		"total_count":    total, // unresolved count → drives pagination
		"total_pages":    int(math.Ceil(float64(total) / float64(pagination.PageSize))),
		"total_all":      totalAll, // all notes count → displayed in UI badge/counter
		"resolved_count": resolvedCount,
	})
}

func (h *NotesHandler) MarkAsResolved(c *gin.Context) {
	var req dto.ResolveNoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid resolve request")
		return
	}

	var note models.AgentNote

	if err := h.DB.First(&note, c.Param("note_id")).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "note not found")
		return
	}

	note.IsResolved = *req.IsResolved
	note.ResolveNote = &req.ResolveNote

	if err := h.DB.Save(&note).Error; err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to mark note as resolved")
		return
	}

	helpers.OK(c, gin.H{
		"message": "note marked as resolved successfully",
	})
}
