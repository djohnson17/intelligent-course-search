package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Answer  string        `json:"answer"`
	Courses []LocalCourse `json:"courses"`
}

type App struct {
	Catalog *CourseCatalog
}

func (app *App) chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Only POST requests are supported",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var request ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	message := strings.TrimSpace(request.Message)
	if message == "" {
		http.Error(w, `{"error":"Please enter a question"}`, http.StatusBadRequest)
		return
	}

	if app.Catalog == nil {
		http.Error(w, `{"error":"Course catalog is not loaded"}`, http.StatusInternalServerError)
		return
	}

	courses := app.Catalog.Search(message)

	answer := "I couldn't find any matching courses. Try searching by course title, subject, or instructor."

	if len(courses) > 0 {
		answer = "Aye aye! I found some courses that match your search."
	}

	// Keep responses manageable for the chat interface.
	if len(courses) > 20 {
		courses = courses[:20]
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ChatResponse{
		Answer:  answer,
		Courses: courses,
	})
}
