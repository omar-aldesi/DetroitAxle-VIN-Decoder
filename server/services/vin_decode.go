package services

import (
	"errors"
	"fmt"
	"log"
	"main/models"
	"os"
	"strings"
	"sync"

	"gorm.io/gorm"
)

// DecodeVINAndSave fetches auto.dev + NHTSA (and, for GM-brand VINs, GM Parts
// Giant) concurrently, merges them onto vehicle, then upserts into the database.
//
// GM contributes ONLY VDS-encoded, build-key-stable fields (Series + engine
// details) via ApplyStableFields. GM's per-VIN data — the RPO `specification`
// list (brake code, options, …) AND transmission, which is not VDS-encoded —
// is never persisted here, since it can differ between units sharing a build
// key. That data is served live by GET /api/gm/decode/:vin.
func DecodeVINAndSave(db *gorm.DB, vin string, vehicle *models.Vehicle) error {
	vin = strings.TrimSpace(strings.ToUpper(vin))

	rawToken := os.Getenv("AUTO_DEV_TOKEN")
	apiToken := strings.Trim(strings.TrimSpace(rawToken), `"`)
	if apiToken == "" {
		return fmt.Errorf("AUTO_DEV_TOKEN environment variable is not set")
	}
	log.Printf("[VIN decode] vin=%s token_prefix=%s", vin, maskToken(apiToken))

	var wg sync.WaitGroup
	wg.Add(2)

	autoCh := make(chan autoDevResult, 1)
	nhtsaCh := make(chan nhtsaFetchResult, 1)
	gmCh := make(chan gmFetchResult, 1)

	go func() {
		defer wg.Done()
		resp, err := fetchAutoDev(vin, apiToken)
		autoCh <- autoDevResult{resp, err}
	}()

	go func() {
		defer wg.Done()
		resp, err := fetchNHTSA(vin)
		nhtsaCh <- nhtsaFetchResult{resp, err}
	}()

	// GM enrichment only for GM-brand VINs (others have no GM data anyway).
	if IsGMBrandVIN(vin) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attrs, err := FetchGMAttributes(vin)
			gmCh <- gmFetchResult{attrs, err}
		}()
	} else {
		gmCh <- gmFetchResult{} // buffered — won't block
	}

	wg.Wait()
	close(autoCh)
	close(nhtsaCh)
	close(gmCh)

	autoRes := <-autoCh
	nhtsaRes := <-nhtsaCh
	gmRes := <-gmCh

	if autoRes.err != nil {
		return fmt.Errorf("auto.dev request failed: %w", autoRes.err)
	}
	if !autoRes.resp.VINValid {
		return fmt.Errorf("invalid VIN: %s", vin)
	}
	if nhtsaRes.err != nil {
		log.Printf("[VIN decode] NHTSA fetch failed (non-fatal): %v", nhtsaRes.err)
	}

	if err := mapToVehicle(vin, autoRes.resp, nhtsaRes.resp, vehicle); err != nil {
		return fmt.Errorf("mapping failed: %w", err)
	}

	// GM: persist only build-key-stable fields. Non-fatal on failure — a GM
	// outage (or a VIN GM doesn't recognize) must never block the decode.
	if IsGMBrandVIN(vin) {
		switch {
		case gmRes.err == nil && gmRes.attrs != nil:
			gmRes.attrs.ApplyStableFields(vehicle)
			vehicle.GMChecked = true
		case errors.Is(gmRes.err, ErrGMNoData):
			vehicle.GMChecked = true // GM definitively has nothing — don't retry
		default:
			// Transient error — leave GMChecked false so a later access retries.
			log.Printf("[VIN decode] GM enrichment failed (non-fatal): %v", gmRes.err)
		}
	}

	// Upsert.
	// Assign(*vehicle) passes a struct VALUE snapshot taken right now, before
	// FirstOrCreate scans the DB row back into the pointer — preventing the
	// enriched values from being silently overwritten by the DB read.
	if err := db.Where(models.Vehicle{BuildKey: vehicle.BuildKey}).
		Assign(*vehicle).
		FirstOrCreate(vehicle).Error; err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}
