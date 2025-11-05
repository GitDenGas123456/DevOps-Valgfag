package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// ==========
// Models
// ==========

// Weather response structures
type EDRFeatureCollection struct {
	Type     string       `json:"type"`
	Features []EDRFeature `json:"features"`
}

type EDRFeature struct {
	Type       string        `json:"type"`
	Geometry   EDRGeometry   `json:"geometry"`
	Properties EDRProperties `json:"properties"`
}

type EDRGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type EDRProperties struct {
	Temperature float64 `json:"temperature-2m"`
	WindSpeed   float64 `json:"wind-speed-10m"`
	WindDir     float64 `json:"wind-dir-10m"`
	Step        string  `json:"step"`
}

// ==========
// Weather API
// ==========

func GetCopenhagenForecast() (*EDRFeatureCollection, error) {
	apiKey := os.Getenv("DMI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing DMI_API_KEY environment variable")
	}

	url := fmt.Sprintf(
		"https://dmigw.govcloud.dk/v1/forecastedr/collections/harmonie_dini_sf/position"+
			"?coords=POINT(12.561%%2055.715)&crs=crs84"+
			"&parameter-name=temperature-2m,wind-speed-10m,wind-dir-10m"+
			"&f=GeoJSON&api-key=%s", apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %s", resp.Status)
	}

	var data EDRFeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}

	return &data, nil
}

// ==========
// Page handler
// ==========

func WeatherPageHandler(w http.ResponseWriter, r *http.Request) {
	data, err := GetCopenhagenForecast()

	var forecast *EDRFeature
	errorMessage := ""
	if err != nil {
		log.Println("Forecast fetch error:", err)
		errorMessage = err.Error()
	} else if len(data.Features) > 0 {
		forecast = &data.Features[0]
	}

	renderTemplate(w, "weather", map[string]any{
		"Title":    "Copenhagen Forecast",
		"Forecast": forecast,
		"Error":    errorMessage,
	})
}
