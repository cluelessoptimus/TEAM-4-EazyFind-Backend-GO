package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Suggestion struct {
	Type  string  `json:"type"` // "restaurant", "cuisine", "area"
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

func SuggestionsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if len(q) < 2 {
			json.NewEncoder(w).Encode([]Suggestion{})
			return
		}

		// Use Trigram similarity to find matches across restaurants, cuisines, and areas
		// We'll use a UNION for a unified list
		query := `
			(SELECT 'restaurant' as type, restaurant_name as text, similarity(restaurant_name, $1) as score
			 FROM restaurants
			 WHERE restaurant_name % $1
			 LIMIT 5)
			UNION ALL
			(SELECT 'cuisine' as type, cuisine_name as text, similarity(cuisine_name, $1) as score
			 FROM cuisines
			 WHERE cuisine_name % $1
			 LIMIT 3)
			UNION ALL
			(SELECT 'area' as type, area as text, similarity(area, $1) as score
			 FROM restaurants
			 WHERE area % $1
			 GROUP BY area
			 LIMIT 3)
			ORDER BY score DESC
			LIMIT 10
		`

		rows, err := db.Query(query, q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		suggestions := []Suggestion{}
		for rows.Next() {
			var s Suggestion
			if err := rows.Scan(&s.Type, &s.Text, &s.Score); err != nil {
				continue
			}
			suggestions = append(suggestions, s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(suggestions)
	}
}
