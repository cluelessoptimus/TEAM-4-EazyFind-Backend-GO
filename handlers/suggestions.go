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

		// Use ILIKE for prefix/substring matching to avoid fuzzy trigram results
		query := `
			(SELECT 'restaurant' as type, restaurant_name as text, 1.0 as score
			 FROM restaurants
			 WHERE restaurant_name ILIKE $1
			 LIMIT 5)
			UNION ALL
			(SELECT 'cuisine' as type, cuisine_name as text, 1.0 as score
			 FROM cuisines
			 WHERE cuisine_name ILIKE $1
			 LIMIT 3)
			UNION ALL
			(SELECT 'area' as type, area as text, 1.0 as score
			 FROM restaurants
			 WHERE area ILIKE $1
			 GROUP BY area
			 LIMIT 3)
			ORDER BY score DESC, text ASC
			LIMIT 10
		`

		searchParams := "%" + q + "%"
		rows, err := db.Query(query, searchParams)
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
