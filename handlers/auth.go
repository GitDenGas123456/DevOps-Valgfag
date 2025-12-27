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

// APILoginHandler authenticates a user and starts a cookie-based session.
//
// Behavior:
// - Expects form fields: username, password (application/x-www-form-urlencoded).
// - On success: stores the authenticated user_id in the "session" cookie and redirects to "/" (302).
// - On failure (bad form / bad credentials): renders the login page with an error and returns 200.
// - Avoids username enumeration by not distinguishing between "unknown user" and "wrong password".
//
// APILoginHandler godoc
// @Summary      User login
// @Description  Authenticate a user and start a session. On failure, renders the login page (HTTP 200) with an error message.
// @Tags         Auth
// @Accept       application/x-www-form-urlencoded
// @Produce      html
// @Param        username  formData  string  true   "Username"
// @Param        password  formData  string  true   "Password"
// @Success      302  {string}  string  "Redirect to home page"
// @Success      200  {string}  string  "Rendered login form with errors"
// @Router       /api/login [post]
func APILoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderTemplate(w, r, "login", map[string]any{
			"Title": loginTitle,
			"Error": "Bad request",
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
			"Error":    "Invalid username or password",
			"Username": username,
		})
		return
	}

	// Create a session for the authenticated user
	sess, err := sessionStore.Get(r, "session")
	if err != nil {
		log.Printf("sessionStore.Get error (login): %v", err)
		renderTemplate(w, r, "login", map[string]any{
			"Title":    loginTitle,
			"Error":    "Internal server error",
			"Username": username,
		})
		return
	}

	sess.Values["user_id"] = u.ID
	if err := sess.Save(r, w); err != nil {
		log.Printf("sess.Save error (login): %v", err)
		renderTemplate(w, r, "login", map[string]any{
			"Title":    loginTitle,
			"Error":    "Internal server error",
			"Username": username,
		})
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// APIRegisterHandler creates a new user account.
//
// Behavior:
// - Expects form fields: username, email, password, password2 (application/x-www-form-urlencoded).
// - On success: inserts the user (bcrypt password hash) and redirects to "/login" (302).
// - On validation / DB errors: renders the register page with an error and returns 200.
//
// APIRegisterHandler godoc
// @Summary      Register user
// @Description  Create a new user account. On validation errors, renders the register page (HTTP 200) with an error message.
// @Tags         Auth
// @Accept       application/x-www-form-urlencoded
// @Produce      html
// @Param        username   formData  string  true   "Username"
// @Param        email      formData  string  true   "Email address"
// @Param        password   formData  string  true   "Password"
// @Param        password2  formData  string  true   "Password confirmation"
// @Success      302  {string}  string  "Redirect to login page"
// @Success      200  {string}  string  "Rendered register form with errors"
// @Router       /api/register [post]
func APIRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"Error": "Bad request",
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
			"Error": "All fields required",
		})
		return
	}

	// Password confirmation check
	if pw1 != pw2 {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"Error": "Passwords do not match",
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
			"Error": "Database error",
		})
		return
	}

	if exists > 0 {
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"Error": "Username already in use",
		})
		return
	}

	// Hash the password using bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(pw1), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt.GenerateFromPassword error: %v", err)
		renderTemplate(w, r, "register", map[string]any{
			"Title": registerTitle,
			"Error": "Internal error, please try again",
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
			"Error": "Registration failed",
		})
		return
	}

	// Redirect to login page after successful registration
	http.Redirect(w, r, "/login", http.StatusFound)
}

// APILogoutHandler clears the current user's session and redirects home.
//
// Notes:
// - If the user is not logged in, sessionStore.Get typically returns an empty session;
//   the handler still redirects home after attempting to clear "user_id".
// - Intended to be POST-only to avoid side effects on GET.
//
// APILogoutHandler godoc
// @Summary      Logout user
// @Description  Clear the user session and redirect home.
// @Tags         Auth
// @Produce      html
// @Security     sessionAuth
// @Success      302  {string}  string  "Redirect to home page"
// @Router       /api/logout [post]
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
