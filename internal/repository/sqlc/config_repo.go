package sqlcrepo

import (
	"context"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type configRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewConfigRepository(db *pgxpool.Pool) domain.ConfigRepository {
	return &configRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *configRepository) GetActiveShippingZones(ctx context.Context) ([]domain.ShippingZone, error) {
	zones, err := r.queries.GetActiveShippingZones(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.ShippingZone, len(zones))
	for i, z := range zones {
		result[i] = domain.ShippingZone{
			ID:        z.ID,
			Key:       z.Key,
			Label:     z.Label,
			Cost:      numericToFloat64(z.Cost),
			IsActive:  z.IsActive,
			CreatedAt: pgtimeToTime(z.CreatedAt),
			UpdatedAt: pgtimeToTime(z.UpdatedAt),
		}
	}
	return result, nil
}

func (r *configRepository) GetAllShippingZones(ctx context.Context) ([]domain.ShippingZone, error) {
	zones, err := r.queries.GetAllShippingZones(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.ShippingZone, len(zones))
	for i, z := range zones {
		result[i] = domain.ShippingZone{
			ID:        z.ID,
			Key:       z.Key,
			Label:     z.Label,
			Cost:      numericToFloat64(z.Cost),
			IsActive:  z.IsActive,
			CreatedAt: pgtimeToTime(z.CreatedAt),
			UpdatedAt: pgtimeToTime(z.UpdatedAt),
		}
	}
	return result, nil
}

func (r *configRepository) GetShippingZoneByID(ctx context.Context, id int32) (*domain.ShippingZone, error) {
	z, err := r.queries.GetShippingZoneByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.ShippingZone{
		ID:        z.ID,
		Key:       z.Key,
		Label:     z.Label,
		Cost:      numericToFloat64(z.Cost),
		IsActive:  z.IsActive,
		CreatedAt: pgtimeToTime(z.CreatedAt),
		UpdatedAt: pgtimeToTime(z.UpdatedAt),
	}, nil
}

func (r *configRepository) GetShippingZoneByKey(ctx context.Context, key string) (*domain.ShippingZone, error) {
	z, err := r.queries.GetShippingZoneByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	return &domain.ShippingZone{
		ID:        z.ID,
		Key:       z.Key,
		Label:     z.Label,
		Cost:      numericToFloat64(z.Cost),
		IsActive:  z.IsActive,
		CreatedAt: pgtimeToTime(z.CreatedAt),
		UpdatedAt: pgtimeToTime(z.UpdatedAt),
	}, nil
}

func (r *configRepository) CreateShippingZone(ctx context.Context, zone *domain.ShippingZone) (*domain.ShippingZone, error) {
	z, err := r.queries.CreateShippingZone(ctx, sqlc.CreateShippingZoneParams{
		Key:      zone.Key,
		Label:    zone.Label,
		Cost:     float64ToNumeric(zone.Cost),
		IsActive: zone.IsActive,
	})
	if err != nil {
		return nil, err
	}

	return &domain.ShippingZone{
		ID:        z.ID,
		Key:       z.Key,
		Label:     z.Label,
		Cost:      numericToFloat64(z.Cost),
		IsActive:  z.IsActive,
		CreatedAt: pgtimeToTime(z.CreatedAt),
		UpdatedAt: pgtimeToTime(z.UpdatedAt),
	}, nil
}

func (r *configRepository) UpdateShippingZone(ctx context.Context, zone *domain.ShippingZone) (*domain.ShippingZone, error) {
	z, err := r.queries.UpdateShippingZone(ctx, sqlc.UpdateShippingZoneParams{
		ID:       zone.ID,
		Label:    zone.Label,
		Cost:     float64ToNumeric(zone.Cost),
		IsActive: zone.IsActive,
	})
	if err != nil {
		return nil, err
	}

	return &domain.ShippingZone{
		ID:        z.ID,
		Key:       z.Key,
		Label:     z.Label,
		Cost:      numericToFloat64(z.Cost),
		IsActive:  z.IsActive,
		CreatedAt: pgtimeToTime(z.CreatedAt),
		UpdatedAt: pgtimeToTime(z.UpdatedAt),
	}, nil
}

func (r *configRepository) UpdateShippingZoneCost(ctx context.Context, key string, cost float64) error {
	return r.queries.UpdateShippingZoneCost(ctx, sqlc.UpdateShippingZoneCostParams{
		Key:  key,
		Cost: float64ToNumeric(cost),
	})
}

func (r *configRepository) DeleteShippingZone(ctx context.Context, id int32) error {
	return r.queries.DeleteShippingZone(ctx, id)
}
