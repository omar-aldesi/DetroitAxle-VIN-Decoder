package handlers

import (
	"gorm.io/gorm"
)

type VehicleHandler struct {
	DB *gorm.DB
}
