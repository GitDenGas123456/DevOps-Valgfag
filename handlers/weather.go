package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ==========
// Models (DMI EDR GeoJSON subset)
// ==========

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
	Coordinates []float64 `json:"coordinates"` // [lon, lat]
}

type EDRProperties struct {
	Temperature float64 `json:"temperature-2m"`
	WindSpeed   float64 `json:"wind-speed-10m"`
	WindDir     float64 `json:"wind-dir-10m"`
	Step        string  `json:"step"`
}

// API response structures
type WeatherAPIResponse struct {
	Location WeatherLocation `json:"location"`
	Forecast WeatherForecast `json:"forecast"`
}

type WeatherLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type WeatherForecast struct {
	Temperature   float64 `json:"temperature"`
	WindSpeed     float64 `json:"wind_speed"`
	WindDirection float64 `json:"wind_direction"`
	Step          string  `json:"step"`
}

type APIErrorResponse struct {
	Error string `json:"error"`
}

var weatherClient = &http.Client{Timeout: 5 * time.Second}

// ==========
// Weather fetcher
// ==========

func GetCopenhagenForecast(ctx context.Context) (*EDRFeatureCollection, error) {
	apiKey := os.Getenv("DMI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing DMI_API_KEY environment variable")
	}

	baseURL := strings.TrimSuffix(os.Getenv("DMI_API_URL"), "/")
	if baseURL == "" {
		baseURL = "https://dmigw.govcloud.dk"
	}

	u := fmt.Sprintf(
		"%s/v1/forecastedr/collections/harmonie_dini_sf/position"+
			"?coords=POINT(12.561%%2055.715)&crs=crs84"+
			"&parameter-name=temperature-2m,wind-speed-10m,wind-dir-10m"+
			"&f=GeoJSON&api-key=%s",
		baseURL,
		apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := weatherClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("failed to close weather response body: %v", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather service unavailable (status %d)", resp.StatusCode)
	}

	var data EDRFeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}

	return &data, nil
}

// ==========
// Page handler: /weather
// ==========

func WeatherPageHandler(w http.ResponseWriter, r *http.Request) {
	data, err := GetCopenhagenForecast(r.Context())
	if err != nil {
		log.Println("Forecast fetch error:", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		renderTemplate(w, r, "weather", map[string]any{
			"Title":    "Copenhagen Forecast",
			"Forecast": nil,
			"Error":    err.Error(),
		})
		return
	}

	var forecast *EDRFeature
	if data != nil && len(data.Features) > 0 {
		forecast = &data.Features[0]
	}

	renderTemplate(w, r, "weather", map[string]any{
		"Title":    "Copenhagen Forecast",
		"Forecast": forecast,
		"Error":    "",
	})
}

// ==========
// API handler: /api/weather
// ==========

// APIWeatherHandler godoc
// @Summary      Get weather forecast
// @Description  Returns the current Copenhagen forecast used by the /weather page.
// @Tags         API
// @Produce      json
// @Success      200  {object}  WeatherAPIResponse  "Forecast retrieved"
// @Failure      503  {object}  APIErrorResponse    "Weather service unavailable"
// @Router       /api/weather [get]
func APIWeatherHandler(w http.ResponseWriter, r *http.Request) {
	data, err := GetCopenhagenForecast(r.Context())
	if err != nil {
		log.Println("weather API fetch error:", err)
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: "weather service unavailable"})
		return
	}

	if data == nil || len(data.Features) == 0 {
		log.Println("weather API: no data/features available")
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: "weather service unavailable"})
		return
	}

	first := data.Features[0]
	if len(first.Geometry.Coordinates) < 2 {
		log.Println("weather API: missing coordinates in response")
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: "weather data incomplete"})
		return
	}

	writeJSON(w, http.StatusOK, WeatherAPIResponse{
		Location: WeatherLocation{
			Latitude:  first.Geometry.Coordinates[1],
			Longitude: first.Geometry.Coordinates[0],
		},
		Forecast: WeatherForecast{
			Temperature:   first.Properties.Temperature,
			WindSpeed:     first.Properties.WindSpeed,
			WindDirection: first.Properties.WindDir,
			Step:          first.Properties.Step,
		},
	})
}
