package domain

// Order Statuses
const (
	OrderStatusPending             = "pending"
	OrderStatusPendingVerification = "pending_verification"
	OrderStatusProcessing          = "processing"
	OrderStatusShipped             = "shipped"
	OrderStatusDelivered           = "delivered"
	OrderStatusPaid                = "paid" // L9: Final state for COD where money is collected
	OrderStatusCancelled           = "cancelled"
	OrderStatusRefunded            = "refunded"
	OrderStatusReturned            = "returned"
	OrderStatusFake                = "fake"
)

// Payment Statuses
const (
	PaymentStatusPending       = "pending"
	PaymentStatusPendingVerif  = "pending_verification"
	PaymentStatusPaid          = "paid"
	PaymentStatusFailed        = "failed"
	PaymentStatusPartialPaid   = "partial_paid"
	PaymentStatusPartialRefund = "partial_refund"
	PaymentStatusRefunded      = "refunded"
)

// Payment Methods
const (
	PaymentMethodCOD   = "cod"
	PaymentMethodBKash = "bkash"
	PaymentMethodNagad = "nagad"
)

// List Exports for API
var OrderStatuses = []string{
	OrderStatusPending,
	OrderStatusPendingVerification,
	OrderStatusProcessing,
	OrderStatusShipped,
	OrderStatusDelivered,
	OrderStatusPaid,
	OrderStatusCancelled,
	OrderStatusRefunded,
	OrderStatusReturned,
	OrderStatusFake,
}

var PaymentStatuses = []string{
	PaymentStatusPending,
	PaymentStatusPendingVerif,
	PaymentStatusPaid,
	PaymentStatusFailed,
	PaymentStatusPartialPaid,
	PaymentStatusPartialRefund,
	PaymentStatusRefunded,
}

var PaymentMethods = []string{
	PaymentMethodCOD,
	PaymentMethodBKash,
	PaymentMethodNagad,
}
