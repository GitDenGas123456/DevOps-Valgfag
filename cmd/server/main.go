// @title WhoKnows API
// @version 0.1.0
// @description API specification for the WhoKnows web application
// @BasePath /
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"

	httpSwagger "github.com/swaggo/http-swagger" // ✅ Swagger HTTP handler
	_ "devops-valgfag/docs"                    // ✅ Replace with your module name (e.g., "github.com/you/whoknows/docs")

)

// =====================
// Globals
// =====================

var (
	// TODO: move these into a small app/context struct later
	db           *sql.DB
	tmpl         *template.Template
	sessionStore *sessions.CookieStore
)

// =====================
// Models
// =====================

// User represents an application user
// @Description Application user with login credentials
type User struct {
	ID       int `json:"id" example:"1"`
	Username string `json:"username" example:"alice"`
	Email    string `json:"email" example:"alice@example.com"`
	Password string `json:"password,omitempty"` // bcrypt hash 
}

type SearchResult struct {
	ID       int `json:"id" example:"1"`
	Language string `json:"language" example:"en"`
	Content  string `json:"content" example:"Sample content"`
}

// SearchResponse represents an API search response
type SearchResponse struct {
	SearchResults []SearchResult `json:"search_results"`
}

// AuthResponse represents a generic auth API response
type AuthResponse struct {
	StatusCode int    `json:"statusCode" example:"200"`
	Message    string `json:"message" example:"Login successful"`
}

// =====================
// API response structure weather
// =====================

// Weather response form for weather api
// @Description Api data structures
type EDRFeatureCollection struct {
	Type     string        `json:"type"`
	Features []EDRFeature  `json:"features"`
}

type EDRFeature struct {
	Type       string         `json:"type"`
	Geometry   EDRGeometry    `json:"geometry"`
	Properties EDRProperties  `json:"properties"`
}

type EDRGeometry struct {
	Type        string      `json:"type"`
	Coordinates []float64   `json:"coordinates"`
}

type EDRProperties struct {
	Temperature float64 `json:"temperature-2m"`
	WindSpeed   float64 `json:"wind-speed-10m"`
	WindDir     float64 `json:"wind-dir-10m"`
	Step        string  `json:"step"`
}

//Weather data fetch from external api
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


//render html template 
func renderTemplateWeather(w http.ResponseWriter, tmpl string, data any) {
	t, err := template.ParseFiles(fmt.Sprintf("templates/%s.html", tmpl))
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		log.Println("Template error:", err)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Println("Template execution error:", err)
	}
}

// =====================
// Main
// =====================

func main() {
	port := getenv("PORT", "8080")
	dbPath := getenv("DATABASE_PATH", "internal/db/whoknows.db")
	sessionKey := getenv("SESSION_KEY", "development key") // dev fallback

	// DB
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Templates with FuncMap so {{ now }} and {{ now.Year }} (or {{ year }}) work
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}
	tmpl = template.Must(template.New("").Funcs(funcs).ParseGlob("./templates/*.html"))

	// Sessions
	sessionStore = sessions.NewCookieStore([]byte(sessionKey))

	// Router
	r := mux.NewRouter()

	// Static
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Page routes
	r.HandleFunc("/", SearchPageHandler).Methods("GET")
	r.HandleFunc("/about", AboutPageHandler).Methods("GET")
	r.HandleFunc("/login", LoginPageHandler).Methods("GET")
	r.HandleFunc("/register", RegisterPageHandler).Methods("GET")
	r.HandleFunc("/weather", WeatherPageHandler).Methods("GET")

	// API routes
	r.HandleFunc("/api/login", APILoginHandler).Methods("POST")
	r.HandleFunc("/api/register", APIRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", APILogoutHandler).Methods("POST", "GET")
	r.HandleFunc("/api/search", APISearchHandler).Methods("POST")
	

	//Swagger documentation route
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	fmt.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// =====================
// Page Handlers
// =====================

// SearchPageHandler godoc
// @Summary Serve Root Page
// @Description Returns the search page
// @Tags Pages
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router / [get]

func SearchPageHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, language, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Language, &it.Content); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	// Execute the WRAPPER template "search"
	renderTemplate(w, "search", map[string]any{
		"Title":   "",
		"Query":   q,
		"Results": results,
	})
}

