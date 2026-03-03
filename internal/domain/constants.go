package domain

// ═══════════════════════════════════════════════════════════════════════════════
// ORDER MANAGEMENT STATE MACHINE — L9 CONTRACT
// ═══════════════════════════════════════════════════════════════════════════════
//
// This file is the SINGLE SOURCE OF TRUTH for the entire order lifecycle.
// Every status, every transition, every side-effect, every flow is defined HERE.
// No implicit logic. No scattered business rules. Everything is explicit.
//
// ┌─────────────────────────────────────────────────────────────────────────┐
// │                        ORDER TYPE FLOWS                                │
// ├─────────────────────────────────────────────────────────────────────────┤
// │                                                                        │
// │  COD Order:                                                            │
// │    pending → pending_verification → processing → shipped → delivered   │
// │    → paid                                                              │
// │                                                                        │
// │  Gateway/Advance Order (money already paid):                           │
// │    pending_verification → processing → shipped → delivered → paid      │
// │                                                                        │
// │  Pre-order (with deposit):                                             │
// │    pending_verification → processing → shipped → delivered → paid      │
// │                                                                        │
// │  Pre-order (zero deposit):                                             │
// │    pending → pending_verification → processing → shipped → delivered   │
// │    → paid                                                              │
// │                                                                        │
// ├─────────────────────────────────────────────────────────────────────────┤
// │                      EXCEPTIONAL FLOWS                                 │
// ├─────────────────────────────────────────────────────────────────────────┤
// │                                                                        │
// │  Cancel (any active state):                                            │
// │    pending|pending_verification|processing|shipped → cancelled         │
// │    Side-effect: AUTO RESTORE STOCK                                     │
// │                                                                        │
// │  Fake (admin marks fraudulent):                                        │
// │    pending|pending_verification|processing → fake                      │
// │    Side-effect: AUTO RESTORE STOCK                                     │
// │                                                                        │
// │  Return (post-delivery):                                               │
// │    shipped|delivered|paid → returned                                   │
// │    Side-effect: AUTO RESTORE STOCK                                     │
// │                                                                        │
// │  Refund (post-delivery/post-payment):                                  │
// │    delivered|paid → refunded                                           │
// │    Side-effect: AUTO SYNC PAYMENT STATUS → refunded                    │
// │                                                                        │
// │  Admin Recovery (mistake fix):                                         │
// │    cancelled → processing (admin only)                                 │
// │    fake → processing (admin only)                                      │
// │    Side-effect: AUTO DEDUCT STOCK (re-reserve)                         │
// │                                                                        │
// └─────────────────────────────────────────────────────────────────────────┘

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 1: ORDER STATUSES
// ═══════════════════════════════════════════════════════════════════════════════

const (
	// OrderStatusPending — Order just created. Awaiting admin action.
	// Entry: COD orders, Pre-orders with zero deposit.
	// Next: pending_verification (admin confirms), cancelled, fake.
	OrderStatusPending = "pending"

	// OrderStatusPendingVerification — Payment info submitted, awaiting admin verification.
	// Entry: Advance/Gateway orders, Pre-orders with deposit, COD after admin review.
	// Next: processing (payment verified), shipped (pre-order bypass), cancelled, fake.
	OrderStatusPendingVerification = "pending_verification"

	// OrderStatusProcessing — Payment verified, order is being prepared.
	// Entry: After admin verifies payment or recovers from cancelled/fake.
	// Next: shipped, cancelled, fake.
	OrderStatusProcessing = "processing"

	// OrderStatusShipped — Order handed to courier. In transit.
	// Entry: After processing is complete.
	// Next: delivered, returned (failed delivery), cancelled (refused at door).
	OrderStatusShipped = "shipped"

	// OrderStatusDelivered — Customer received the package.
	// Entry: Courier confirms delivery.
	// Next: paid (COD collected / final confirmation), returned, refunded.
	OrderStatusDelivered = "delivered"

	// OrderStatusPaid — Full payment confirmed and collected.
	// Entry: COD cash collected after delivery, or final payment confirmation.
	// This is the HAPPY PATH terminal state.
	// Next: refunded, returned (post-payment issues).
	OrderStatusPaid = "paid"

	// OrderStatusCancelled — Order cancelled before delivery.
	// Entry: Admin or system cancels at any pre-delivery stage.
	// Side-effect: STOCK IS AUTO-RESTORED.
	// Next: processing (admin recovery only).
	OrderStatusCancelled = "cancelled"

	// OrderStatusRefunded — Money returned to customer.
	// Entry: Post-delivery or post-payment refund.
	// Side-effect: PAYMENT STATUS AUTO-SYNCED to refunded.
	// Terminal state (no forward transitions).
	OrderStatusRefunded = "refunded"

	// OrderStatusReturned — Product physically returned by customer.
	// Entry: Customer returns after shipment/delivery/payment.
	// Side-effect: STOCK IS AUTO-RESTORED.
	// Terminal state (no forward transitions).
	OrderStatusReturned = "returned"

	// OrderStatusFake — Admin marks order as fraudulent.
	// Entry: Admin identifies fraud at any pre-delivery stage.
	// Side-effect: STOCK IS AUTO-RESTORED.
	// Next: processing (admin recovery only, if mistake).
	OrderStatusFake = "fake"
)

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 2: VALID ORDER TRANSITIONS (Finite State Machine)
// ═══════════════════════════════════════════════════════════════════════════════
//
// Rule: If a transition is NOT in this map, it is FORBIDDEN. Period.
// The usecase layer MUST check this map before allowing any status change.

