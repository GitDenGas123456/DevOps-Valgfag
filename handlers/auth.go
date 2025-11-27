package handlers

// Imports
import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// atrributes for user element
type User struct {
	ID       int
	Username string
	Email    string
	Password string // bcrypt hash
}

// Api handler for login
func APILoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form"})
		return
	}

	// set values for form
	username := r.FormValue("username")
	password := r.FormValue("password")

	// set user value
	u := User{}

	// Calling for values in database
	err := db.QueryRow(
		`SELECT id, username, email, password FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password)

	// Error catch if username not found in database
	if err != nil {
		renderTemplate(w, "login", map[string]any{"Title": "Sign In", "error": "Invalid username"})
		return
	}
	// Error catch if hashed password not found in database
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		renderTemplate(w, "login", map[string]any{"Title": "Sign In", "error": "Invalid password", "username": username})
		return
	}

	// Start session for logged in user
	sess, _ := sessionStore.Get(r, "session")
	sess.Values["user_id"] = u.ID
	_ = sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

// API for handling registration
func APIRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form"})
		return
	}

	// Atrributd used for user element
	username := r.FormValue("username")
	email := r.FormValue("email")
	pw1 := r.FormValue("password")
	pw2 := r.FormValue("password2")

	// Error catch for possible empty fields when trying to register
	if username == "" || email == "" || pw1 == "" {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "All fields required"})
		return
	}

	// Error catch for non matching passwords in registration
	if pw1 != pw2 {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Password do not match"})
		return
	}

	// Checking if user username already exists
	var exists int
	_ = db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&exists)
	if exists > 0 {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed, Username already in use"})
		return
	}

	// Generating hash for password for extra security
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw1), bcrypt.DefaultCost)

	//Inserts user into database table and error catch for failed registration
	if _, err := db.Exec(
		`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`,
		username, email, string(hash),
	); err != nil {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed"})
		return
	}

	// Directing to login page after succesful registration
	http.Redirect(w, r, "/login", http.StatusFound)
}

// API for log out handler 
func APILogoutHandler(w http.ResponseWriter, r *http.Request) {

	// Calls session id
	sess, _ := sessionStore.Get(r, "session")

	// Deletes session from user and saves 
	delete(sess.Values, "user_id")
	_ = sess.Save(r, w)

	// Redirects to main page after log out
	http.Redirect(w, r, "/", http.StatusFound)
}
