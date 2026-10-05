// Command coursesite serves the course website in site/ on port 8000.
// Run it from the repository root: make course
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := ":8000"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	files := http.FileServer(http.Dir("."))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "site/index.html")
			return
		}
		files.ServeHTTP(w, r)
	})

	log.Printf("course site on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
