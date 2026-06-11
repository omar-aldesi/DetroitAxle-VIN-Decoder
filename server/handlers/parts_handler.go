package handlers

import (
	"gorm.io/gorm"
)

type PartsHandler struct {
	DB *gorm.DB
}
