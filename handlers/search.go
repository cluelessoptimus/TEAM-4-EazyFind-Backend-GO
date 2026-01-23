package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"eazyfind/models"
)

const JS_OFFSET = 12

func SearchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// CamelCase support from JS
		pageStr := query.Get("page")
		minCostStr := query.Get("minCost")
		if minCostStr == "" {
			minCostStr = query.Get("min_cost")
		}
		maxCostStr := query.Get("maxCost")
		if maxCostStr == "" {
			maxCostStr = query.Get("max_cost")
		}

		searchTerm := query.Get("name")
		if searchTerm == "" {
			searchTerm = query.Get("q")
		}

		ratingStr := query.Get("rating")
		discountStr := query.Get("discount")
		freeStr := query.Get("free")
		city := query.Get("city")
		area := query.Get("area")

		cuisineIds := query.Get("cuisineIds")
		mealtypeIds := query.Get("mealtypeIds")
		cuisine := query.Get("cuisine")
		mealType := query.Get("meal_type")

		// Location params (V2 Extra)
		latStr := query.Get("lat")
		lonStr := query.Get("lon")
		radiusStr := query.Get("radius")

		page, _ := strconv.Atoi(pageStr)
		if page <= 0 {
			page = 1
		}

		limit := JS_OFFSET
		offset := (page - 1) * limit

		var lat, lon, radius float64
		hasLocation := false
		if city == "" && latStr != "" && lonStr != "" {
			lat, _ = strconv.ParseFloat(latStr, 64)
			lon, _ = strconv.ParseFloat(lonStr, 64)
			radius, _ = strconv.ParseFloat(radiusStr, 64)
			hasLocation = true
		}

		buildQuery := func() (string, string, []interface{}) {
			var args []interface{}
			var conditions []string
			idx := 1

			selectFields := "r.id, r.restaurant_name, r.city, r.area, r.cost_for_two, r.rating, r.latitude, r.longitude, r.image_url, r.effective_discount, r.free"
			distanceExpr := "0.0"
			similarityExpr := "1.0"

			if city != "" {
				conditions = append(conditions, fmt.Sprintf("r.city ILIKE $%d", idx))
				args = append(args, city)
				idx++
			} else if hasLocation {
				distanceExpr = fmt.Sprintf("ST_Distance(r.geo, ST_SetSRID(ST_MakePoint($%d, $%d), 4326))", idx, idx+1)
				searchRadius := radius
				if searchRadius <= 0 {
					searchRadius = 50000
				} // 50km
				conditions = append(conditions, fmt.Sprintf("ST_DWithin(r.geo, ST_SetSRID(ST_MakePoint($%d, $%d), 4326), $%d)", idx, idx+1, idx+2))
				args = append(args, lon, lat, searchRadius)
				idx += 3
			}

			if searchTerm != "" {
				similarityExpr = "1.0"
				conditions = append(conditions, fmt.Sprintf("(r.restaurant_name ILIKE $%d OR r.area ILIKE $%d)", idx, idx))
				args = append(args, "%"+searchTerm+"%")
				idx++
			}

			if area != "" {
				conditions = append(conditions, fmt.Sprintf("r.area ILIKE $%d", idx))
				args = append(args, "%"+area+"%")
				idx++
			}

			if cuisine != "" {
				conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT restaurant_id FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE c.cuisine_name ILIKE $%d)", idx))
				args = append(args, cuisine)
				idx++
			}
			if cuisineIds != "" {
				ids := strings.Split(cuisineIds, ",")
				placeholders := []string{}
				for _, id := range ids {
					placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
					args = append(args, id)
					idx++
				}
				conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT restaurant_id FROM restaurant_cuisines WHERE cuisine_id IN (%s))", strings.Join(placeholders, ",")))
			}

			if mealType != "" {
				conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT restaurant_id FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE m.meal_type ILIKE $%d)", idx))
				args = append(args, mealType)
				idx++
			}
			if mealtypeIds != "" {
				ids := strings.Split(mealtypeIds, ",")
				placeholders := []string{}
				for _, id := range ids {
					placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
					args = append(args, id)
					idx++
				}
				conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT restaurant_id FROM restaurant_meal_types WHERE meal_type_id IN (%s))", strings.Join(placeholders, ",")))
			}

			if minCost, _ := strconv.Atoi(minCostStr); minCost > 0 {
				conditions = append(conditions, fmt.Sprintf("r.cost_for_two >= $%d", idx))
				args = append(args, minCost)
				idx++
			}
			if maxCost, _ := strconv.Atoi(maxCostStr); maxCost > 0 {
				conditions = append(conditions, fmt.Sprintf("r.cost_for_two <= $%d", idx))
				args = append(args, maxCost)
				idx++
			}
			if rating, _ := strconv.ParseFloat(ratingStr, 64); rating > 0 {
				conditions = append(conditions, fmt.Sprintf("r.rating >= $%d", idx))
				args = append(args, rating)
				idx++
			}
			if dValue, _ := strconv.ParseFloat(discountStr, 64); dValue > 0 {
				conditions = append(conditions, fmt.Sprintf("r.effective_discount >= $%d", idx))
				args = append(args, dValue/100.0)
				idx++
			}
			if freeStr == "true" {
				conditions = append(conditions, "r.free = true")
			}

			whereStr := "WHERE 1=1"
			if len(conditions) > 0 {
				whereStr = "WHERE " + strings.Join(conditions, " AND ")
			}

			countQuery := "SELECT COUNT(*) FROM restaurants r " + whereStr
			resultQuery := fmt.Sprintf(`
				SELECT %s, %s as distance, %s as similarity_score,
				       COALESCE((SELECT json_agg(json_build_object('id', c.id, 'cuisine_name', c.cuisine_name)) FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE rc.restaurant_id = r.id), '[]') as cuisines,
				       COALESCE((SELECT json_agg(json_build_object('id', m.id, 'meal_type', m.meal_type)) FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE rmt.restaurant_id = r.id), '[]') as meal_types
				FROM restaurants r %s
			`, selectFields, distanceExpr, similarityExpr, whereStr)

			return countQuery, resultQuery, args
		}

		countQ, resultQ, args := buildQuery()
		var totalCount int
		err := db.QueryRow(countQ, args...).Scan(&totalCount)
		if err != nil {
			log.Println("Count query error:", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"restaurants": []models.Restaurant{}, "pages": 0})
			return
		}

		totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
		if page > totalPages && totalPages > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"restaurants": []models.Restaurant{},
				"pages":       totalPages,
			})
			return
		}

		// Sort matches JS Default: Highest Discount first
		orderBy := "ORDER BY effective_discount DESC"
		sort := query.Get("sort")
		if sort == "rating_desc" {
			orderBy = "ORDER BY rating DESC"
		} else if sort == "cost_asc" {
			orderBy = "ORDER BY cost_for_two ASC"
		} else if (hasLocation || sort == "distance") && latStr != "" {
			orderBy = "ORDER BY distance ASC"
		} else if searchTerm != "" {
			orderBy = "ORDER BY similarity_score DESC"
		}

		finalQuery := fmt.Sprintf("%s %s LIMIT %d OFFSET %d", resultQ, orderBy, limit, offset)
		rows, err := db.Query(finalQuery, args...)
		if err != nil {
			log.Println("Search result query error:", err)
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		results := []models.Restaurant{}
		for rows.Next() {
			var r models.Restaurant
			var cuisinesJSON, mealTypesJSON []byte
			err := rows.Scan(&r.ID, &r.RestaurantName, &r.City, &r.Area, &r.CostForTwo, &r.Rating, &r.Latitude, &r.Longitude, &r.ImageURL, &r.EffectiveDiscount, &r.Free, &r.Distance, &r.SimilarityScore, &cuisinesJSON, &mealTypesJSON)
			if err != nil {
				continue
			}
			json.Unmarshal(cuisinesJSON, &r.Cuisines)
			json.Unmarshal(mealTypesJSON, &r.MealTypes)
			results = append(results, r)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"restaurants": results,
			"pages":       totalPages,
		})
	}
}

func GetRestaurantsByCityHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		city := r.PathValue("city")
		if city == "" {
			http.Error(w, "City is required", http.StatusBadRequest)
			return
		}

		query := `
			SELECT r.id, r.restaurant_name, r.city, r.area, r.cost_for_two, r.rating, r.latitude, r.longitude, r.image_url, r.effective_discount, r.free,
				COALESCE((SELECT json_agg(json_build_object('id', c.id, 'cuisine_name', c.cuisine_name)) FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE rc.restaurant_id = r.id), '[]') as cuisines,
				COALESCE((SELECT json_agg(json_build_object('id', m.id, 'meal_type', m.meal_type)) FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE rmt.restaurant_id = r.id), '[]') as meal_types
			FROM restaurants r
			WHERE r.city ILIKE $1
			ORDER BY r.effective_discount DESC
			LIMIT 10
		`

		rows, err := db.Query(query, city)
		if err != nil {
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		results := []models.Restaurant{}
		for rows.Next() {
			var r models.Restaurant
			var cuisinesJSON, mealTypesJSON []byte
			err := rows.Scan(&r.ID, &r.RestaurantName, &r.City, &r.Area, &r.CostForTwo, &r.Rating, &r.Latitude, &r.Longitude, &r.ImageURL, &r.EffectiveDiscount, &r.Free, &cuisinesJSON, &mealTypesJSON)
			if err != nil {
				continue
			}
			json.Unmarshal(cuisinesJSON, &r.Cuisines)
			json.Unmarshal(mealTypesJSON, &r.MealTypes)
			results = append(results, r)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
