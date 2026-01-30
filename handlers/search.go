package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"eazyfind/models"
)

const (
	DefaultLimit = 12
)

type SearchParams struct {
	Page        int
	Limit       int
	Offset      int
	Name        string
	MinCost     int
	MaxCost     int
	Rating      float64
	Discount    float64
	Free        bool
	City        string
	Area        string
	CuisineIds  string
	MealTypeIds string
	Cuisine     string
	MealType    string
	Cuisines    string
	MealTypes   string
	Lat         float64
	Lon         float64
	Radius      float64
	HasLocation bool
	Sort        string
}

// ParseSearchParams extracts and normalizes restaurant search filters from the URL query.
func ParseSearchParams(query url.Values) SearchParams {
	p := SearchParams{
		Limit: DefaultLimit,
	}

	p.Page, _ = strconv.Atoi(query.Get("page"))
	if p.Page <= 0 {
		p.Page = 1
	}
	p.Offset = (p.Page - 1) * p.Limit

	p.Name = query.Get("name")
	if p.Name == "" {
		p.Name = query.Get("q")
	}

	p.MinCost, _ = strconv.Atoi(query.Get("minCost"))
	if p.MinCost == 0 {
		p.MinCost, _ = strconv.Atoi(query.Get("min_cost"))
	}
	p.MaxCost, _ = strconv.Atoi(query.Get("maxCost"))
	if p.MaxCost == 0 {
		p.MaxCost, _ = strconv.Atoi(query.Get("max_cost"))
	}

	p.Rating, _ = strconv.ParseFloat(query.Get("rating"), 64)
	if d, _ := strconv.ParseFloat(query.Get("discount"), 64); d > 0 {
		p.Discount = d / 100.0
	}
	p.Free = query.Get("free") == "true"

	p.City = query.Get("city")
	p.Area = query.Get("area")
	p.CuisineIds = query.Get("cuisineIds")
	p.MealTypeIds = query.Get("mealtypeIds")
	p.Cuisine = query.Get("cuisine")
	p.MealType = query.Get("meal_type")
	p.Cuisines = query.Get("cuisines")
	p.MealTypes = query.Get("mealtypes")

	latStr, lonStr := query.Get("lat"), query.Get("lon")
	if latStr != "" && lonStr != "" {
		p.Lat, _ = strconv.ParseFloat(latStr, 64)
		p.Lon, _ = strconv.ParseFloat(lonStr, 64)
		p.Radius, _ = strconv.ParseFloat(query.Get("radius"), 64)
		if p.Radius <= 0 {
			p.Radius = 50000
		}
		p.HasLocation = true
	}

	p.Sort = query.Get("sort")
	return p
}

