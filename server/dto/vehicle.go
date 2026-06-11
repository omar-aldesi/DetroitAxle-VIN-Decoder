package dto

import "main/models"

type VehicleResponse struct {
	ID                 uint     `json:"id"`
	BuildKey           string   `json:"build_key"`
	ExampleBuildNumber string   `json:"example_build_number"`
	ViewedVIN          string   `json:"viewed_vin,omitempty"` // full VIN from this request (17-char lookups)
	KnownVINs          []string `json:"known_vins,omitempty"` // distinct units seen for this build key

	Year      int    `json:"year"`
	Make      string `json:"make"`
	Model     string `json:"model"`
	Trim      string `json:"trim"`
	Series    string `json:"series"`
	BodyType  string `json:"body_type"`
	DriveType string `json:"drive_type"`
	Country   string `json:"country"`

	Cylinders     string `json:"cylinders"`
	DisplacementL string `json:"displacement_l"`
	FuelType      string `json:"fuel_type"`

	TransmissionType string `json:"transmission_type"`
	Speeds           int    `json:"speeds"`

	GVWR string `json:"gvwr_lbs"`

	ABS string `json:"abs"`

	FrontBrakeType string `json:"front_brake_type"`
	RearBrakeType  string `json:"rear_brake_type"`

	CustomFields map[string]any `json:"custom_fields"`

	BrakeSystemType     string `json:"brake_system_type"`
	Doors               string `json:"doors"`
	EngineConfiguration string `json:"engine_configuration"`

	GMChecked bool `json:"gm_checked"`

	Notes   []NoteResponse    `json:"notes"`
	History []HistoryResponse `json:"history"`
}

func VehicleFromModel(v models.Vehicle) VehicleResponse {
	return VehicleResponse{
		ID:                  v.ID,
		BuildKey:            v.BuildKey,
		ExampleBuildNumber:  v.ExampleBuildNumber,
		Year:                v.Year,
		Make:                v.Make,
		Model:               v.Model,
		Trim:                v.Trim,
		Series:              v.Series,
		BodyType:            v.BodyType,
		DriveType:           v.DriveType,
		Country:             v.Country,
		Cylinders:           v.Cylinders,
		DisplacementL:       v.DisplacementL,
		FuelType:            v.FuelType,
		TransmissionType:    v.TransmissionType,
		Speeds:              v.Speeds,
		GVWR:                v.GVWR,
		ABS:                 v.ABS,
		FrontBrakeType:      v.FrontBrakeType,
		RearBrakeType:       v.RearBrakeType,
		CustomFields:        v.CustomFields,
		BrakeSystemType:     v.BrakeSystemType,
		Doors:               v.Doors,
		EngineConfiguration: v.EngineConfiguration,
		GMChecked:           v.GMChecked,
		Notes:               NotesFromModels(v.Notes),
		History:             HistoryFromModels(v.History),
	}
}

type VehicleResponseWithNoteCount struct {
	VehicleResponse
	NotesCount int64 `json:"notes_count"`
}

func NotesFromModels(notes []models.AgentNote) []NoteResponse {
	result := make([]NoteResponse, len(notes))
	for i, n := range notes {
		result[i] = NoteFromModel(&n)
	}
	return result
}

func HistoryFromModels(history []models.VehicleFieldHistory) []HistoryResponse {
	result := make([]HistoryResponse, len(history))
	for i, h := range history {
		result[i] = HistoryFromModel(&h)
	}
	return result
}

func VehicleWithNoteCountFromModel(v *models.Vehicle, count int64) VehicleResponseWithNoteCount {
	return VehicleResponseWithNoteCount{
		VehicleResponse: VehicleFromModel(*v),
		NotesCount:      count,
	}
}

func VehiclesFromModels(vehicles []models.Vehicle) []VehicleResponseWithNoteCount {
	result := make([]VehicleResponseWithNoteCount, len(vehicles))
	for i, v := range vehicles {
		result[i] = VehicleResponseWithNoteCount{
			VehicleResponse: VehicleFromModel(v),
			NotesCount:      int64(len(v.Notes)),
		}
	}
	return result
}
