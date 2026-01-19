package search

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindNearby(
	ctx context.Context,
	lat float64,
	lon float64,
	radiusKm float64,
) ([]Place, error) {

	const query = `
	SELECT
		id,
		name,
		lat,
		lon
	FROM places
	WHERE ST_DWithin(
		geography(ST_MakePoint(lon, lat)),
		geography(ST_MakePoint($1, $2)),
		$3
	)
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		lon,
		lat,
		radiusKm*1000,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Place

	for rows.Next() {
		var p Place
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Latitude,
			&p.Longitude,
		); err != nil {
			return nil, err
		}
		results = append(results, p)
	}

	return results, nil
}
