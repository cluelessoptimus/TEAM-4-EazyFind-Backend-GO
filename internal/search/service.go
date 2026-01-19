package search

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SearchNearby(
	ctx context.Context,
	lat float64,
	lon float64,
	radiusKm float64,
) ([]Place, error) {

	if lat < -90 || lat > 90 {
		return nil, errors.New("latitude must be between -90 and 90")
	}

	if lon < -180 || lon > 180 {
		return nil, errors.New("longitude must be between -180 and 180")
	}

	if radiusKm <= 0 {
		return nil, errors.New("radius must be greater than zero")
	}

	if radiusKm > 50 {
		return nil, errors.New("radius exceeds maximum allowed limit")
	}

	return s.repo.FindNearby(ctx, lat, lon, radiusKm)
}
