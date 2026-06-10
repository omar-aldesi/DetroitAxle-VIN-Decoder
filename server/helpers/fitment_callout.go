package helpers

import (
	"main/models"
	"strings"
)

func evaluateCallout(vehicle models.Vehicle, co models.FitmentCallout) calloutStatus {
	val := strings.TrimSpace(resolveField(vehicle, co.Field))
	if val == "" || val == "0" {
		return calloutMissing
	}
	if strings.EqualFold(val, strings.TrimSpace(co.Value)) {
		return calloutMatch
	}
	return calloutMismatch
}
