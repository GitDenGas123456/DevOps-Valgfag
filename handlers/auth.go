package handlers

import (
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

const (
	loginTitle    = "Sign In"
	registerTitle = "Sign Up"
)

// User represents the user object returned from the database.
// The Password field contains a bcrypt hash (never a plaintext password).
type User struct {
	ID       int
	Username string
	Email    string
	Password string
}

// APILoginHandler handles user login requests.
// It validates the incoming form, checks the database for a matching user,
// and verifies the password using bcrypt.
//
// APILoginHandler godoc
// @Summary      User login
// @Description  Authenticate a user and start a session.
// @Tags         Auth
// @Accept       application/x-www-form-urlencoded
// @Produce      html
// @Param        username  formData  string  true   "Username"
// @Param        password  formData  string  true   "Password"
// @Success      302  {string}  string  "Redirect to home page"
// @Failure      200  {string}  string  "Rendered login form with errors"
// @Router       /api/login [post]
func APILoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderTemplate(w, r, "login", map[string]any{
			"Title": loginTitle,
			"error": "Bad request",
		})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	u := User{}

	// Query PostgreSQL using parameter placeholder $1
	err := db.QueryRow(
		`SELECT id, username, email, password FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password)

	// Avoid username enumeration by not distinguishing between "bad user" and "bad password"
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		renderTemplate(w, r, "login", map[string]any{
			"Title":    loginTitle,
			"error":    "Invalid username or password",
			"username": username,
		})
		return
	}

	// Create a session for the authenticated user
	sess, err := sessionStore.Get(r, "session")
	if err != nil {
		log.Printf("sessionStore.Get error (login): %v", err)
		renderTemplate(w, r, "login", map[string]any{
			"Title":    loginTitle,
			"error":    "Internal server error",
			"username": username,
		})
		return
	}

	sess.Values["user_id"] = u.ID
	if err := sess.Save(r, w); err != nil {
		log.Printf("sess.Save error (login): %v", err)
		renderTemplate(w, r, "login", map[string]any{
			"Title":    loginTitle,
			"error":    "Internal server error",
			"username": username,
		})
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// APIRegisterHandler handles new user registration.
// It validates input, checks for existing usernames, hashes the password,
// and inserts the user into PostgreSQL.
//
// APIRegisterHandler godoc
// @Summary      Register user
// @Description  Create a new user account.
// @Tags         Auth
// @Accept       application/x-www-form-urlencoded
// @Produce      html
// @Param        username   formData  string  true   "Username"
// @Param        email      formData  string  true   "Email address"
// @Param        password   formData  string  true   "Password"
// @Param        password2  formData  string  true   "Password confirmation"
// @Success      302  {string}  string  "Redirect to login page"
// @Failure      200  {string}  string  "Rendered register form with errors"
// @Router       /api/register [post]
func APIRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "Bad request",
		})
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	pw1 := r.FormValue("password")
	pw2 := r.FormValue("password2")

	// Basic validation for required fields
	if username == "" || email == "" || pw1 == "" {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "All fields required",
		})
		return
	}

	// Password confirmation check
	if pw1 != pw2 {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "Passwords do not match",
		})
		return
	}

	// Check if username already exists
	var exists int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE username = $1`,
		username,
	).Scan(&exists)
	if err != nil {
		log.Printf("register exists query error: %v", err)
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "Database error",
		})
		return
	}

	if exists > 0 {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "Username already in use",
		})
		return
	}

	// Hash the password using bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(pw1), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt.GenerateFromPassword error: %v", err)
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "Internal error, please try again",
		})
		return
	}

	// Insert new user into PostgreSQL
	_, err = db.Exec(
		`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`,
		username, email, string(hash),
	)
	if err != nil {
		log.Printf("register insert error: %v", err)
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"error": "Registration failed",
		})
		return
	}

	// Redirect to login page after successful registration
	http.Redirect(w, r, "/login", http.StatusFound)
}

// APILogoutHandler logs out the user by removing the session value.
//
// APILogoutHandler godoc
// @Summary      Logout user
// @Description  Clear the user session and redirect home.
// @Tags         Auth
// @Produce      html
// @Security     sessionAuth
// @Success      302  {string}  string  "Redirect to home page"
// @Router       /api/logout [get]
func APILogoutHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := sessionStore.Get(r, "session")
	if err != nil {
		log.Printf("sessionStore.Get error (logout): %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	delete(sess.Values, "user_id")
	if err := sess.Save(r, w); err != nil {
		log.Printf("sess.Save error (logout): %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
