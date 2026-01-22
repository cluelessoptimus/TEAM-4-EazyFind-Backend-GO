package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// Configurations for tuning
const (
	BatchSize        = 100
	WorkerPoolSize   = 20
	IntervalDuration = 5 * time.Second
)

func StartGeocodingWorker(db *sql.DB) {
	log.Printf("Starting optimized Geocoding Worker (Batch: %d, Concurrency: %d, Interval: %v)", BatchSize, WorkerPoolSize, IntervalDuration)
	ticker := time.NewTicker(IntervalDuration)
	go func() {
		for range ticker.C {
			processPendingRestaurants(db)
		}
	}()
}

func processPendingRestaurants(db *sql.DB) {
	// Fetch a larger batch of pending restaurants
	query := fmt.Sprintf("SELECT id, restaurant_name, city FROM restaurants WHERE geo_status = 'PENDING' LIMIT %d", BatchSize)
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Worker query error:", err)
		return
	}
	defer rows.Close()

	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		log.Println("GOOGLE_MAPS_API_KEY not set, skipping geocoding")
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, WorkerPoolSize)

	for rows.Next() {
		var id int64
		var name, city string
		if err := rows.Scan(&id, &name, &city); err != nil {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire token

		go func(id int64, name, city string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release token

			lat, lon, err := fetchCoordinates(name, city, apiKey)
			if err != nil {
				log.Printf("Geocoding failed for [%d] %s: %v", id, name, err)
				return
			}

			// Update DB with lat/lon and PostGIS point
			_, err = db.Exec(`
				UPDATE restaurants 
				SET latitude = $1, longitude = $2, 
				    geo = ST_SetSRID(ST_MakePoint($2, $1), 4326),
				    geo_status = 'RESOLVED'
				WHERE id = $3
			`, lat, lon, id)

			if err != nil {
				log.Printf("Failed to update restaurant %d: %v", id, err)
			} else {
				log.Printf("Resolved: %s (%v, %v)", name, lat, lon)
			}
		}(id, name, city)
	}

	wg.Wait()
}

func fetchCoordinates(name, city, apiKey string) (float64, float64, error) {
	query := fmt.Sprintf("%s, %s", name, city)
	apiURL := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s", url.QueryEscape(query), apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, err
	}

	if result.Status != "OK" {
		return 0, 0, fmt.Errorf("API error: %s", result.Status)
	}

	if len(result.Results) == 0 {
		return 0, 0, fmt.Errorf("no results found")
	}

	return result.Results[0].Geometry.Location.Lat, result.Results[0].Geometry.Location.Lng, nil
}