// AboutPageHandler godoc
// @Summary Serve About Page
// @Tags Pages
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router /about [get]
func AboutPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about", map[string]any{
		"Title": "About",
	})
}

// LoginPageHandler godoc
// @Summary Serve Login Page
// @Tags Pages
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router /login [get]
func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login", map[string]any{
		"Title": "Sign In",
	})
}

// RegisterPageHandler godoc
// @Summary Serve Register Page
// @Tags Pages
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router /register [get]
func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register", map[string]any{
		"Title": "Sign Up",
	})
}

// WeatherPageHandler godoc
// @Summary Serve Weather Page
// @Tags Pages
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router /weather [get]

//Weather web route handler
func WeatherPageHandler(w http.ResponseWriter, r *http.Request) {
	data, err := GetCopenhagenForecast()
	if err != nil {
		http.Error(w, "Failed to fetch forecast data", http.StatusInternalServerError)
		log.Println("Forecast fetch error:", err)
		return
	}

	// Extract first feature
	var forecast *EDRFeature
	if len(data.Features) > 0 {
		forecast = &data.Features[0]
	}

	renderTemplateWeather(w, "weather", map[string]any{
		"Title":    "Copenhagen Forecast",
		"Forecast": forecast,
	})
}


// =====================
// API Handlers
// =====================

// APISearchHandler godoc
// @Summary Search
// @Description Search for content by query and language
// @Tags API
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param language query string false "Language code (e.g., 'en')"
// @Success 200 {object} SearchResponse
// @Router /api/search [post]
func APISearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, language, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Language, &it.Content); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"search_results": results})
}

// APILoginHandler godoc
// @Summary Login
// @Description Authenticate a user with username and password
// @Tags API
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} AuthResponse
// @Router /api/login [post]
func APILoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form"})
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	u := User{}
	err := db.QueryRow(
		`SELECT id, username, email, password FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password)
	if err != nil {
		renderTemplate(w, "login", map[string]any{"Title": "Sign In", "error": "Invalid username"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		renderTemplate(w, "login", map[string]any{"Title": "Sign In", "error": "Invalid password", "username": username})
		return
	}

	sess, _ := sessionStore.Get(r, "session")
	sess.Values["user_id"] = u.ID
	_ = sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

// APIRegisterHandler godoc
// @Summary Register
// @Description Register a new user
// @Tags API
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "Username"
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Param password2 formData string false "Password Confirmation"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} AuthResponse
// @Router /api/register [post]
func APIRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form"})
		return
	}
	username := r.FormValue("username")
	email := r.FormValue("email")
	pw1 := r.FormValue("password")
	pw2 := r.FormValue("password2")

	if username == "" || email == "" || pw1 == "" {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "All fields required"})
		return
	}
	if pw1 != pw2 {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Password do not match"})
		return
	}

	var exists int
	_ = db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&exists)
	if exists > 0 {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed, Username already in use"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(pw1), bcrypt.DefaultCost)
	_, err := db.Exec(
		`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`,
		username, email, string(hash),
	)
	if err != nil {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed"})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// APILogoutHandler godoc
// @Summary Logout
// @Description Log out the current user
// @Tags API
// @Produce json
// @Success 200 {object} AuthResponse
// @Router /api/logout [get]
func APILogoutHandler(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionStore.Get(r, "session")
	delete(sess.Values, "user_id")
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// =====================
// Helpers
// =====================

func renderTemplate(w http.ResponseWriter, page string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ""
	}
	// Execute the page wrapper ("search", "about", "login", "register")
	if err := tmpl.ExecuteTemplate(w, page, data); err != nil {
		http.Error(w, "template exec error: "+err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "json encode error: "+err.Error(), http.StatusInternalServerError)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}