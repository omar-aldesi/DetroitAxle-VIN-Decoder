package handlers

import (
	"errors"
	"main/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GMHandler struct {
	DB *gorm.DB
}

// DecodeGMLive calls GM Parts Giant for a specific full VIN and returns its
// RPO / build-option codes as JSON.
//
// This data is never persisted — RPO codes are VIN-specific and would corrupt
// a build-key-grouped record if stored there.
//
// We do NOT hard-reject on WMI: some GM-branded vehicles (e.g. upfitter-built
// chassis cabs) carry a non-GM WMI. We let GM Parts Giant be the source of
// truth and return a clean 404 when it has no data for the VIN.
func (h *GMHandler) DecodeGMLive(c *gin.Context) {
	vin := strings.TrimSpace(strings.ToUpper(c.Param("vin")))

	if len(vin) != 17 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a full 17-character VIN is required"})
		return
	}

	attrs, err := services.FetchGMAttributes(vin)
	if err != nil {
		if errors.Is(err, services.ErrGMNoData) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No GM build data is available for this VIN."})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not reach GM Parts Giant. Please try again."})
		return
	}

	data := attrs.FormatAsJSON()
	if data == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No GM build data is available for this VIN."})
		return
	}

	c.JSON(http.StatusOK, data)
}
