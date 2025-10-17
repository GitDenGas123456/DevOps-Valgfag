package handlers

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int
	Username string
	Email    string
	Password string // bcrypt hash
}

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
	if _, err := db.Exec(
		`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`,
		username, email, string(hash),
	); err != nil {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed"})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func APILogoutHandler(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionStore.Get(r, "session")
	delete(sess.Values, "user_id")
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}
