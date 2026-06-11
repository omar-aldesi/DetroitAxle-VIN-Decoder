package dto

import "main/models"

type UserResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	IsTrusted bool   `json:"is_trusted"`
	Role      string `json:"role"`

	// Usage counters — used by the agent leaderboard on the admin dashboard
	VinUsageCount  int `json:"vin_usage_count"`
	FreeNotesCount int `json:"free_notes_count"`
	PartNotesCount int `json:"part_notes_count"`
	UpdatesCount   int `json:"updates_count"`
}

type UserSummary struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func UserFromModel(user *models.User) UserResponse {
	return UserResponse{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		IsActive:       user.IsActive,
		IsTrusted:      user.IsTrusted,
		Role:           user.Role,
		VinUsageCount:  user.VinUsageCount,
		FreeNotesCount: user.FreeNotesCount,
		PartNotesCount: user.PartNotesCount,
		UpdatesCount:   user.UpdatesCount,
	}
}
