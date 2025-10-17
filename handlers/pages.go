package handlers

import "net/http"

func AboutPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about", map[string]any{"Title": "About"})
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login", map[string]any{"Title": "Sign In"})
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register", map[string]any{"Title": "Sign Up"})
}
