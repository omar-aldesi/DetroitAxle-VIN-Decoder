package handlers

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	dto "main/dto"
	"main/helpers"
	"main/models"
)

// ruleCandidate is the scan target for the SQL pre-filter join query.
// We pull rule fields + enough part info to build the response,
// without loading any extra columns.
type ruleCandidate struct {
	// From part_fitment_rules
	RuleID           uint           `gorm:"column:rule_id"`
	YearMin          *int           `gorm:"column:year_min"`
	YearMax          *int           `gorm:"column:year_max"`
	RuleMake         string         `gorm:"column:rule_make"`
	RuleModel        string         `gorm:"column:rule_model"`
	RuleTrim         string         `gorm:"column:rule_trim"`
	Cylinders        string         `gorm:"column:cylinders"`
	DisplacementL    string         `gorm:"column:displacement_l"`
	FuelType         string         `gorm:"column:fuel_type"`
	DriveType        string         `gorm:"column:drive_type"`
	BodyType         string         `gorm:"column:body_type"`
	TransmissionType string         `gorm:"column:transmission_type"`
	Callouts         datatypes.JSON `gorm:"column:callouts"`
	Note             string         `gorm:"column:note"`
	// From catalog_parts
	CatalogID  uint   `gorm:"column:catalog_id"`
	PartNumber string `gorm:"column:part_number"`
	PartName   string `gorm:"column:part_name"`
	Category   string `gorm:"column:category"`
}

func (rc ruleCandidate) toRule() models.PartFitmentRule {
	return models.PartFitmentRule{
		YearMin:          rc.YearMin,
		YearMax:          rc.YearMax,
		Make:             rc.RuleMake,
		VehicleModel:     rc.RuleModel,
		Trim:             rc.RuleTrim,
		Cylinders:        rc.Cylinders,
		DisplacementL:    rc.DisplacementL,
		FuelType:         rc.FuelType,
		DriveType:        rc.DriveType,
		BodyType:         rc.BodyType,
		TransmissionType: rc.TransmissionType,
		Callouts:         rc.Callouts,
		Note:             rc.Note,
	}
}

// GET /parts/by-vehicle/:vin
//
// Returns compatible parts grouped by category.
//
// Scale design: instead of loading all 100K parts + rules into memory, a
// single SQL query JOINs part_fitment_rules with catalog_parts and pre-filters
// by the vehicle's indexed fields (year, make, cylinders, displacement, etc.).
// PostgreSQL reduces the result set to the relevant subset; Go then evaluates
// callouts on that small set.  Model/trim/body_type are substring checks done
// in Go after the SQL pass, not in SQL, to keep the index path clean.
func (h *PartsHandler) GetCompatibleParts(c *gin.Context) {
	vin := strings.TrimSpace(strings.ToUpper(c.Param("vin")))

	var buildKey string
	switch len(vin) {
	case 17:
		buildKey = helpers.ExtractBuildKey(vin)
	case 10:
		buildKey = vin
	default:
		helpers.Fail(c, http.StatusBadRequest, "VIN must be 10 or 17 characters")
		return
	}

	var vehicle models.Vehicle
	if err := h.DB.Where("build_key = ?", buildKey).First(&vehicle).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "vehicle not found")
		return
	}

	// Only the highly selective indexed columns are filtered here.
	// Model, trim, body_type, and callouts are handled in Go after.
	var candidates []ruleCandidate
	err := h.DB.Raw(`
		SELECT
			pfr.id          AS rule_id,
			pfr.year_min,
			pfr.year_max,
			pfr.make        AS rule_make,
			pfr.model       AS rule_model,
			pfr.trim        AS rule_trim,
			pfr.cylinders,
			pfr.displacement_l,
			pfr.fuel_type,
			pfr.drive_type,
			pfr.body_type,
			pfr.transmission_type,
			pfr.callouts,
			pfr.note,
			cp.id           AS catalog_id,
			cp.part_number,
			cp.name         AS part_name,
			cp.category
		FROM part_fitment_rules pfr
		JOIN catalog_parts cp ON cp.id = pfr.part_id
		WHERE pfr.deleted_at IS NULL
		  AND cp.deleted_at  IS NULL
		  AND (pfr.year_min  IS NULL OR pfr.year_min <= ?)
		  AND (pfr.year_max  IS NULL OR pfr.year_max >= ?)
		  AND (pfr.make        = '' OR LOWER(pfr.make)        = LOWER(?))
		  AND (pfr.cylinders   = '' OR LOWER(pfr.cylinders)   = LOWER(?))
		  AND (pfr.fuel_type   = '' OR LOWER(pfr.fuel_type)   = LOWER(?))
		  AND (pfr.transmission_type = '' OR LOWER(pfr.transmission_type) = LOWER(?))
	`,
		// Intentionally excluded from SQL — handled in Go with correct semantics:
		//   displacement_l: numeric float comparison ("3.5" == "3.50")
		//   drive_type:     normalization ("FOUR WHEEL DRIVE" == "4WD")
		//   model/trim:     token-based, order-independent matching
		//   body_type:      token-based matching
		// year + make alone reduces 100K+ rules to a manageable Go-evaluation set.
		vehicle.Year, vehicle.Year,
		vehicle.Make,
		vehicle.Cylinders,
		vehicle.FuelType,
		vehicle.TransmissionType,
	).Scan(&candidates).Error

	if err != nil {
		helpers.Fail(c, http.StatusInternalServerError, "failed to query compatible parts")
		return
	}

	type partEntry struct {
		summary dto.PartSummary
		rules   []models.PartFitmentRule
	}
	partMap := map[uint]*partEntry{}
	for _, rc := range candidates {
		if _, ok := partMap[rc.CatalogID]; !ok {
			partMap[rc.CatalogID] = &partEntry{
				summary: dto.PartSummary{
					ID:         rc.CatalogID,
					PartNumber: rc.PartNumber,
					Name:       rc.PartName,
					Category:   rc.Category,
				},
			}
		}
		partMap[rc.CatalogID].rules = append(partMap[rc.CatalogID].rules, rc.toRule())
	}

	categoryMap := map[string][]dto.PartFitResult{}
	for _, entry := range partMap {
		eval := helpers.BestFitForPart(vehicle, entry.rules)
		if eval.Result == helpers.FitNone {
			continue
		}
		notes := eval.Notes
		if notes == nil {
			notes = []string{}
		}
		categoryMap[entry.summary.Category] = append(
			categoryMap[entry.summary.Category],
			dto.PartFitResult{
				PartSummary: entry.summary,
				FitResult:   helpers.FitResultString(eval.Result),
				FitNotes:    notes,
				RuleNote:    eval.RuleNote,
			},
		)
	}

	groups := make([]dto.CompatiblePartsGroup, 0, len(categoryMap))
	for cat, parts := range categoryMap {
		// Within each category: exact fits first, then alphabetical by part number
		sort.Slice(parts, func(i, j int) bool {
			if parts[i].FitResult != parts[j].FitResult {
				return parts[i].FitResult == "exact"
			}
			return parts[i].PartNumber < parts[j].PartNumber
		})
		groups = append(groups, dto.CompatiblePartsGroup{Category: cat, Parts: parts})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Category < groups[j].Category })

	helpers.OK(c, gin.H{
		"vehicle_id":  vehicle.ID,
		"build_key":   vehicle.BuildKey,
		"total_parts": countParts(groups),
		"groups":      groups,
	})
}

