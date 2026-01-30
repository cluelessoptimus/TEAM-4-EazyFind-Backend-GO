# EazyFind Backend

A robust, high-currency Go server designed to power the EazyFind restaurant discovery platform. The backend focuses on high-performance geospatial queries and efficient metadata management.

## Key Features

- **Geospatial Search**: Advanced PostGIS-powered proximity search for restaurants based on user coordinates.
- **Optimized Indexing**: High-performance B-tree indexes for rapid sorting by price, rating, and real-time discounts.
- **Metadata Management**: Centralized handlers for cities, cuisines, and meal-type metadata population.
- **Automated Geocoding**: Background worker system for synchronizing restaurant coordinates with external geocoding providers.
- **Duplicate Prevention**: Integrated logic to identify and filter redundant data entries while preserving database integrity.

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: Standard Library (`net/http`)
- **Database**: PostgreSQL with PostGIS extensions
- **Data Access**: Standard `database/sql` for maximum performance and control.

## Getting Started

### Prerequisites

- Go (v1.23 or higher)
- PostgreSQL with PostGIS enabled

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/cluelessoptimus/TEAM-4-EazyFind-Backend-GO.git
   ```

2. Configure environment variables in `.env`:
   ```env
   DATABASE_URL=postgres://user:pass@host:port/dbname
   PORT=8080
   GEOAPIFY_API_KEY=your_key_here
   ```

3. Apply the database schema:
   ```bash
   psql -f database_sql/schema.sql
   ```

4. Start the server:
   ```bash
   go run cmd/server/main.go
   ```

## API Documentation

- `GET /api/search`: Filtered restaurant discovery.
- `GET /api/detect-city`: Coordinate-based city identification.
- `GET /api/cities`: List of available service areas.
- `GET /api/cuisines`: Global list of restaurant cuisines.
- `GET /api/mealtypes`: Standardized meal categories.

## Architecture

- `cmd/server`: Application entry point and router initialization.
- `handlers`: Functional entry points for API endpoints.
- `models`: Shared data structures and database mappings.
- `database`: Pool management and connection logic.
- `worker`: Background tasks for data enrichment and geocoding.
