package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

type Result struct {
	Title       string
	Description string
	URL         string
}

type PageData struct {
	Query   string
	Results []Result
}

var tmpl *template.Template

func loadTemplates() {
	var err error
	tmpl, err = template.ParseFiles(
		"rewrite/templates/layout.html",
		"rewrite/templates/search.html",
		"rewrite/templates/about.html",
	)
	if err != nil {
		log.Fatalf("template load error: %v", err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "../whoknows.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	loadTemplates()

	r := mux.NewRouter()

	// statiske filer
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("src/backend/static"))),
	)

	// about (bruger samme layout + body-block)
	r.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, "layout", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// search /
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		lang := r.URL.Query().Get("language")
		if lang == "" {
			lang = "en"
		}

		results := []Result{}
		if q != "" {
			rows, err := db.Query(`
                SELECT title, content AS description, url
                FROM pages
                WHERE language = ? AND content LIKE ?
            `, lang, "%"+q+"%")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			for rows.Next() {
				var it Result
				if err := rows.Scan(&it.Title, &it.Description, &it.URL); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				results = append(results, it)
			}
			if err := rows.Err(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if err := tmpl.ExecuteTemplate(w, "layout", PageData{
			Query:   q,
			Results: results,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	log.Printf("Server kører på http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
