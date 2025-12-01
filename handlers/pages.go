package handlers

// Imports
import "net/http"

func AboutPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "about", map[string]any{"Title": "About"})
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login", map[string]any{"Title": "Sign In"})
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "register", map[string]any{"Title": "Sign Up"})
}
