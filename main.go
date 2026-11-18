package main

import (
	"html/template"
	"log"
	"net/http"
)

func main() {
	catalog, err := LoadCourseCatalog("Fall 2024 Class Schedule 08082024.csv")
	if err != nil {
		log.Fatalf("Could not load course catalog: %v", err)
	}

	log.Printf("Loaded %d courses", len(catalog.Courses))

	page, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Could not load HTML template: %v", err)
	}

	app := &App{
		Catalog: catalog,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := page.Execute(w, nil); err != nil {
			log.Printf("Error rendering page: %v", err)
		}
	})

	http.HandleFunc("/api/chat", app.chatHandler)

	http.Handle("/static/", http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("static")),
	))

	log.Println("Bikini Bottom University Course Finder running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
