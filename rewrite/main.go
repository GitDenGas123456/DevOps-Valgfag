package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

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
	// valgfrit:
	User    *struct{ Username string }
	Flashes []string
}

func parseTmpl() (*template.Template, error) {
	return template.ParseFiles(
		"templates/layout.html",
		"templates/search.html",
	)
}

func main() {
	// Kør fra mappen: ...\DevOps-Valgfag\rewrite
	db, err := sql.Open("sqlite", "../whoknows.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := mux.NewRouter()

	// /static/*
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Parse templates på hver request, så du altid ser dine seneste ændringer
		tmpl, err := parseTmpl()
		if err != nil {
			http.Error(w, "template parse error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query().Get("q")
		language := r.URL.Query().Get("language")
		if language == "" {
			language = "en"
		}

		results := make([]Result, 0)
		if q != "" {
			rows, err := db.Query(`
				SELECT title, description, url
				FROM pages
				WHERE language = ? AND content LIKE ?
			`, language, "%"+q+"%")
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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "layout", PageData{
			Query:   q,
			Results: results,
			// User:    &struct{ Username string }{"demo"},
			// Flashes: []string{"Welcome back"},
		}); err != nil {
			http.Error(w, "template exec error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	addr := ":8080"
	log.Printf("Server kører på http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))

}
