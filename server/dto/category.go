package dto

import "main/models"

type CategoryResponse struct {
	ID   uint   `json:"category_id"`
	Name string `json:"name"`
}

func CategoryFromModel(category *models.PartCategory) CategoryResponse {
	return CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}
}
