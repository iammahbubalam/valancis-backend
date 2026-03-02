package domain

import (
	"context"
	"time"
)

type OrderFilter struct {
	Page          int
	Limit         int
	Status        string
	PaymentStatus string
	IsPreOrder    *bool
	Search        string
}

// --- Cart Entities ---

type Cart struct {
	ID        string     `json:"id"`
	UserID    *string    `json:"userId"` // Optional: guest carts could be supported, but for now we link to User if logged in
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type CartItem struct {
	ID           string   `json:"id"`
	CartID       string   `json:"cartId"`
	ProductID    string   `json:"productId"`
	Product      Product  `json:"product"`
	VariantID    *string  `json:"variantId"`
	VariantName  *string  `json:"variantName"`
	VariantImage *string  `json:"variantImage"`
	Quantity     int      `json:"quantity"`
	Price        float64  `json:"price"`     // Effective price
	SalePrice    *float64 `json:"salePrice"` // Effective sale price (if any)
}

// --- Order Entities ---

type Order struct {
	ID              string      `json:"id"`
	UserID          string      `json:"userId"`
	User            User        `json:"user"`
	Status          string      `json:"status"` // pending, processing, shipped, delivered, cancelled
	TotalAmount     float64     `json:"totalAmount"`
	ShippingFee     float64     `json:"shippingFee"`
	ShippingAddress JSONB       `json:"shippingAddress"`
	PaymentMethod   string      `json:"paymentMethod"`
	PaymentStatus   string      `json:"paymentStatus"`
	PaidAmount      float64     `json:"paidAmount"`
	RefundedAmount  float64     `json:"refundedAmount"`
	PaymentDetails  JSONB       `json:"paymentDetails"`
	IsPreOrder      bool        `json:"isPreOrder"`
	Items           []OrderItem `json:"items"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}

type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"orderId"`
	ProductID string  `json:"productId"`
	Product   Product `json:"product"`
	VariantID *string `json:"variantId"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"` // Price at time of purchase
}

// --- Interfaces ---

type OrderHistory struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"orderId"`
	PreviousStatus *string   `json:"previousStatus"`
	NewStatus      string    `json:"newStatus"`
	Reason         *string   `json:"reason"`
	CreatedBy      *string   `json:"createdBy"`             // UserID
	CreatedName    *string   `json:"createdName,omitempty"` // Enriched
	CreatedAt      time.Time `json:"createdAt"`
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	GetByUserID(ctx context.Context, userID string) ([]Order, error)
	GetAll(ctx context.Context, filter OrderFilter) ([]Order, int64, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdatePaymentStatus(ctx context.Context, id, status string) error

	// Cart
	GetCartByUserID(ctx context.Context, userID string) (*Cart, error)
	CreateCart(ctx context.Context, cart *Cart) error
	GetCartWithItems(ctx context.Context, userID string) ([]CartItem, error)
	UpsertCartItemAtomic(ctx context.Context, userID, cartID, productID string, variantID *string, quantity int) ([]CartItem, error)
	AtomicRemoveCartItem(ctx context.Context, userID, productID, variantID string) error
	ClearCart(ctx context.Context, cartID string) error

	// Refunds & History
	CreateRefund(ctx context.Context, orderID string, amount float64, reason string, restock bool, createdBy *string) error
	CreateOrderHistory(ctx context.Context, history *OrderHistory) error
	GetOrderHistory(ctx context.Context, orderID string) ([]OrderHistory, error)

	HasPurchasedProduct(ctx context.Context, userID, productID string) (bool, error)
}
