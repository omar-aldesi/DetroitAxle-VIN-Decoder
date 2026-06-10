package dto

import (
	"main/models"
	"time"
)

type HistoryResponse struct {
	ID        uint   `json:"id"`
	VehicleID uint   `json:"vehicle_id"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	FieldName string `json:"field_name"`
	OldValue  string `json:"old_value"`
	NewValue  string `json:"new_value"`
	IsTrusted bool   `json:"is_trusted"`

	IsVerified bool      `json:"is_verified"`
	VerifierID *uint     `json:"verifier_id"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`

	// Build-number context (fork system).
	OriginSerial *int64 `json:"origin_serial"`
	Tier         string `json:"tier,omitempty"`
}

type FullHistoryResponse struct {
	HistoryResponse
	Vehicle *VehicleResponse `json:"vehicle,omitempty"`
}

func HistoryFromModel(history *models.VehicleFieldHistory) HistoryResponse {
	return HistoryResponse{
		ID:         history.ID,
		VehicleID:  history.VehicleID,
		UserID:     history.UserID,
		Username:   history.Username,
		FieldName:  history.FieldName,
		OldValue:   history.OldValue,
		NewValue:   history.NewValue,
		IsTrusted:  history.IsTrusted,
		IsVerified:   history.IsVerified,
		VerifierID:   history.VerifierID,
		Source:       history.Source,
		CreatedAt:    history.CreatedAt,
		OriginSerial: history.OriginSerial,
		Tier:         history.Tier,
	}
}

func FullHistoryFromModel(history *models.VehicleFieldHistory, vehicle *VehicleResponse) FullHistoryResponse {
	return FullHistoryResponse{
		HistoryResponse: HistoryFromModel(history),
		Vehicle:         vehicle,
	}
}
