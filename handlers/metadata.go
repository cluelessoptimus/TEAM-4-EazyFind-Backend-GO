package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"eazyfind/models"
)

// CitiesHandler retrieves all available cities from the database for filter population.
func CitiesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, city_name, COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(geo_status, 'PENDING') FROM cities ORDER BY id ASC")
		if err != nil {
			log.Println("Cities query error:", err)
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		cities := []models.City{}
		for rows.Next() {
			var c models.City
			if err := rows.Scan(&c.ID, &c.CityName, &c.Latitude, &c.Longitude, &c.GeoStatus); err == nil {
				cities = append(cities, c)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cities)
	}
}

// CuisinesHandler retrieves the full list of cuisines to populate the searchable multi-select filter.
func CuisinesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, cuisine_name FROM cuisines ORDER BY id ASC")
		if err != nil {
			log.Println("Cuisines query error:", err)
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		cuisines := []models.Cuisine{}
		for rows.Next() {
			var c models.Cuisine
			if err := rows.Scan(&c.ID, &c.CuisineName); err == nil {
				cuisines = append(cuisines, c)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cuisines)
	}
}

// MealTypesHandler retrieves all defined meal categories (e.g., Breakfast, Lunch, Dinner).
func MealTypesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, meal_type FROM meal_types ORDER BY id ASC")
		if err != nil {
			log.Println("MealTypes query error:", err)
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		meals := []models.MealType{}
		for rows.Next() {
			var m models.MealType
			if err := rows.Scan(&m.ID, &m.MealType); err == nil {
				meals = append(meals, m)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meals)
	}
}

// DetectCityHandler identifies the user's city based on latitude and longitude coordinates,
// using reverse geocoding via Geoapify or a nearest-neighbor distance search in the database.
func DetectCityHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latStr := r.URL.Query().Get("lat")
		lonStr := r.URL.Query().Get("lon")

		if latStr == "" || lonStr == "" {
			http.Error(w, "lat and lon are required", http.StatusBadRequest)
			return
		}

		lat, _ := strconv.ParseFloat(latStr, 64)
		lon, _ := strconv.ParseFloat(lonStr, 64)

		log.Printf("Detecting city for lat: %v, lon: %v", lat, lon)

		apiKey := os.Getenv("GEOAPIFY_API_KEY")
		resolvedCity := ""

		if apiKey != "" {
			apiURL := fmt.Sprintf("https://api.geoapify.com/v1/geocode/reverse?lat=%f&lon=%f&apiKey=%s", lat, lon, apiKey)
			resp, err := http.Get(apiURL)
			if err != nil {
				log.Println("Geoapify request error:", err)
			} else {
				defer resp.Body.Close()
				var result struct {
					Features []struct {
						Properties struct {
							City string `json:"city"`
						} `json:"properties"`
					} `json:"features"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && len(result.Features) > 0 {
					city := result.Features[0].Properties.City
					log.Printf("Geoapify resolved city: %s", city)
					if city == "Delhi" || city == "Noida" || city == "Gurugram" || city == "New Delhi" || city == "Gurgaon" {
						resolvedCity = "delhi-ncr"
					} else {
						resolvedCity = city
					}
				} else if err != nil {
					log.Println("Geoapify decode error:", err)
				}
			}
		}

		var dbCity string
		if resolvedCity != "" {
			err := db.QueryRow("SELECT city_name FROM cities WHERE city_name ILIKE $1", resolvedCity).Scan(&dbCity)
			if err == nil {
				log.Printf("Found match in DB for resolved city: %s", dbCity)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"city": dbCity})
				return
			}
			log.Printf("Resolved city %s not found in DB, falling back to closest", resolvedCity)
		}

		// Use fmt.Sprintf for coordinates to avoid prepared statement issues in this specific environment if params fail
		// Cast the point to geography explicitly to match the 'geo' column type
		query := fmt.Sprintf(`
			SELECT city_name 
			FROM cities 
			ORDER BY ST_Distance(geo, ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography) ASC 
			LIMIT 1
		`, lon, lat)

		err := db.QueryRow(query).Scan(&dbCity)

		if err != nil {
			log.Printf("Closest city query error for lat %f, lon %f: %v", lat, lon, err)
			http.Error(w, "Could not detect city", http.StatusInternalServerError)
			return
		}

		log.Printf("Closest city found in DB: %s", dbCity)

		if dbCity == "delhi-ncr" || dbCity == "delhi" || dbCity == "noida" || dbCity == "gurugram" {
			dbCity = "delhi-ncr"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"city": dbCity})
	}
}
