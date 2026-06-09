package dto

import (
	"errors"
	"main/models"
	"time"
)

type NoteResponse struct {
	VehicleID      uint              `json:"vehicle_id"`
	UserID         uint              `json:"user_id"`
	ID             uint              `json:"note_id"`
	NoteType       string            `json:"note_type"`
	FreeText       *string           `json:"free_text"`
	PartNumber     *string           `json:"part_number"`
	PartCategoryID *uint             `json:"part_category_id"`
	PartCategory   *CategoryResponse `json:"part_category"`
	Username       string            `json:"username"`
	IsResolved     bool              `json:"is_resolved"`
	ResolveNote    *string           `json:"resolve_note"`
	CreatedAt      time.Time         `json:"created_at"`

	// Build-number scope. OriginSerial is the serial the note was entered against (nil =
	// build-key-wide). Scope ("applies" | "warn" | "") is computed per viewed VIN by the
	// vehicle handler; empty when not evaluated in a per-VIN context.
	OriginSerial *int64 `json:"origin_serial"`
	Scope        string `json:"scope,omitempty"`
}

type CreateNoteRequest struct {
	NoteType       models.NoteType `json:"note_type"        binding:"required"`
	FreeText       *string         `json:"free_text"`
	PartNumber     *string         `json:"part_number"`
	PartCategoryID *uint           `json:"part_category_id"`
}

type UpdateNoteRequest struct {
	FreeText       *string `json:"free_text"`
	PartNumber     *string `json:"part_number"`
	PartCategoryID *uint   `json:"part_category_id"`
}

type ResolveNoteRequest struct {
	IsResolved  *bool  `json:"is_resolved" binding:"required"`
	ResolveNote string `json:"resolve_note" binding:"required"`
}

func (p *CreateNoteRequest) Validate() error {
	switch p.NoteType {
	case models.NoteTypeFreeText:
		if p.FreeText == nil || *p.FreeText == "" {
			return errors.New("free_text is required for free_text notes")
		}
	case models.NoteTypePart:
		if p.PartNumber == nil || *p.PartNumber == "" {
			return errors.New("part_number is required for part_number notes")
		}
	case models.NoteTypeListingError:
		if p.FreeText == nil || *p.FreeText == "" || p.PartNumber == nil || *p.PartNumber == "" {
			return errors.New("free_text and part_number are required for listing_error notes")
		}
	default:
		return errors.New("invalid note_type, must be free_text or part_number")
	}
	return nil
}

func NoteFromModel(note *models.AgentNote) NoteResponse {
	var category *CategoryResponse
	if note.PartCategory != nil {
		c := CategoryFromModel(note.PartCategory)
		category = &c
	}
	return NoteResponse{
		VehicleID:      note.VehicleID,
		UserID:         note.UserID,
		ID:             note.ID,
		NoteType:       string(note.NoteType),
		FreeText:       note.FreeText,
		PartNumber:     note.PartNumber,
		PartCategoryID: note.PartCategoryID,
		PartCategory:   category,
		Username:       note.Username,
		IsResolved:     note.IsResolved,
		ResolveNote:    note.ResolveNote,
		CreatedAt:      note.CreatedAt,
		OriginSerial:   note.OriginSerial,
	}
}

func NotesFromModel(notes []models.AgentNote) []NoteResponse {
	result := make([]NoteResponse, 0, len(notes))
	for _, note := range notes {
		result = append(result, NoteFromModel(&note))
	}
	return result
}
