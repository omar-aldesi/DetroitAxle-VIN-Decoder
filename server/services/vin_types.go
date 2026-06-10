package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type autoDevResponse struct {
	VIN      string `json:"vin"`
	VINValid bool   `json:"vinValid"`
	Origin   string `json:"origin"` // country of manufacture
	Make     string `json:"make"`
	Model    string `json:"model"`
	Trim     string `json:"trim"`
	Style    string `json:"style"`  // e.g. "4x4 4dr Crew Cab 5.8 ft. SB"
	Body     string `json:"body"`   // e.g. "Truck", "Sedan"
	Engine   string `json:"engine"` // e.g. "5.3L V8 OHV 16V FFV"
	Drive    string `json:"drive"`
	Trans    string `json:"transmission"`
	Vehicle  struct {
		Year  int    `json:"year"`
		Make  string `json:"make"`
		Model string `json:"model"`
	} `json:"vehicle"`
}

type nhtsaResponse struct {
	Results []nhtsaResult `json:"Results"`
}

type nhtsaResult struct {
	// Engine / drivetrain
	EngineCylinders     string `json:"EngineCylinders"`
	DisplacementL       string `json:"DisplacementL"`
	EngineConfiguration string `json:"EngineConfiguration"` // "V", "Inline", etc.
	FuelTypePrimary     string `json:"FuelTypePrimary"`
	FuelTypeSecondary   string `json:"FuelTypeSecondary"`
	TransmissionStyle   string `json:"TransmissionStyle"`
	TransmissionSpeeds  string `json:"TransmissionSpeeds"`
	DriveType           string `json:"DriveType"`

	// Body
	BodyClass string `json:"BodyClass"` // "Pickup", "SUV", etc.
	Doors     string `json:"Doors"`

	// Identity — may fill gaps auto.dev misses
	ModelYear string `json:"ModelYear"`
	Make      string `json:"Make"`
	Model     string `json:"Model"`
	Trim      string `json:"Trim"`
	Series    string `json:"Series"`

	// Safety
	ABS             string `json:"ABS"`             // "Standard", "Optional", "Not Available"
	BrakeSystemType string `json:"BrakeSystemType"` // "Hydraulic", etc.

	// Weight
	GVWR string `json:"GVWR"`

	// Assembly plant (used to infer Country when auto.dev's Origin is blank)
	PlantCity        string `json:"PlantCity"`
	PlantCountry     string `json:"PlantCountry"`     // e.g. "UNITED STATES (USA)"
	PlantCompanyName string `json:"PlantCompanyName"` // e.g. "GENERAL MOTORS"

	// Decode quality
	ErrorCode string `json:"ErrorCode"` // "0" = clean decode
	ErrorText string `json:"ErrorText"`
	Note      string `json:"Note"`
}

type autoDevResult struct {
	resp *autoDevResponse
	err  error
}

type nhtsaFetchResult struct {
	resp *nhtsaResult
	err  error
}

type gmFetchResult struct {
	attrs *GMVINAttributes
	err   error
}

func fetchAutoDev(vin, apiKey string) (*autoDevResponse, error) {
	url := fmt.Sprintf("https://api.auto.dev/vin/%s", vin)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[auto.dev] vin=%s status=%d body=%s", vin, resp.StatusCode, truncate(string(body), 400))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auto.dev returned status %d: %s", resp.StatusCode, truncate(string(body), 120))
	}

	var result autoDevResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse auto.dev response: %w", err)
	}

	return &result, nil
}

func fetchNHTSA(vin string) (*nhtsaResult, error) {
	url := fmt.Sprintf("https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValuesExtended/%s?format=json", vin)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[nhtsa] vin=%s status=%d body=%s", vin, resp.StatusCode, truncate(string(body), 400))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NHTSA returned status %d: %s", resp.StatusCode, truncate(string(body), 120))
	}

	var result nhtsaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse NHTSA response: %w", err)
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("NHTSA returned empty results for VIN %s", vin)
	}

	return &result.Results[0], nil
}
