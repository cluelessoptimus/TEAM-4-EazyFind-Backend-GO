package models

// Restaurant represents the restaurant table exactly as per JS Prisma schema
type Restaurant struct {
	ID                int64   `json:"id,string"`
	RestaurantName    string  `json:"restaurant_name" db:"restaurant_name"`
	URL               string  `json:"url,omitempty"`
	City              string  `json:"city"`
	Area              string  `json:"area,omitempty"`
	CostForTwo        int     `json:"cost_for_two"`
	Rating            float64 `json:"rating"`
	Page              int     `json:"page"`
	Offer             string  `json:"offer,omitempty"`
	Percentage        string  `json:"percentage,omitempty"`
	EffectiveDiscount float64 `json:"effective_discount"`
	Free              bool    `json:"free"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	GeoStatus         string  `json:"geo_status"`
	ImageURL          string  `json:"image_url,omitempty"`

	// Extras for V2 (included in JSON to be safe)
	Distance        float64    `json:"distance,omitempty"`
	SimilarityScore float64    `json:"similarity_score,omitempty"`
	Cuisines        []Cuisine  `json:"cuisines,omitempty"`
	MealTypes       []MealType `json:"meal_types,omitempty"`
}

// Cuisine represents the cuisines table
type Cuisine struct {
	ID          int64  `json:"id,string"`
	CuisineName string `json:"cuisine_name"`
}

// MealType represents the meal_types table
type MealType struct {
	ID       int64  `json:"id,string"`
	MealType string `json:"meal_type"`
}

// City represents the cities table
type City struct {
	ID       int64  `json:"id,string"`
	CityName string `json:"city_name"`
}
