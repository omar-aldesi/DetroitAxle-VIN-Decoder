package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const gmDecodeURL = "https://www.gmpartsgiant.com/api/vehicle/mul/decode-vin-attributes"

// ErrGMNoData means GM Parts Giant responded successfully but has no build data
// for this VIN (e.g. an upfitter-built incomplete vehicle). Callers should treat
// this as a clean "not found", not a server error.
var ErrGMNoData = errors.New("no GM build data for this VIN")

var gmClient = &http.Client{
	Timeout: 15 * time.Second,
}

// --- Response types ---

type gmAPIResponse struct {
	Code int       `json:"code"`
	Data gmAPIData `json:"data"`
}

type gmAPIData struct {
	VinInfos []gmVinInfo `json:"vinInfos"`
}

type gmVinInfo struct {
	VehicleInfo        string       `json:"vehicleInfo"`
	RequiredInfo       string       `json:"requiredInfo"`
	OptionalInfo       string       `json:"optionalInfo"`
	RedirectURL        string       `json:"redirectUrl"`
	VehicleInformation []gmNameDesc `json:"vehicleInformation"`
	MajorAttribute     []gmNameDesc `json:"majorAttribute"`
	Specification      []gmNameDesc `json:"specification"`
}

type gmNameDesc struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// GMVINAttributes holds the parsed response from GM Parts Giant.
type GMVINAttributes struct {
	VinInfos []gmVinInfo
}

// --- HTTP fetch ---

// FetchGMAttributes calls the GM Parts Giant VIN decode API.
// Retries once after 2 s on 429 or 503. Returns ErrGMNoData when the VIN
// decodes successfully but carries no build data.
func FetchGMAttributes(vin string) (*GMVINAttributes, error) {
	const maxAttempts = 2
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attrs, err := doGMFetch(vin)
		if err == nil {
			return attrs, nil
		}
		// Don't retry a definitive "no data" result.
		if errors.Is(err, ErrGMNoData) {
			return nil, err
		}
		lastErr = err
		if attempt < maxAttempts {
			log.Printf("[gm-parts-giant] attempt %d failed: %v — retrying after 2s", attempt, err)
			time.Sleep(2 * time.Second)
		}
	}
	return nil, lastErr
}

func doGMFetch(vin string) (*GMVINAttributes, error) {
	payload, err := json.Marshal(map[string]string{"vin": vin})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, gmDecodeURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	guid := gmUUID()
	now := time.Now()
	logKey := fmt.Sprintf("%d%02d.%014d", now.UnixMilli(), now.UnixNano()%100, now.UnixNano()%int64(1e14))
	vinURL := "https://www.gmpartsgiant.com/vin-decoder.html?vin=" + vin

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("If-Modified-Since", "0")
	req.Header.Set("Origin", "https://www.gmpartsgiant.com")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", vinURL)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("accessToken", "")
	req.Header.Set("currentHost", "www.gmpartsgiant.com")
	req.Header.Set("currentUrl", vinURL)
	req.Header.Set("guid", guid)
	req.Header.Set("logkey", logKey)
	req.Header.Set("site", "GPG")

	resp, err := gmClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[gm-parts-giant] vin=%s status=%d body=%s", vin, resp.StatusCode, truncate(string(body), 400))

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("rate limited (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(body), 120))
	}

	var apiResp gmAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	// code 200 with zero vinInfos = decoded fine, but no build data for this VIN.
	if len(apiResp.Data.VinInfos) == 0 {
		return nil, ErrGMNoData
	}

	return &GMVINAttributes{VinInfos: apiResp.Data.VinInfos}, nil
}
