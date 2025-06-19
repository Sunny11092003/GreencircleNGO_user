package main

import (
	"context"
	"html/template"
	"log"
	"net/http"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"google.golang.org/api/option"

	"github.com/gorilla/mux"
)

var firebaseClient *db.Client

type Tree struct {
	ID                string              `json:"ID"`
	Name              string              `json:"Name"`
	Published         bool                `json:"Published"`
	Volunteer         string              `json:"Volunteer"`
	Timestamp         string              `json:"Timestamp"`
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
	r.HandleFunc("/tree/{id}", handleTreePage)
	http.Handle("/", r)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

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

	tmpl := template.Must(template.ParseFiles("static/index.html"))
	tmpl.Execute(w, tree)
}
