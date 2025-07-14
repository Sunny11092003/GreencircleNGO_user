package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

var firebaseClient *db.Client

// Updated Tree struct to match Firebase schema
type Tree struct {
	ID                string              `json:"ID"`
	Name              string              `json:"Name"`
	Published         bool                `json:"Published"`
	QR                bool                `json:"QR"`
	Saved             bool                `json:"Saved"`
	Volunteer         string              `json:"volunteerName"`
	Timestamp         string              `json:"timestamp"`
	Botanical         string              `json:"botanical"`
	Category          string              `json:"category"`
	Classification    map[string]string   `json:"classification"`
	Description       string              `json:"description"`
	Environmental     string              `json:"environmentalBenefits"`
	Images            []map[string]string `json:"images"`
	LastUpdated       string              `json:"lastUpdated"`
	Location          map[string]string   `json:"location"`
	MedicinalBenefits string              `json:"medicinalBenefits"`
	Native            string              `json:"native"`
	UID               string              `json:"uid"`
}

func main() {
	ctx := context.Background()

	conf := &firebase.Config{
		DatabaseURL: "https://treeqrsystem-default-rtdb.firebaseio.com/",
	}
	opt := option.WithCredentialsFile("treeqrsystem-firebase-adminsdk-fbsvc-8b56ea8e0c.json")
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase: %v", err)
	}
	firebaseClient, err = app.Database(ctx)
	if err != nil {
		log.Fatalf("Error initializing database client: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/{id}", handleTreePage).Methods("GET")
	r.HandleFunc("/generate-description/{id}", generateDescriptionHandler).Methods("GET")
	r.HandleFunc("/speak", speakHandler).Methods("GET")

	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Renders the tree details HTML page
func handleTreePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	treeID := vars["id"]

	ctx := context.Background()
	ref := firebaseClient.NewRef("trees/" + treeID)

	var tree Tree
	if err := ref.Get(ctx, &tree); err != nil {
		http.Error(w, "Tree not found", http.StatusNotFound)
		return
	}

	// Initialize optional maps if nil
	if tree.Classification == nil {
		tree.Classification = map[string]string{}
	}
	if tree.Location == nil {
		tree.Location = map[string]string{}
	}
	if tree.Images == nil {
		tree.Images = []map[string]string{}
	}

	tmpl := template.Must(template.ParseFiles("static/index.html"))
	tmpl.Execute(w, tree)
}

// Generates AI description using OpenRouter
func generateDescriptionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	treeID := vars["id"]

	ctx := context.Background()
	ref := firebaseClient.NewRef("trees/" + treeID)

	var tree Tree
	if err := ref.Get(ctx, &tree); err != nil {
		http.Error(w, "Tree not found", http.StatusNotFound)
		return
	}

	description, err := generateTreeInfoAI(tree.Name)
	if err != nil {
		log.Printf("AI generation failed: %v", err)
		http.Error(w, "Failed to generate description", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, description)
}

// Converts given text to speech using local TTS
func speakHandler(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if text == "" {
		http.Error(w, "Missing 'text' parameter", http.StatusBadRequest)
		return
	}

	encodedText := url.QueryEscape(text)
	fullURL := "http://localhost:5002/speak?text=" + encodedText

	resp, err := http.Get(fullURL)
	if err != nil {
		http.Error(w, "Failed to get TTS audio", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

// Calls OpenRouter (Gemma 3) to generate AI-based tree info
func generateTreeInfoAI(treeName string) (string, error) {
	reqBody := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}{
		Model: "google/gemma-3-4b-it:free",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: "You are an expert botanist and storyteller. When asked, respond as if you are the tree speaking about yourself in a friendly, informative tone."},
			{Role: "user", Content: fmt.Sprintf("Introduce yourself as the %s tree. Share your characteristics, origin, importance, and something interesting about you as if you're telling your own story.", treeName)},
		},
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer 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")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if len(res.Choices) > 0 {
		return res.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response from AI")
}