var ValidTransitions = map[string][]string{
	// ── Happy Path ─────────────────────────────────────────────────────────
	OrderStatusPending: {
		OrderStatusPendingVerification, // Admin starts verification
		OrderStatusCancelled,           // Admin/System cancels
		OrderStatusFake,                // Admin marks as fraud
	},
	OrderStatusPendingVerification: {
		OrderStatusProcessing, // Payment verified → start preparing
		OrderStatusShipped,    // Pre-order direct ship (skip processing)
		OrderStatusCancelled,  // Cancel before processing
		OrderStatusFake,       // Fraud detected
	},
	OrderStatusProcessing: {
		OrderStatusShipped,   // Hand to courier
		OrderStatusCancelled, // Cancel during preparation
		OrderStatusFake,      // Fraud detected late
	},
	OrderStatusShipped: {
		OrderStatusDelivered, // Courier confirms delivery
		OrderStatusReturned,  // Failed delivery / customer refused
		OrderStatusCancelled, // Shipment recalled / refused at door
	},
	OrderStatusDelivered: {
		OrderStatusPaid,     // COD collected / final payment confirmed
		OrderStatusReturned, // Customer returns product
		OrderStatusRefunded, // Money refunded post-delivery
	},
	OrderStatusPaid: {
		OrderStatusRefunded, // Post-payment refund
		OrderStatusReturned, // Post-payment return
	},

	// ── Exceptional / Recovery ─────────────────────────────────────────────
	OrderStatusCancelled: {
		OrderStatusProcessing, // Admin recovery: was cancelled by mistake
	},
	OrderStatusFake: {
		OrderStatusProcessing, // Admin recovery: was marked fake by mistake
	},

	// ── Terminal States (no transitions out) ────────────────────────────────
	// OrderStatusRefunded: {} — implicitly empty, no forward transitions
	// OrderStatusReturned: {} — implicitly empty, no forward transitions
}

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 3: PAYMENT STATUSES
// ═══════════════════════════════════════════════════════════════════════════════

const (
	// PaymentStatusPending — No payment received yet. Default for COD orders.
	PaymentStatusPending = "pending"

	// PaymentStatusPendingVerif — Customer submitted payment info (trxID, provider).
	// Admin must verify this manually.
	PaymentStatusPendingVerif = "pending_verification"

	// PaymentStatusPaid — Full payment confirmed.
	PaymentStatusPaid = "paid"

	// PaymentStatusFailed — Payment attempt failed (gateway error, invalid trx).
	PaymentStatusFailed = "failed"

	// PaymentStatusPartialPaid — Pre-order deposit collected, remainder outstanding.
	PaymentStatusPartialPaid = "partial_paid"

	// PaymentStatusPartialRefund — Part of the payment refunded. Remainder kept.
	PaymentStatusPartialRefund = "partial_refund"

	// PaymentStatusRefunded — Full payment refunded to customer.
	PaymentStatusRefunded = "refunded"
)

