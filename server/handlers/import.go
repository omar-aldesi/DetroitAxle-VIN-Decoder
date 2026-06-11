package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"main/auth"
	"main/helpers"
	"main/models"
	"main/services"
)

// --- In-memory job store ---

type ImportJob struct {
	ID            string            `json:"id"`
	CreatedBy     uint              `json:"created_by"`
	CreatedByName string            `json:"created_by_name"`
	Status        string            `json:"status"` // running|completed|cancelled
	Total         int               `json:"total"`
	Processed     int               `json:"processed"`
	Succeeded     int               `json:"succeeded"`
	Failed        int               `json:"failed"`
	Skipped       int               `json:"skipped"`
	Invalid       int               `json:"invalid"`
	Results       []VINImportResult `json:"results"`
	StartedAt     time.Time         `json:"started_at"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`

	mu     sync.RWMutex
	cancel context.CancelFunc
}

type VINImportResult struct {
	VIN        string `json:"vin"`
	Status     string `json:"status"` // success|failed|skipped|invalid
	IsGM       bool   `json:"is_gm"`
	Make       string `json:"make,omitempty"`
	Model      string `json:"model,omitempty"`
	Year       int    `json:"year,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// ImportJobSnapshot is a safe copy for JSON serialization.
type ImportJobSnapshot struct {
	ID            string            `json:"id"`
	CreatedBy     uint              `json:"created_by"`
	CreatedByName string            `json:"created_by_name"`
	Status        string            `json:"status"`
	Total         int               `json:"total"`
	Processed     int               `json:"processed"`
	Succeeded     int               `json:"succeeded"`
	Failed        int               `json:"failed"`
	Skipped       int               `json:"skipped"`
	Invalid       int               `json:"invalid"`
	Results       []VINImportResult `json:"results"`
	StartedAt     time.Time         `json:"started_at"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
}

func (j *ImportJob) Snapshot() ImportJobSnapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()
	results := make([]VINImportResult, len(j.Results))
	copy(results, j.Results)
	return ImportJobSnapshot{
		ID:            j.ID,
		CreatedBy:     j.CreatedBy,
		CreatedByName: j.CreatedByName,
		Status:        j.Status,
		Total:         j.Total,
		Processed:     j.Processed,
		Succeeded:     j.Succeeded,
		Failed:        j.Failed,
		Skipped:       j.Skipped,
		Invalid:       j.Invalid,
		Results:       results,
		StartedAt:     j.StartedAt,
		CompletedAt:   j.CompletedAt,
	}
}

var (
	importJobs   = make(map[string]*ImportJob)
	importJobsMu sync.RWMutex
)

func init() {
	// Prune completed jobs older than 2 hours every 30 minutes.
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			cutoff := time.Now().Add(-2 * time.Hour)
			importJobsMu.Lock()
			for id, job := range importJobs {
				if job.Status != "running" && job.StartedAt.Before(cutoff) {
					delete(importJobs, id)
				}
			}
			importJobsMu.Unlock()
		}
	}()
}

// --- Handler ---

type ImportHandler struct {
	DB *gorm.DB
}

type startImportRequest struct {
	VINs         []string `json:"vins"`
	Concurrency  int      `json:"concurrency"`
	SkipExisting bool     `json:"skip_existing"`
}

// POST /api/import
func (h *ImportHandler) StartImport(c *gin.Context) {
	var req startImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.VINs) == 0 {
		helpers.Fail(c, http.StatusBadRequest, "no VINs provided")
		return
	}
	if len(req.VINs) > 1000 {
		helpers.Fail(c, http.StatusBadRequest, "maximum 1000 VINs per import")
		return
	}

	concurrency := req.Concurrency
	if concurrency < 1 || concurrency > 10 {
		concurrency = 3
	}

	user := auth.CurrentUser(c)
	ctx, cancel := context.WithCancel(context.Background())

	job := &ImportJob{
		ID:            uuid.New().String(),
		CreatedBy:     user.ID,
		CreatedByName: user.Username,
		Status:        "running",
		Total:         len(req.VINs),
		Results:       make([]VINImportResult, 0, len(req.VINs)),
		StartedAt:     time.Now(),
		cancel:        cancel,
	}

	importJobsMu.Lock()
	importJobs[job.ID] = job
	importJobsMu.Unlock()

	go h.runImport(ctx, job, req.VINs, concurrency, req.SkipExisting)

	helpers.OK(c, gin.H{"job_id": job.ID})
}

