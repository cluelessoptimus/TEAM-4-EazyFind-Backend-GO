--EazyFind Schema

-- Enable PostGIS extension for spatial grouping and distance calculations
CREATE EXTENSION IF NOT EXISTS postgis;

-- Restaurants table: Core entity storing establishment details and calculated discounts
CREATE TABLE IF NOT EXISTS restaurants (
    id BIGSERIAL PRIMARY KEY,
    restaurant_name TEXT,
    url TEXT,
    city TEXT,
    area TEXT,
    cost_for_two INTEGER,
    rating NUMERIC(2, 1),
    page INTEGER,
    offer TEXT,
    percentage TEXT,
    effective_discount DOUBLE PRECISION,
    free BOOLEAN DEFAULT false,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    geo GEOGRAPHY(POINT, 4326),
    geo_status TEXT DEFAULT 'PENDING',
    image_url TEXT,
    is_duplicate BOOLEAN DEFAULT false
);

-- Cuisines table: Metadata lookup for restaurant cuisines
CREATE TABLE IF NOT EXISTS cuisines (
    id BIGSERIAL PRIMARY KEY,
    cuisine_name TEXT UNIQUE
);

-- Meal Types table: Metadata lookup for meal categories (e.g., Breakfast, Lunch)
CREATE TABLE IF NOT EXISTS meal_types (
    id BIGSERIAL PRIMARY KEY,
    meal_type TEXT UNIQUE
);

-- Cities table: Geographic metadata for supported cities
CREATE TABLE IF NOT EXISTS cities (
    id BIGSERIAL PRIMARY KEY,
    city_name TEXT UNIQUE,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    geo GEOGRAPHY(POINT, 4326),
    geo_status TEXT DEFAULT 'PENDING'
);

-- Restaurant -> Cuisines junction table
CREATE TABLE IF NOT EXISTS restaurant_cuisines (
    restaurant_id BIGINT REFERENCES restaurants(id) ON DELETE CASCADE,
    cuisine_id BIGINT REFERENCES cuisines(id) ON DELETE CASCADE,
    PRIMARY KEY (restaurant_id, cuisine_id)
);

-- Restaurant -> Meal Types junction table
CREATE TABLE IF NOT EXISTS restaurant_meal_types (
    restaurant_id BIGINT REFERENCES restaurants(id) ON DELETE CASCADE,
    meal_type_id BIGINT REFERENCES meal_types(id) ON DELETE CASCADE,
    PRIMARY KEY (restaurant_id, meal_type_id)
);

-- Spatial Indexes for efficient location-based lookups
CREATE INDEX IF NOT EXISTS idx_restaurants_geo
ON restaurants USING GIST (geo)
WHERE geo_status = 'RESOLVED';

CREATE INDEX IF NOT EXISTS idx_cities_geo 
ON cities USING GIST (geo) 
WHERE geo_status = 'RESOLVED';

-- Relational Indexes
CREATE INDEX IF NOT EXISTS idx_restaurants_city ON restaurants(city);
CREATE INDEX IF NOT EXISTS idx_restaurants_is_duplicate ON restaurants(is_duplicate);

-- Sorting & Filter Optimization (Standard B-tree)
CREATE INDEX IF NOT EXISTS idx_restaurants_rating ON restaurants(rating DESC);
CREATE INDEX IF NOT EXISTS idx_restaurants_cost ON restaurants(cost_for_two);
CREATE INDEX IF NOT EXISTS idx_restaurants_discount ON restaurants(effective_discount DESC);