// ValidPaymentTransitions defines the strict FSM for payment status changes.
// Rule: If a transition is NOT in this map, it is FORBIDDEN.
var ValidPaymentTransitions = map[string][]string{
	PaymentStatusPending: {
		PaymentStatusPendingVerif, // Customer submits payment info
		PaymentStatusPaid,         // Direct confirmation (e.g., COD collected)
		PaymentStatusFailed,       // Payment attempt failed
	},
	PaymentStatusPendingVerif: {
		PaymentStatusPaid,   // Admin verifies payment
		PaymentStatusFailed, // Admin rejects payment
	},
	PaymentStatusFailed: {
		PaymentStatusPendingVerif, // Customer retries payment
		PaymentStatusPending,      // Reset to pending
	},
	PaymentStatusPaid: {
		PaymentStatusPartialRefund, // Partial refund issued
		PaymentStatusRefunded,      // Full refund issued
	},
	PaymentStatusPartialPaid: {
		PaymentStatusPaid,          // Remaining balance collected
		PaymentStatusPartialRefund, // Partial refund on deposit
		PaymentStatusRefunded,      // Full refund of deposit
	},
	PaymentStatusPartialRefund: {
		PaymentStatusRefunded, // Remaining amount also refunded
	},
	// PaymentStatusRefunded: {} — Terminal, no forward transitions
}

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 4: PAYMENT METHODS
// ═══════════════════════════════════════════════════════════════════════════════

const (
	PaymentMethodCOD        = "cod"
	PaymentMethodAdvance    = "advance" // Manual advance payment (bKash, Nagad — customer provides trxID)
	PaymentMethodBKash      = "bkash"
	PaymentMethodNagad      = "nagad"
	PaymentMethodSSLCommerz = "sslcommerz" // SSLCommerz payment gateway
)

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 5: SIDE-EFFECTS MAP
// ═══════════════════════════════════════════════════════════════════════════════
//
// Defines what automated actions MUST happen when an order moves to a status.
// The usecase layer reads this and executes accordingly.
// This is the contract — if it's not here, it doesn't happen automatically.

type SideEffect string

const (
	SideEffectRestoreStock      SideEffect = "restore_stock"       // Return items to inventory
	SideEffectDeductStock       SideEffect = "deduct_stock"        // Re-reserve items from inventory
	SideEffectSyncPaymentPaid   SideEffect = "sync_payment_paid"   // Set PaymentStatus → paid
	SideEffectSyncPaymentRefund SideEffect = "sync_payment_refund" // Set PaymentStatus → refunded
)

// StatusSideEffects maps each order status to the automated actions
// that MUST be triggered when an order transitions TO that status.
var StatusSideEffects = map[string][]SideEffect{
	OrderStatusCancelled: {SideEffectRestoreStock},
	OrderStatusFake:      {SideEffectRestoreStock},
	OrderStatusReturned:  {SideEffectRestoreStock},
	OrderStatusRefunded:  {SideEffectSyncPaymentRefund},
	OrderStatusPaid:      {SideEffectSyncPaymentPaid},
	// Recovery: when admin moves cancelled/fake → processing, stock must be re-deducted
	OrderStatusProcessing: {SideEffectDeductStock},
}

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 6: ORDER TYPE FLOWS (Expected Happy-Path Sequences)
// ═══════════════════════════════════════════════════════════════════════════════
//
// These define the EXPECTED linear progression for each order type.
// Used for validation, progress tracking, and admin UI step indicators.

type OrderType string

const (
	OrderTypeCOD        OrderType = "cod"         // Cash on Delivery
	OrderTypeAdvance    OrderType = "advance"     // Advance payment via gateway
	OrderTypePreorder   OrderType = "preorder"    // Pre-order with deposit
	OrderTypePreorderND OrderType = "preorder_nd" // Pre-order with no deposit
)

