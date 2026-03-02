package domain

import (
	"context"
	"time"
)

type ShippingZone struct {
	ID        int32     `json:"id"`
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	Cost      float64   `json:"cost"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ConfigRepository interface {
	GetActiveShippingZones(ctx context.Context) ([]ShippingZone, error)
	GetAllShippingZones(ctx context.Context) ([]ShippingZone, error)
	GetShippingZoneByID(ctx context.Context, id int32) (*ShippingZone, error)
	GetShippingZoneByKey(ctx context.Context, key string) (*ShippingZone, error)
	CreateShippingZone(ctx context.Context, zone *ShippingZone) (*ShippingZone, error)
	UpdateShippingZone(ctx context.Context, zone *ShippingZone) (*ShippingZone, error)
	UpdateShippingZoneCost(ctx context.Context, key string, cost float64) error
	DeleteShippingZone(ctx context.Context, id int32) error
}