func countParts(groups []dto.CompatiblePartsGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Parts)
	}
	return n
}

// GET /parts/:id/vehicles?page=1&page_size=20
//
// Scale note: vehicle count is small (≤ ~10K), so loading candidates via SQL
// and evaluating callouts in Go is fine here.
func (h *PartsHandler) GetCompatibleVehicles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var part models.CatalogPart
	if err := h.DB.Preload("FitmentRules").First(&part, c.Param("id")).Error; err != nil {
		helpers.Fail(c, http.StatusNotFound, "part not found")
		return
	}

	// SQL pre-filter: only apply clean, exact-match columns that reliably narrow
	// the candidate set without risking false negatives.
	//
	// Excluded from SQL (handled by Go's EvaluateRule):
	//   model        — token-based, word-order-independent matching
	//   trim         — comma-separated list of valid values
	//   drive_type   — normalisation required ("FOUR WHEEL DRIVE" == "4WD")
	//   body_type    — token-based matching
	candidateIDs := map[uint]struct{}{}
	for _, rule := range part.FitmentRules {
		db := h.DB.Model(&models.Vehicle{})
		if rule.YearMin != nil {
			db = db.Where("year >= ?", *rule.YearMin)
		}
		if rule.YearMax != nil {
			db = db.Where("year <= ?", *rule.YearMax)
		}
		if rule.Make != "" {
			db = db.Where("LOWER(make) = LOWER(?)", rule.Make)
		}
		if rule.Cylinders != "" {
			db = db.Where("LOWER(cylinders) = LOWER(?)", rule.Cylinders)
		}
		if rule.DisplacementL != "" {
			db = db.Where("LOWER(displacement_l) = LOWER(?)", rule.DisplacementL)
		}
		if rule.FuelType != "" {
			db = db.Where("LOWER(fuel_type) = LOWER(?)", rule.FuelType)
		}
		if rule.TransmissionType != "" {
			db = db.Where("LOWER(transmission_type) = LOWER(?)", rule.TransmissionType)
		}

		var ids []uint
		db.Pluck("id", &ids)
		for _, id := range ids {
			candidateIDs[id] = struct{}{}
		}
	}

	if len(candidateIDs) == 0 {
		helpers.OK(c, gin.H{
			"items": []dto.VehicleFitResult{}, "total_count": 0,
			"exact_count": 0, "note_count": 0,
		})
		return
	}

	ids := make([]uint, 0, len(candidateIDs))
	for id := range candidateIDs {
		ids = append(ids, id)
	}

	var vehicles []models.Vehicle
	h.DB.Where("id IN ?", ids).Find(&vehicles)

	var results []dto.VehicleFitResult
	exactCount, noteCount := 0, 0
	for _, v := range vehicles {
		eval := helpers.BestFitForPart(v, part.FitmentRules)
		if eval.Result == helpers.FitNone {
			continue
		}
		notes := eval.Notes
		if notes == nil {
			notes = []string{}
		}
		fr := helpers.FitResultString(eval.Result)
		if fr == "exact" {
			exactCount++
		} else {
			noteCount++
		}
		results = append(results, dto.VehicleFitResult{
			VehicleResponse: dto.VehicleFromModel(v),
			FitResult:       fr,
			FitNotes:        notes,
			RuleNote:        eval.RuleNote,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].FitResult != results[j].FitResult {
			return results[i].FitResult == "exact"
		}
		return results[i].Year > results[j].Year
	})

	total := len(results)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	helpers.OK(c, gin.H{
		"items":       results[start:end],
		"total_count": total,
		"exact_count": exactCount,
		"note_count":  noteCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": int(math.Ceil(float64(total) / float64(pageSize))),
	})
}
