package dto

import (
	"encoding/json"
	"main/models"
	"time"
)

/* ── Part summary (list view) ──────────────────────────────────────── */

type PartSummary struct {
	ID         uint      `json:"id"`
	PartNumber string    `json:"part_number"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	RulesCount int       `json:"rules_count"`
	CreatedAt  time.Time `json:"created_at"`
}

func PartSummaryFromModel(p models.CatalogPart) PartSummary {
	return PartSummary{
		ID:         p.Model.ID,
		PartNumber: p.PartNumber,
		Name:       p.Name,
		Category:   p.Category,
		RulesCount: len(p.FitmentRules),
		CreatedAt:  p.Model.CreatedAt,
	}
}

/* ── Callout ─────────────────────────────────────────────────────── */

type CalloutResponse struct {
	Field string `json:"field"`
	Value string `json:"value"`
	Note  string `json:"note"`
}

func calloutsFromJSON(raw []byte) []CalloutResponse {
	if len(raw) == 0 {
		return []CalloutResponse{}
	}
	var out []CalloutResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return []CalloutResponse{}
	}
	return out
}

/* ── Fitment rule ────────────────────────────────────────────────── */

type FitmentRuleResponse struct {
	ID               uint              `json:"id"`
	YearMin          *int              `json:"year_min"`
	YearMax          *int              `json:"year_max"`
	Make             string            `json:"make"`
	Model            string            `json:"model"`
	Trim             string            `json:"trim"`
	Cylinders        string            `json:"cylinders"`
	DisplacementL    string            `json:"displacement_l"`
	FuelType         string            `json:"fuel_type"`
	DriveType        string            `json:"drive_type"`
	BodyType         string            `json:"body_type"`
	TransmissionType string            `json:"transmission_type"`
	Callouts         []CalloutResponse `json:"callouts"`
	Note             string            `json:"note"`
	CreatedAt        time.Time         `json:"created_at"`
}

func RuleFromModel(r models.PartFitmentRule) FitmentRuleResponse {
	return FitmentRuleResponse{
		ID:               r.Model.ID,
		YearMin:          r.YearMin,
		YearMax:          r.YearMax,
		Make:             r.Make,
		Model:            r.VehicleModel,
		Trim:             r.Trim,
		Cylinders:        r.Cylinders,
		DisplacementL:    r.DisplacementL,
		FuelType:         r.FuelType,
		DriveType:        r.DriveType,
		BodyType:         r.BodyType,
		TransmissionType: r.TransmissionType,
		Callouts:         calloutsFromJSON(r.Callouts),
		Note:             r.Note,
		CreatedAt:        r.Model.CreatedAt,
	}
}

/* ── Full part detail ────────────────────────────────────────────── */

type PartResponse struct {
	ID           uint                  `json:"id"`
	PartNumber   string                `json:"part_number"`
	Name         string                `json:"name"`
	Category     string                `json:"category"`
	Description  string                `json:"description"`
	InternalNote string                `json:"internal_note"`
	FitmentRules []FitmentRuleResponse `json:"fitment_rules"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

func PartFromModel(p models.CatalogPart) PartResponse {
	rules := make([]FitmentRuleResponse, len(p.FitmentRules))
	for i, r := range p.FitmentRules {
		rules[i] = RuleFromModel(r)
	}
	return PartResponse{
		ID:           p.Model.ID,
		PartNumber:   p.PartNumber,
		Name:         p.Name,
		Category:     p.Category,
		Description:  p.Description,
		InternalNote: p.InternalNote,
		FitmentRules: rules,
		CreatedAt:    p.Model.CreatedAt,
		UpdatedAt:    p.Model.UpdatedAt,
	}
}

/* ── Fitment query results ───────────────────────────────────────── */

// PartFitResult is returned by the "what parts fit this vehicle?" query.
type PartFitResult struct {
	PartSummary
	FitResult string   `json:"fit_result"` // "exact" | "note"
	FitNotes  []string `json:"fit_notes"`  // callout verification notes
	RuleNote  string   `json:"rule_note"`  // matched rule's descriptive note
}

// CompatiblePartsGroup groups PartFitResult by category.
type CompatiblePartsGroup struct {
	Category string          `json:"category"`
	Parts    []PartFitResult `json:"parts"`
}

// VehicleFitResult is returned by the "what vehicles fit this part?" query.
type VehicleFitResult struct {
	VehicleResponse
	FitResult string   `json:"fit_result"`
	FitNotes  []string `json:"fit_notes"`
	RuleNote  string   `json:"rule_note"` // matched rule's descriptive note
}
