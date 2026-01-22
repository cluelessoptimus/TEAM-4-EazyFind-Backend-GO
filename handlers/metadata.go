package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"eazyfind/models"
)

func CitiesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, city_name FROM cities ORDER BY id ASC")
		if err != nil {
			log.Println("Cities query error:", err)
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		cities := []models.City{}
		for rows.Next() {
			var c models.City
			if err := rows.Scan(&c.ID, &c.CityName); err == nil {
				cities = append(cities, c)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cities)
	}
}

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