// BuildSearchQueries generates SQL WHERE clauses and arguments based on provided SearchParams.
// It handles spatial queries (PostGIS), text similarity, and relational filters.
func BuildSearchQueries(p SearchParams) (string, string, []interface{}) {
	var args []interface{}
	var conditions []string
	idx := 1

	distanceExpr := "0.0"

	// Calculate distance expression whenever coordinates are provided, regardless of city filter.
	// This ensures that even when filtering by city, the frontend receives proximity data.
	if p.HasLocation {
		distanceExpr = fmt.Sprintf("ST_Distance(r.geo, ST_SetSRID(ST_MakePoint($%d, $%d), 4326))", idx, idx+1)
		args = append(args, p.Lon, p.Lat)
		idx += 2

		// Enforce a 100km proximity limit specifically for "Best Deals" to ensure relevance.
		if p.Sort == "" || p.Sort == "discount" {
			conditions = append(conditions, fmt.Sprintf("ST_DWithin(r.geo, ST_SetSRID(ST_MakePoint($%d, $%d), 4326), 100000)", idx-2, idx-1))
		}
	}

	if p.City != "" {
		conditions = append(conditions, fmt.Sprintf("r.city ILIKE $%d", idx))
		args = append(args, p.City)
		idx++
	} else if p.HasLocation {
		// If NO city is provided but location is active, use ST_DWithin for discovery.
		conditions = append(conditions, fmt.Sprintf("ST_DWithin(r.geo, ST_SetSRID(ST_MakePoint($%d, $%d), 4326), $%d)", idx-2, idx-1, idx))
		args = append(args, p.Radius)
		idx++
	}

	if p.Name != "" {
		conditions = append(conditions, fmt.Sprintf("(r.restaurant_name ILIKE $%d OR r.area ILIKE $%d)", idx, idx))
		args = append(args, "%"+p.Name+"%")
		idx++
	}

	if p.Area != "" {
		conditions = append(conditions, fmt.Sprintf("r.area ILIKE $%d", idx))
		args = append(args, "%"+p.Area+"%")
		idx++
	}

	if p.Cuisine != "" {
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT rc.restaurant_id FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE c.cuisine_name ILIKE $%d)", idx))
		args = append(args, p.Cuisine)
		idx++
	}

	if p.Cuisines != "" {
		names := strings.Split(p.Cuisines, ",")
		var placeholders []string
		for _, name := range names {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, strings.TrimSpace(name))
			idx++
		}
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT rc.restaurant_id FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE c.cuisine_name IN (%s))", strings.Join(placeholders, ",")))
	}

	if p.CuisineIds != "" {
		ids := strings.Split(p.CuisineIds, ",")
		var placeholders []string
		for _, id := range ids {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, id)
			idx++
		}
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT restaurant_id FROM restaurant_cuisines WHERE cuisine_id IN (%s))", strings.Join(placeholders, ",")))
	}

	if p.MealType != "" {
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT rmt.restaurant_id FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE m.meal_type ILIKE $%d)", idx))
		args = append(args, p.MealType)
		idx++
	}

	if p.MealTypes != "" {
		names := strings.Split(p.MealTypes, ",")
		var placeholders []string
		for _, name := range names {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, strings.TrimSpace(name))
			idx++
		}
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT rmt.restaurant_id FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE m.meal_type IN (%s))", strings.Join(placeholders, ",")))
	}

	if p.MealTypeIds != "" {
		ids := strings.Split(p.MealTypeIds, ",")
		var placeholders []string
		for _, id := range ids {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, id)
			idx++
		}
		conditions = append(conditions, fmt.Sprintf("r.id IN (SELECT restaurant_id FROM restaurant_meal_types WHERE meal_type_id IN (%s))", strings.Join(placeholders, ",")))
	}

	if p.MinCost > 0 {
		conditions = append(conditions, fmt.Sprintf("r.cost_for_two >= $%d", idx))
		args = append(args, p.MinCost)
		idx++
	}
	if p.MaxCost > 0 {
		conditions = append(conditions, fmt.Sprintf("r.cost_for_two <= $%d", idx))
		args = append(args, p.MaxCost)
		idx++
	}
	if p.Rating > 0 {
		conditions = append(conditions, fmt.Sprintf("r.rating >= $%d", idx))
		args = append(args, p.Rating)
		idx++
	}
	if p.Discount > 0 {
		conditions = append(conditions, fmt.Sprintf("r.effective_discount >= $%d", idx))
		args = append(args, p.Discount)
		idx++
	}
	if p.Free {
		conditions = append(conditions, "r.free = true")
	}

	conditions = append(conditions, "r.is_duplicate = false")

	whereStr := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := "SELECT COUNT(*) FROM restaurants r " + whereStr

	selectFields := "r.id, r.restaurant_name, r.city, r.area, r.cost_for_two, r.rating, r.latitude, r.longitude, r.image_url, r.effective_discount, r.free, r.offer, r.percentage"
	resultQuery := fmt.Sprintf(`
		SELECT %s, %s as distance,
		       COALESCE((SELECT json_agg(json_build_object('id', c.id, 'cuisine_name', c.cuisine_name)) FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE rc.restaurant_id = r.id), '[]') as cuisines,
		       COALESCE((SELECT json_agg(json_build_object('id', m.id, 'meal_type', m.meal_type)) FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE rmt.restaurant_id = r.id), '[]') as meal_types
		FROM restaurants r %s
	`, selectFields, distanceExpr, whereStr)

	return countQuery, resultQuery, args
}