// OrderTypeFlows maps each order type to its expected happy-path status sequence.
// These are the IDEAL paths. Exceptional flows (cancel, fake, return, refund)
// can branch off at any permitted point per ValidTransitions.
var OrderTypeFlows = map[OrderType][]string{
	// COD: Customer pays nothing upfront. Cash collected on delivery.
	// Step 1: pending          — Order received, awaiting admin review
	// Step 2: pending_verification — Admin reviews order details
	// Step 3: processing       — Order confirmed, being prepared
	// Step 4: shipped          — Handed to courier
	// Step 5: delivered        — Customer received package
	// Step 6: paid             — COD cash collected and confirmed
	OrderTypeCOD: {
		OrderStatusPending,
		OrderStatusPendingVerification,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusPaid,
	},

	// Advance: Customer already paid via bKash/Nagad/gateway.
	// Step 1: pending_verification — Payment submitted, awaiting admin verification
	// Step 2: processing            — Payment verified, order being prepared
	// Step 3: shipped               — Handed to courier
	// Step 4: delivered             — Customer received package
	// Step 5: paid                  — Final payment confirmation
	OrderTypeAdvance: {
		OrderStatusPendingVerification,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusPaid,
	},

	// Pre-order with deposit: Customer pays deposit upfront.
	// Same flow as Advance — deposit is already submitted.
	// Step 1: pending_verification — Deposit submitted, awaiting verification
	// Step 2: processing            — Deposit verified, order queued for production
	// Step 3: shipped               — Product ready and shipped
	// Step 4: delivered             — Customer received package
	// Step 5: paid                  — Remaining balance collected
	OrderTypePreorder: {
		OrderStatusPendingVerification,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusPaid,
	},

	// Pre-order with no deposit: Customer reserves without paying.
	// Same as COD flow — starts at pending.
	// Step 1: pending          — Order received, awaiting admin review
	// Step 2: pending_verification — Admin reviews pre-order details
	// Step 3: processing       — Pre-order confirmed, queued
	// Step 4: shipped          — Product ready and shipped
	// Step 5: delivered        — Customer received package
	// Step 6: paid             — Full payment collected
	OrderTypePreorderND: {
		OrderStatusPending,
		OrderStatusPendingVerification,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusPaid,
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 7: VALIDATION HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

// IsValidTransition checks if moving from `current` to `next` is permitted.
func IsValidTransition(current, next string) bool {
	allowed, exists := ValidTransitions[current]
	if !exists {
		return false // Current status has no outgoing transitions (terminal)
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// IsValidPaymentTransition checks if a payment status change is permitted.
func IsValidPaymentTransition(current, next string) bool {
	allowed, exists := ValidPaymentTransitions[current]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// IsTerminalOrderStatus returns true if the status has no forward transitions.
func IsTerminalOrderStatus(status string) bool {
	_, exists := ValidTransitions[status]
	return !exists // If no entry in map, it's terminal
}

// IsStockRestoringStatus returns true if transitioning TO this status
// should trigger automatic stock restoration.
func IsStockRestoringStatus(status string) bool {
	return status == OrderStatusCancelled ||
		status == OrderStatusFake ||
		status == OrderStatusReturned
}

// IsRecoveryTransition returns true if this is a cancelled/fake → processing recovery.
// Recovery transitions require stock re-deduction.
func IsRecoveryTransition(from, to string) bool {
	return (from == OrderStatusCancelled || from == OrderStatusFake) &&
		to == OrderStatusProcessing
}

// GetSideEffects returns the list of side-effects for a given transition.
// Takes both from and to status to handle conditional side-effects (e.g., recovery).
func GetSideEffects(from, to string) []SideEffect {
	// Recovery transitions: stock was already restored when cancelled/faked,
	// so moving back to processing means we need to re-deduct.
	if IsRecoveryTransition(from, to) {
		return []SideEffect{SideEffectDeductStock}
	}

	// Normal transitions: check the target status
	if effects, exists := StatusSideEffects[to]; exists {
		// For processing in non-recovery context, no side-effects
		if to == OrderStatusProcessing {
			return nil
		}
		return effects
	}
	return nil
}

// DetermineInitialStatus returns the correct starting order status and payment status
// based on the order type and whether payment info was provided.
func DetermineInitialStatus(paymentMethod string, isPreorder bool, depositRequired float64, hasTrxID bool) (orderStatus, paymentStatus string) {
	switch {
	case isPreorder && depositRequired > 0:
		// Pre-order with deposit: starts at pending_verification
		return OrderStatusPendingVerification, PaymentStatusPendingVerif

	case isPreorder && depositRequired <= 0:
		// Pre-order with no deposit: starts at pending
		return OrderStatusPending, PaymentStatusPending

	case paymentMethod == PaymentMethodCOD:
		// COD: starts at pending, no payment yet
		return OrderStatusPending, PaymentStatusPending

	case hasTrxID:
		// Advance with trx info: starts at pending_verification
		return OrderStatusPendingVerification, PaymentStatusPendingVerif

	default:
		// Fallback: pending
		return OrderStatusPending, PaymentStatusPending
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// SECTION 8: LIST EXPORTS FOR API
// ═══════════════════════════════════════════════════════════════════════════════

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
	PaymentMethodAdvance,
	PaymentMethodBKash,
	PaymentMethodNagad,
	PaymentMethodSSLCommerz,
}
