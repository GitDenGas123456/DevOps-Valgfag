package main

import (
    "fmt"
    "net/http"

    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()

    r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        fmt.Fprint(w, `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Search</title></head>
<body>
  <div>
    <input id="search-input" placeholder="Search..." />
    <button id="search-button" onclick="alert('Søgning kobles på i næste trin')">Search</button>
  </div>
</body>
</html>`)
    }).Methods("GET")

    addr := ":8080"
    println("Server kører på http://localhost" + addr)
    if err := http.ListenAndServe(addr, r); err != nil {
        panic(err)
    }
}