func ScanRestaurant(rows *sql.Rows, hasExtraFields bool) (models.Restaurant, error) {
	var r models.Restaurant
	var cuisinesJSON, mealTypesJSON []byte
	var err error

	if hasExtraFields {
		err = rows.Scan(&r.ID, &r.RestaurantName, &r.City, &r.Area, &r.CostForTwo, &r.Rating, &r.Latitude, &r.Longitude, &r.ImageURL, &r.EffectiveDiscount, &r.Free, &r.Offer, &r.Percentage, &r.Distance, &cuisinesJSON, &mealTypesJSON)
	} else {
		err = rows.Scan(&r.ID, &r.RestaurantName, &r.City, &r.Area, &r.CostForTwo, &r.Rating, &r.Latitude, &r.Longitude, &r.ImageURL, &r.EffectiveDiscount, &r.Free, &r.Offer, &r.Percentage, &cuisinesJSON, &mealTypesJSON)
	}

	if err != nil {
		return r, err
	}

	json.Unmarshal(cuisinesJSON, &r.Cuisines)
	json.Unmarshal(mealTypesJSON, &r.MealTypes)
	return r, nil
}

// SearchHandler coordinates the multi-stage search process: parameter parsing,
// result counting for pagination, and final data retrieval with ordering.
func SearchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := ParseSearchParams(r.URL.Query())
		countQ, resultQ, args := BuildSearchQueries(p)

		var totalCount int
		err := db.QueryRow(countQ, args...).Scan(&totalCount)
		if err != nil {
			log.Println("Count query error:", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"restaurants": []models.Restaurant{}, "pages": 0})
			return
		}

		totalPages := int(math.Ceil(float64(totalCount) / float64(p.Limit)))
		if p.Page > totalPages && totalPages > 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"restaurants": []models.Restaurant{}, "pages": totalPages})
			return
		}

		orderBy := "ORDER BY effective_discount DESC, id ASC"
		switch p.Sort {
		case "rating_desc":
			orderBy = "ORDER BY rating DESC, id ASC"
		case "cost_asc":
			orderBy = "ORDER BY cost_for_two ASC, id ASC"
		default:
		}

		finalQuery := fmt.Sprintf("%s %s LIMIT %d OFFSET %d", resultQ, orderBy, p.Limit, p.Offset)
		rows, err := db.Query(finalQuery, args...)
		if err != nil {
			log.Println("Search result query error:", err)
			http.Error(w, "Something went wrong", http.StatusBadRequest)
			return
		}
		defer rows.Close()

		results := []models.Restaurant{}
		for rows.Next() {
			if res, err := ScanRestaurant(rows, true); err == nil {
				results = append(results, res)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"restaurants": results,
			"pages":       totalPages,
			"total_count": totalCount,
		})
	}
}

// GetRestaurantsByCityHandler provides a high-performance entry point for city-specific
// restaurant discovery. It leverages case-insensitive matching and filters out
// duplicate entries to ensure a clean result set for the initial landing views.
func GetRestaurantsByCityHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		city := r.PathValue("city")
		if city == "" {
			http.Error(w, "City is required", http.StatusBadRequest)
			return
		}

		// The query uses complex sub-query aggregation to fetch related metadata
		// (cuisines, meal types) in a single database round-trip, significantly
		// reducing network overhead. The ILIKE filter provides flexible city
		// matching without the complexity of trigram indexes.
		query := `
			SELECT r.id, r.restaurant_name, r.city, r.area, r.cost_for_two, r.rating, r.latitude, r.longitude, r.image_url, r.effective_discount, r.free, r.offer, r.percentage,
				COALESCE((SELECT json_agg(json_build_object('id', c.id, 'cuisine_name', c.cuisine_name)) FROM restaurant_cuisines rc JOIN cuisines c ON rc.cuisine_id = c.id WHERE rc.restaurant_id = r.id), '[]') as cuisines,
				COALESCE((SELECT json_agg(json_build_object('id', m.id, 'meal_type', m.meal_type)) FROM restaurant_meal_types rmt JOIN meal_types m ON rmt.meal_type_id = m.id WHERE rmt.restaurant_id = r.id), '[]') as meal_types
			FROM restaurants r
			WHERE r.city ILIKE $1 AND r.is_duplicate = false
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
			if res, err := ScanRestaurant(rows, false); err == nil {
				results = append(results, res)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
