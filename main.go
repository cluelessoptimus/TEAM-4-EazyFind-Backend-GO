package main

import (
	"log"
	"net/http"
	"os"

	"eazyfind/database"
	"eazyfind/handlers"
	"eazyfind/worker"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// Load environment variables (optional, for local dev)
	_ = godotenv.Load()

	// Connect to Database
	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Start Background Worker
	go worker.StartGeocodingWorker(db)

	// Setup Routes (Exact JS Alignment)
	mux := http.NewServeMux()

	// Primary Routes
	mux.HandleFunc("/restaurants", handlers.SearchHandler(db))
	mux.HandleFunc("/restaurants/{city}", handlers.GetRestaurantsByCityHandler(db))
	mux.HandleFunc("/cities", handlers.CitiesHandler(db))
	mux.HandleFunc("/meal-types", handlers.MealTypesHandler(db))
	mux.HandleFunc("/cuisines", handlers.CuisinesHandler(db))

	// Aliases & Extras
	mux.HandleFunc("/api/restaurants", handlers.SearchHandler(db))
	mux.HandleFunc("/api/search", handlers.SearchHandler(db))
	mux.HandleFunc("/api/cities", handlers.CitiesHandler(db))
	mux.HandleFunc("/api/cuisines", handlers.CuisinesHandler(db))
	mux.HandleFunc("/api/meal-types", handlers.MealTypesHandler(db))
	mux.HandleFunc("/api/suggestions", handlers.SuggestionsHandler(db))
	mux.HandleFunc("/api/restaurants/{city}", handlers.GetRestaurantsByCityHandler(db))

	// CORS Setup - Explicitly allow the frontend dev server
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)

	// Start Server (JS Port: 3003)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3003"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed:", err)
	}
}
