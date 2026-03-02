package domain

import (
	"context"
	"time"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type User struct {
	ID        string    `json:"id"` // UUID
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Phone     string    `json:"phone"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Address struct {
	ID     string `json:"id"` // addr_...
	UserID string `json:"userId"`
	Label  string `json:"label"` // "Home", "Office"

	// Contact Info
	ContactEmail string `json:"contactEmail"`
	Phone        string `json:"phone"`

	// Recipient
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`

	// Location
	DeliveryZone string `json:"deliveryZone"` // e.g., "Inside Dhaka"
	Division     string `json:"division"`
	District     string `json:"district"`
	Thana        string `json:"thana"`
	AddressLine  string `json:"addressLine"` // House/Road/Block/Flat
	Landmark     string `json:"landmark"`
	PostalCode   string `json:"postalCode"`

	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
}

type RefreshToken struct {
	Token     string    `json:"token"` // UUID
	UserID    string    `json:"userId"`
	User      User      `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	Revoked   bool      `json:"revoked"`
	Device    string    `json:"device"` // "Chrome on Linux", etc.
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetAll(ctx context.Context, limit, offset int) ([]*User, int64, error)
	Update(ctx context.Context, user *User) error

	UpdateProfile(ctx context.Context, id, firstName, lastName, phone string) (*User, error)

	// Addresses
	AddAddress(ctx context.Context, addr *Address) error
	UpdateAddress(ctx context.Context, addr *Address) error
	GetAddresses(ctx context.Context, userID string) ([]Address, error)
	DeleteAddress(ctx context.Context, id, userID string) error

	// Refresh Tokens
	SaveRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}
