package models

// Restaurant represents the core model for a dining establishment, including
// metadata, location, and associated relational data (cuisines, meal types).
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
	Distance  float64    `json:"distance,omitempty"`
	Cuisines  []Cuisine  `json:"cuisines,omitempty"`
	MealTypes []MealType `json:"meal_types,omitempty"`
}

// Cuisine represents a specific culinary category used for filtering and search.
type Cuisine struct {
	ID          int64  `json:"id,string"`
	CuisineName string `json:"cuisine_name"`
}

// MealType defines the time or category of a meal (e.g., Breakfast, Dinner).
type MealType struct {
	ID       int64  `json:"id,string"`
	MealType string `json:"meal_type"`
}

// City represents the cities table
type City struct {
	ID        int64   `json:"id,string"`
	CityName  string  `json:"city_name" db:"city_name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	GeoStatus string  `json:"geo_status"`
}