// GET /api/import/:job_id
func (h *ImportHandler) GetImportJob(c *gin.Context) {
	jobID := c.Param("job_id")
	importJobsMu.RLock()
	job, exists := importJobs[jobID]
	importJobsMu.RUnlock()

	if !exists {
		helpers.Fail(c, http.StatusNotFound, "job not found")
		return
	}

	helpers.OK(c, job.Snapshot())
}

// DELETE /api/import/:job_id
func (h *ImportHandler) CancelImport(c *gin.Context) {
	jobID := c.Param("job_id")
	importJobsMu.RLock()
	job, exists := importJobs[jobID]
	importJobsMu.RUnlock()

	if !exists {
		helpers.Fail(c, http.StatusNotFound, "job not found")
		return
	}

	job.mu.Lock()
	if job.Status == "running" {
		job.cancel()
		job.Status = "cancelled"
	}
	job.mu.Unlock()

	helpers.OK(c, gin.H{"message": "cancellation requested"})
}

// --- Background runner ---

func (h *ImportHandler) runImport(ctx context.Context, job *ImportJob, vins []string, concurrency int, skipExisting bool) {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, vin := range vins {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(vin string) {
			defer wg.Done()
			defer func() { <-sem }()

			result := h.processVIN(ctx, vin, skipExisting)

			job.mu.Lock()
			job.Results = append(job.Results, result)
			job.Processed++
			switch result.Status {
			case "success":
				job.Succeeded++
			case "failed":
				job.Failed++
			case "skipped":
				job.Skipped++
			case "invalid":
				job.Invalid++
			}
			job.mu.Unlock()
		}(vin)
	}

	wg.Wait()

	now := time.Now()
	job.mu.Lock()
	if job.Status == "running" {
		job.Status = "completed"
		job.CompletedAt = &now
	}
	job.mu.Unlock()
}

func (h *ImportHandler) processVIN(ctx context.Context, rawVIN string, skipExisting bool) VINImportResult {
	start := time.Now()
	vin := strings.TrimSpace(strings.ToUpper(rawVIN))

	result := VINImportResult{
		VIN:  vin,
		IsGM: services.IsGMBrandVIN(vin),
	}

	// Validate: only accept full 17-char VINs for decode
	if len(vin) != 17 || !helpers.VinValidator(vin) {
		result.Status = "invalid"
		result.Error = "invalid VIN format"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	buildKey := helpers.ExtractBuildKey(vin)

	// Already in DB?
	var existing models.Vehicle
	if h.DB.Where("build_key = ?", buildKey).First(&existing).Error == nil {
		if skipExisting {
			// Even when skipping the re-decode, backfill GM build-key-stable
			// fields for GM cars that haven't been enriched yet.
			if !existing.GMChecked && services.IsGMBrandVIN(vin) {
				if err := services.EnrichExistingWithGM(h.DB, vin, &existing); err != nil {
					log.Printf("import GM backfill failed for %s (non-fatal): %v", vin, err)
				}
			}
			result.Status = "skipped"
			result.Make = existing.Make
			result.Model = existing.Model
			result.Year = existing.Year
			result.DurationMs = time.Since(start).Milliseconds()
			return result
		}
	}

	// Context cancelled before we start the API calls?
	if ctx.Err() != nil {
		result.Status = "failed"
		result.Error = "import cancelled"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	var vehicle models.Vehicle
	if err := services.DecodeVINAndSave(h.DB, vin, &vehicle); err != nil {
		result.Status = "failed"
		result.Error = err.Error()
	} else {
		result.Status = "success"
		result.Make = vehicle.Make
		result.Model = vehicle.Model
		result.Year = vehicle.Year
		if err := services.RecordKnownVIN(h.DB, vehicle.BuildKey, vin); err != nil {
			log.Printf("known vin record failed for import %s (non-fatal): %v", vin, err)
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result
}
