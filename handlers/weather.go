package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

const (
	weatherServiceUnavailableMsg = "weather service unavailable"
	weatherDataIncompleteMsg     = "weather data incomplete"
)

var (
	// Default timeout can be overridden via env: DMI_HTTP_TIMEOUT (e.g. "20s", "5s", "1m")
	weatherTimeout = parseDurationEnv("DMI_HTTP_TIMEOUT", 20*time.Second)
	weatherClient  = &http.Client{Timeout: weatherTimeout}
)

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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%s (status %d): %s", weatherServiceUnavailableMsg, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data EDRFeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}

	return &data, nil
}

// parseDurationEnv matches the naming convention used in cmd/server/main.go.
// Kept local (no shared util refactor) to minimize risk/merge complexity.
func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
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
			"Error":    weatherServiceUnavailableMsg, // <-- sanitize (don’t leak err.Error())
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

func APIWeatherHandler(w http.ResponseWriter, r *http.Request) {
	data, err := GetCopenhagenForecast(r.Context())
	if err != nil {
		log.Println("weather API fetch error:", err)
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: weatherServiceUnavailableMsg})
		return
	}

	if data == nil {
		log.Println("weather API: empty response body")
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: weatherServiceUnavailableMsg})
		return
	}

	if len(data.Features) == 0 {
		log.Println("weather API: empty feature list")
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: weatherServiceUnavailableMsg})
		return
	}

	first := data.Features[0]
	if len(first.Geometry.Coordinates) < 2 {
		log.Println("weather API: missing coordinates in response")
		writeJSON(w, http.StatusServiceUnavailable, APIErrorResponse{Error: weatherDataIncompleteMsg})
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
