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

// main initializes the server, database connections, and background workers.
func main() {
	_ = godotenv.Load()

	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	go worker.StartGeocodingWorker(db)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /restaurants", handlers.SearchHandler(db))
	mux.HandleFunc("GET /restaurants/{city}", handlers.GetRestaurantsByCityHandler(db))
	mux.HandleFunc("GET /cities", handlers.CitiesHandler(db))
	mux.HandleFunc("GET /meal-types", handlers.MealTypesHandler(db))
	mux.HandleFunc("GET /cuisines", handlers.CuisinesHandler(db))

	mux.HandleFunc("GET /api/restaurants", handlers.SearchHandler(db))
	mux.HandleFunc("GET /api/search", handlers.SearchHandler(db))
	mux.HandleFunc("GET /api/cities", handlers.CitiesHandler(db))
	mux.HandleFunc("GET /api/detect-city", handlers.DetectCityHandler(db))
	mux.HandleFunc("GET /api/cuisines", handlers.CuisinesHandler(db))
	mux.HandleFunc("GET /api/meal-types", handlers.MealTypesHandler(db))
	mux.HandleFunc("GET /api/restaurants/{city}", handlers.GetRestaurantsByCityHandler(db))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3003"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed:", err)
	}
}
