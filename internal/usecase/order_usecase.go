package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"valancis-backend/internal/domain"
	"valancis-backend/internal/infrastructure/facebook"
	"valancis-backend/pkg/utils"
)

type OrderUsecase struct {
	orderRepo   domain.OrderRepository
	productRepo domain.ProductRepository
	configRepo  domain.ConfigRepository
	couponRepo  domain.CouponRepository
	txManager   domain.TransactionManager
	capiClient  *facebook.CAPIClient
}

func NewOrderUsecase(repo domain.OrderRepository, pRepo domain.ProductRepository, configRepo domain.ConfigRepository, cRepo domain.CouponRepository, txManager domain.TransactionManager, capiClient *facebook.CAPIClient) *OrderUsecase {
	return &OrderUsecase{
		orderRepo:   repo,
		productRepo: pRepo,
		configRepo:  configRepo,
		couponRepo:  cRepo,
		txManager:   txManager,
		capiClient:  capiClient,
	}
}

// --- Cart Logic ---

func (u *OrderUsecase) GetMyCart(ctx context.Context, userID string) (*domain.Cart, error) {
	items, err := u.orderRepo.GetCartWithItems(ctx, userID)
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "sql: no rows in result set" {
			cart := &domain.Cart{
				ID:     utils.GenerateUUID(),
				UserID: &userID,
			}
			if createErr := u.orderRepo.CreateCart(ctx, cart); createErr != nil {
				return nil, createErr
			}
			return cart, nil
		}
		return nil, err
	}

	cart, cartErr := u.orderRepo.GetCartByUserID(ctx, userID)
	if cartErr != nil || cart == nil {
		if cartErr != nil {
			slog.Error("Usecase: GetMyCart - GetCartByUserID failed", "error", cartErr)
		} else {
			slog.Info("Usecase: GetMyCart - Cart not found, creating new one")
		}

		cart = &domain.Cart{
			ID:     utils.GenerateUUID(),
			UserID: &userID,
		}
		if createErr := u.orderRepo.CreateCart(ctx, cart); createErr != nil {
			slog.Error("Usecase: GetMyCart - CreateCart failed", "error", createErr)
			return nil, createErr
		}
		slog.Info("Usecase: GetMyCart - Created new cart", "cart_id", cart.ID)
	}

	cart.Items = items
	return cart, nil
}

func (u *OrderUsecase) AddToCart(ctx context.Context, userID string, productID string, variantID *string, quantity int) (*domain.Cart, error) {
	slog.Info("Usecase: AddToCart", "user_id", userID, "product_id", productID, "variant_id", variantID, "quantity", quantity)

	// L9: Enforce "Everything is a Variant" rule
	if variantID == nil {
		product, err := u.productRepo.GetProductByID(ctx, productID)
		if err != nil {
			return nil, fmt.Errorf("product not found: %w", err)
		}
		if len(product.Variants) == 1 {
			// Auto-select the only variant
			vID := product.Variants[0].ID
			variantID = &vID
			slog.Info("Usecase: AddToCart - Auto-resolved single variant", "variant_id", vID)
		} else if len(product.Variants) > 1 {
			return nil, fmt.Errorf("please select a variant option")
		} else {
			return nil, fmt.Errorf("product configuration error: item has no variants")
		}
	} else {
		// Validation: Verify the variant actually belongs to the product?
		// Basic Upsert will check FK, but maybe good to be safe.
		// For now, trust the ID if provided.
	}

	// Get cart (create if not exists)
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		slog.Error("Usecase: AddToCart - GetMyCart failed", "error", err)
		return nil, err
	}
	slog.Info("Usecase: AddToCart - Got Cart", "cart_id", cart.ID)

	// Check existing quantity for the SPECIFIC variant
	existingQty := 0
	for _, item := range cart.Items {
		if item.ProductID == productID {
			// Check if variant matches (handling nil pointers)
			match := false
			v1 := "nil"
			if item.VariantID != nil {
				v1 = *item.VariantID
			}
			v2 := "nil"
			if variantID != nil {
				v2 = *variantID
			}

			if item.VariantID == nil && variantID == nil {
				match = true
			} else if item.VariantID != nil && variantID != nil && *item.VariantID == *variantID {
				match = true
			}

			slog.Debug("Usecase: AddToCart - Matching variant", "item_variant", v1, "req_variant", v2, "match", match)

			if match {
				existingQty = item.Quantity
				break
			}
		}
	}

	// Calculate new total
	newTotal := existingQty + quantity
	slog.Info("Usecase: AddToCart - Final calculation", "existing_qty", existingQty, "add_qty", quantity, "new_total", newTotal)

	// Use atomic upsert with new total
	items, err := u.orderRepo.UpsertCartItemAtomic(ctx, userID, cart.ID, productID, variantID, newTotal)
	if err != nil {
		slog.Error("Usecase: AddToCart - UpsertCartItemAtomic DB Error", "error", err)
		if variantID != nil {
			return nil, fmt.Errorf("insufficient stock or product unavailable for variant %s", *variantID)
		} else {
			return nil, fmt.Errorf("insufficient stock or product unavailable")
		}
	}
	slog.Info("Usecase: AddToCart - Success", "items_count", len(items))

	// Return cart with all items
	return &domain.Cart{
		ID:     items[0].CartID,
		UserID: &userID,
		Items:  items,
	}, nil
}

// RemoveFromCart removes a product from the user's cart
func (u *OrderUsecase) RemoveFromCart(ctx context.Context, userID string, productID string, variantID string) (*domain.Cart, error) {
	// Atomic O(1) Remove
	if err := u.orderRepo.AtomicRemoveCartItem(ctx, userID, productID, variantID); err != nil {
		return nil, err
	}

	// Fetch fresh cart to return
	return u.GetMyCart(ctx, userID)
}

func (u *OrderUsecase) UpdateCartItemQuantity(ctx context.Context, userID string, productID string, variantID *string, quantity int) (*domain.Cart, error) {
	variantIDStr := "nil"
	if variantID != nil {
		variantIDStr = *variantID
	}
	slog.Info("Usecase: UpdateCartItemQuantity", "user_id", userID, "product_id", productID, "variant_id", variantIDStr, "quantity", quantity)

	if quantity <= 0 {
		if variantID == nil {
			return nil, fmt.Errorf("variant_id required to remove item")
		}
		return u.RemoveFromCart(ctx, userID, productID, *variantID)
	}

	// Make sure we have a cart handle
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 🔥 ATOMIC: 1 DB CALL DOES EVERYTHING 🔥
	items, err := u.orderRepo.UpsertCartItemAtomic(ctx, userID, cart.ID, productID, variantID, quantity)
	if err != nil {
		slog.Error("Usecase: UpdateCartItemQuantity - DB Error", "error", err)
		return nil, fmt.Errorf("failed to update cart: %w", err)
	}

	// If no items returned, the atomic operation didn't insert/update anything
	// This means EITHER user has no cart OR product stock is insufficient
	if len(items) == 0 {
		// Simplified error since we rely on atomic query
		return nil, fmt.Errorf("unable to update cart: insufficient stock or invalid item")
	}

	// Success! Build cart from returned items
	cart = &domain.Cart{
		UserID: &userID,
		Items:  items,
	}

	if len(items) > 0 {
		cart.ID = items[0].CartID
	}

	return cart, nil
}

// --- Order Logic ---

type CheckoutReq struct {
	Address         domain.JSONB `json:"address"`
	Payment         string       `json:"paymentMethod"`
	CouponCode      string       `json:"couponCode,omitempty"`
	PaymentTrxID    string       `json:"paymentTrxId,omitempty"`
	PaymentProvider string       `json:"paymentProvider,omitempty"`
	PaymentPhone    string       `json:"paymentPhone,omitempty"`
	// Tracking Information
	FBP       string `json:"fbp"`
	FBC       string `json:"fbc"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

// ApplyCouponResp represents the result of applying a coupon
type ApplyCouponResp struct {
	Valid          bool    `json:"valid"`
	Code           string  `json:"code"`
	DiscountAmount float64 `json:"discountAmount"`
	NewTotal       float64 `json:"newTotal"`
	Message        string  `json:"message"`
}

func (u *OrderUsecase) ApplyCoupon(ctx context.Context, userID, code string) (*ApplyCouponResp, error) {
	// 1. Get Cart
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart")
	}

	// 2. Calculate Subtotal
	var subtotal float64
	for _, item := range cart.Items {
		// Use the resolved prices from CartItem
		price := item.Price
		if item.SalePrice != nil {
			price = *item.SalePrice
		}
		subtotal += price * float64(item.Quantity)
	}

	// 3. Validate Coupon
	res, err := u.couponRepo.ValidateCoupon(ctx, code, subtotal)
	if err != nil {
		// If validation query returns no rows or error
		return &ApplyCouponResp{Valid: false, Message: "Invalid coupon code"}, nil
	}

	if res.ValidationStatus != "valid" {
		return &ApplyCouponResp{Valid: false, Message: fmt.Sprintf("Coupon is %s", res.ValidationStatus), Code: code}, nil
	}

	// 4. Calculate Discount
	discount := 0.0
	if res.Type == "percentage" {
		discount = subtotal * (res.Value / 100)
	} else {
		discount = res.Value
	}

	// Cap discount at subtotal (no negative total)
	if discount > subtotal {
		discount = subtotal
	}

	return &ApplyCouponResp{
		Valid:          true,
		Code:           code,
		DiscountAmount: discount,
		NewTotal:       subtotal - discount,
		Message:        "Coupon applied successfully",
	}, nil
}

func (u *OrderUsecase) Checkout(ctx context.Context, userID string, req CheckoutReq) (*domain.Order, error) {
	// 1. Get Cart Items
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil || len(cart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}
	processItems := cart.Items
	cartID := cart.ID

	// 2. Calculate Total & Prepare Order Items & Check Pre-order
	var total float64
	var orderItems []domain.OrderItem
	var preOrderDepositRequired float64
	isPreOrderOrder := false

	// TODO: Get from Config/DB (Admin Setting)
	const PreOrderPercentage = 0.50

	for _, item := range processItems {
		product, err := u.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found", item.ProductID)
		}

		// Verify Variant & Pricing
		var price float64
		// Default to product price
		price = product.BasePrice
		if product.SalePrice != nil {
			price = *product.SalePrice
		}

		// Find relevant variant logic
		var targetVariantID string
		if item.VariantID != nil {
			targetVariantID = *item.VariantID
		}

		// L9 Fix: Iterate variants to find price override and validate ID
		foundVariant := false
		if len(product.Variants) > 0 {
			// If target is empty, use first/default
			if targetVariantID == "" {
				targetVariantID = product.Variants[0].ID
			}
			for _, v := range product.Variants {
				if v.ID == targetVariantID {
					foundVariant = true
					// Check for variant price override
					if v.Price != nil {
						price = *v.Price
					}
					// Check for variant sale price
					if v.SalePrice != nil {
						price = *v.SalePrice
					}
					// Update VariantID for checking stock status if variant had one (currently Product level)
					// Product.StockStatus is global for now.
					break
				}
			}
		} else {
			// No variants? Should not happen with SSOT backfill, but handle gracefully
			if targetVariantID == "" {
				return nil, fmt.Errorf("product %s has no inventory variants", product.Name)
			}
		}

		if !foundVariant && len(product.Variants) > 0 {
			return nil, fmt.Errorf("variant %s not found for product %s", targetVariantID, product.Name)
		}

		itemTotal := price * float64(item.Quantity)
		total += itemTotal

		// Pre-order Calculation
		if product.StockStatus == "pre_order" {
			isPreOrderOrder = true
			preOrderDepositRequired += itemTotal * PreOrderPercentage
		}

		// Use local variable for safe pointer
		variantIDPtr := &targetVariantID

		orderItems = append(orderItems, domain.OrderItem{
			ID:        utils.GenerateUUID(),
			ProductID: item.ProductID,
			VariantID: variantIDPtr,
			Quantity:  item.Quantity,
			Price:     price,
		})
	}

	// 3. (Coupon Logic Removed)

	// 4. Shipping Calculation
	deliveryLocation := "inside_dhaka"
	if loc, ok := req.Address["deliveryLocation"].(string); ok {
		deliveryLocation = loc
	}

	zone, zoneErr := u.configRepo.GetShippingZoneByKey(ctx, deliveryLocation)
	if zoneErr != nil {
		slog.Error("Usecase: Checkout - Shipping configuration not found", "location", deliveryLocation, "error", zoneErr)
		return nil, fmt.Errorf("shipping configuration for %s not found", deliveryLocation)
	}
	shippingFee := zone.Cost
	total += shippingFee

	// 5. Pre-Order Validation
	paymentDetails := domain.JSONB{}
	paymentStatus := "pending"
	paidAmount := 0.0

	if isPreOrderOrder && preOrderDepositRequired > 0 {
		if req.PaymentTrxID == "" || req.PaymentProvider == "" || req.PaymentPhone == "" {
			return nil, fmt.Errorf("pre-order items require partial payment info (TrxID, Provider, Phone)")
		}
		paymentStatus = "pending_verification"
		paidAmount = preOrderDepositRequired

		// Map details
		detailsMap := map[string]interface{}{
			"provider":         req.PaymentProvider,
			"transaction_id":   req.PaymentTrxID,
			"sender_number":    req.PaymentPhone,
			"deposit_required": preOrderDepositRequired,
			"shipping_fee":     shippingFee,
		}
		paymentDetails = domain.JSONB(detailsMap)
	}

	order := &domain.Order{
		ID:              utils.GenerateUUID(),
		UserID:          userID,
		Status:          "pending",
		TotalAmount:     total,
		ShippingFee:     shippingFee,
		ShippingAddress: req.Address,
		PaymentMethod:   req.Payment,
		PaymentStatus:   paymentStatus,
		PaidAmount:      paidAmount,
		IsPreOrder:      isPreOrderOrder,
		PaymentDetails:  paymentDetails,
		Items:           orderItems,
	}

	if isPreOrderOrder {
		order.Status = "pending_verification"
	}

	// 6. Transaction: Create Order, Update Stock, Increment Coupon, Clear Cart
	err = u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.CreateOrder(txCtx, order); err != nil {
			return err
		}

		// Update Stock
		for _, item := range order.Items {
			if item.VariantID == nil {
				return fmt.Errorf("item %s has no variant ID", item.ProductID)
			}
			if err := u.productRepo.UpdateStock(txCtx, *item.VariantID, -item.Quantity, "order_placed", order.ID); err != nil {
				return err
			}
		}

		// (Coupon Increment Logic Removed)

		// Clear Cart
		if cartID != "" {
			if err := u.orderRepo.ClearCart(txCtx, cartID); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// L9 Analytics: Send Purchase event to Facebook CAPI (async, non-blocking)
	if u.capiClient != nil {
		var contentItems []facebook.ContentItem
		for _, item := range order.Items {
			contentItems = append(contentItems, facebook.ContentItem{
				ID:       item.ProductID,
				Quantity: item.Quantity,
				Price:    item.Price,
			})
		}

		// Build UserData from order context for high IMQ (Identity Match Quality)
		userData := facebook.UserData{
			Country:    "BD",
			ClientIP:   req.IPAddress,
			UserAgent:  req.UserAgent,
			FBP:        req.FBP,
			FBC:        req.FBC,
			ExternalID: order.UserID,
		}
		// Extract PII and Location from shipping address
		if email, ok := order.ShippingAddress["email"].(string); ok && email != "" {
			userData.Email = email
		}
		if phone, ok := order.ShippingAddress["phone"].(string); ok && phone != "" {
			userData.Phone = phone
		}
		if fname, ok := order.ShippingAddress["firstName"].(string); ok && fname != "" {
			userData.FirstName = fname
		}
		if lname, ok := order.ShippingAddress["lastName"].(string); ok && lname != "" {
			userData.LastName = lname
		}
		if city, ok := order.ShippingAddress["district"].(string); ok && city != "" {
			userData.City = city
		}
		if state, ok := order.ShippingAddress["division"].(string); ok && state != "" {
			userData.State = state
		}
		if zip, ok := order.ShippingAddress["zip"].(string); ok && zip != "" {
			userData.Zip = zip
		}

		u.capiClient.SendPurchaseEvent(
			order.ID,
			order.TotalAmount,
			"BDT",
			contentItems,
			userData,
			order.ID, // Use order ID as event_id for deduplication
		)
	}

	return order, nil
}

func (u *OrderUsecase) GetMyOrders(ctx context.Context, userID string) ([]domain.Order, error) {
	return u.orderRepo.GetByUserID(ctx, userID)
}

// --- Admin Usecase ---

func (u *OrderUsecase) GetAllOrders(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, int64, error) {
	return u.orderRepo.GetAll(ctx, filter)
}

func (u *OrderUsecase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return u.orderRepo.GetByID(ctx, id)
}

func (u *OrderUsecase) UpdateOrderStatus(ctx context.Context, orderID, newStatus, note, actorID string) error {
	// 1. Get existing order
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	oldStatus := order.Status

	// Terminal Status Check - REMOVED for Admin flexibility
	// Admins need to be able to correct mistakes (e.g. accidentally marked as Fake/Cancelled).
	// if order.Status == domain.OrderStatusCancelled ||
	//    order.Status == domain.OrderStatusRefunded ||
	//    order.Status == domain.OrderStatusReturned ||
	//    order.Status == domain.OrderStatusFake {
	// 	return fmt.Errorf("cannot update order with terminal status: %s", order.Status)
	// }

	// 2. Validate Transition (L9 State Machine)
	if err := u.validateOrderTransition(order, newStatus); err != nil {
		return err
	}

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		// 3. Handle Side Effects (Stock, Payment, Analytics triggers)
		if err := u.handleOrderStateSideEffects(txCtx, order, newStatus, actorID); err != nil {
			return err
		}

		// 4. Update Status
		if err := u.orderRepo.UpdateStatus(txCtx, orderID, newStatus); err != nil {
			return err
		}

		// 5. Create History Entry
		// Determine Reason
		finalReason := note
		if finalReason == "" {
			finalReason = fmt.Sprintf("System: Status changed from %s to %s", oldStatus, newStatus)
		}

		var reasonPtr *string
		if finalReason != "" {
			reasonPtr = &finalReason
		}

		history := domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &oldStatus,
			NewStatus:      newStatus,
			Reason:         reasonPtr,
			CreatedBy:      &actorID,
		}
		if err := u.orderRepo.CreateOrderHistory(txCtx, &history); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
		}

		return nil
	})
}

// L9: Strict State Transition Rules
// L9: Simple Forward-Only Logic (Weight Based)
func (u *OrderUsecase) validateOrderTransition(order *domain.Order, newStatus string) error {
	// We define a "Progress Weight" for each state.
	// Users can jump forward (e.g. Pending -> Cancelled), but NEVER backward (e.g. Cancelled -> Pending).

	weights := map[string]int{
		domain.OrderStatusPending:             10,
		domain.OrderStatusPendingVerification: 10,
		domain.OrderStatusProcessing:          20,
		domain.OrderStatusShipped:             30,
		domain.OrderStatusDelivered:           40,
		domain.OrderStatusPaid:                50,
		domain.OrderStatusReturned:            60,
		domain.OrderStatusRefunded:            70,
		domain.OrderStatusCancelled:           80, // Terminal
		domain.OrderStatusFake:                90, // Terminal
	}

	currentWeight, okCurrent := weights[order.Status]
	newWeight, okNew := weights[newStatus]

	// If unknown status, allow update to fix data
	if !okCurrent || !okNew {
		return nil
	}

	if newWeight < currentWeight {
		return fmt.Errorf("invalid transition: cannot go backward from '%s' (%d) to '%s' (%d)", order.Status, currentWeight, newStatus, newWeight)
	}

	return nil
}

// L9: No Side Effects (Manual Management Mode)
func (u *OrderUsecase) handleOrderStateSideEffects(ctx context.Context, order *domain.Order, newStatus, actorID string) error {
	// User explicitly requested NO AUTOMATED STOCK OR PAYMENT UPDATES.
	// Operations will be handled manually in Inventory/Product management.
	// We only update the status field itself (handled by caller).
	return nil
}

// VerifyOrderPayment verifies a pre-order payment
func (u *OrderUsecase) VerifyOrderPayment(ctx context.Context, orderID, adminID string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if !order.IsPreOrder {
		return fmt.Errorf("order is not a pre-order")
	}
	if order.Status != "pending_verification" {
		return fmt.Errorf("order status is %s, cannot verify payment", order.Status)
	}

	oldStatus := order.Status

	// Atomic Update: Status -> processing, PaymentStatus -> partial_paid
	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdateStatus(txCtx, orderID, "processing"); err != nil {
			return err
		}
		if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, "partial_paid"); err != nil {
			return err
		}

		// Record History
		reason := "Payment Verified"
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &oldStatus,
			NewStatus:      "processing",
			Reason:         &reason,
			CreatedBy:      &adminID,
		}
		if err := u.orderRepo.CreateOrderHistory(txCtx, history); err != nil {
			return err
		}

		return nil
	})
}

// UpdatePaymentStatus updates the payment status of an order manually
func (u *OrderUsecase) UpdatePaymentStatus(ctx context.Context, orderID, newStatus, actorID string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	oldPaymentStatus := order.PaymentStatus
	if oldPaymentStatus == newStatus {
		return nil
	}

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, newStatus); err != nil {
			return err
		}

		// Record History
		reason := fmt.Sprintf("Payment status changed: %s -> %s", oldPaymentStatus, newStatus)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status, // Status didn't change
			NewStatus:      order.Status,
			Reason:         &reason,
			CreatedBy:      &actorID,
		}
		if err := u.orderRepo.CreateOrderHistory(txCtx, history); err != nil {
			return err
		}
		return nil
	})
}

// ProcessRefund handles the refund logic
func (u *OrderUsecase) ProcessRefund(ctx context.Context, orderID string, amount float64, reason string, restock bool, adminID string) error {
	// 1. Get Order
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Validate Refund
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive")
	}
	remainingRefundable := order.PaidAmount - order.RefundedAmount
	if amount > remainingRefundable {
		return fmt.Errorf("cannot refund %.2f (max refundable: %.2f)", amount, remainingRefundable)
	}

	// Determine if status should change (e.g. if full refund -> refunded)
	// For now, partial refund doesn't change order status usually, but full refund might.
	// L9: Let's assume explicit status change is handled separately via UpdateStatus,
	// OR if restock is true, maybe we should mark as refunded?
	// Current logic just updates refunded amount.
	// Ensure we log this action.

	// 3. Execute Transaction
	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		// Create Refund & Update Order
		if err := u.orderRepo.CreateRefund(txCtx, orderID, amount, reason, restock, &adminID); err != nil {
			return err
		}

		// Handle Stock Restoration
		if restock {
			for _, item := range order.Items {
				targetID := item.ProductID
				if item.VariantID != nil {
					targetID = *item.VariantID
				}
				// Restock
				if err := u.productRepo.UpdateStock(txCtx, targetID, item.Quantity, "refund_restock", orderID); err != nil {
					return fmt.Errorf("failed to restock item %s: %v", targetID, err)
				}
			}
		}

		// Record History (Log the refund action)
		// Previous status is same as current if we don't change it.
		// We log "refunded" action but status might remain "delivered" or "cancelled".
		// Let's log it as a status update if we change status, but here we are just refunding money.
		// But the audit log is `order_history` which tracks status.
		// Maybe we just log a "note" entry? The schema requires new_status.
		// Let's use current status as new_status but add reason "Refunded X Amount".

		histReason := fmt.Sprintf("Refunded %.2f: %s", amount, reason)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status,
			NewStatus:      order.Status, // Status didn't change unless we force it
			Reason:         &histReason,
			CreatedBy:      &adminID,
		}
		// If it was a full refund and restock, maybe auto-set to refunded?
		// User asked for robust, so automation is good.
		if restock && amount >= remainingRefundable {
			history.NewStatus = domain.OrderStatusRefunded
			if err := u.orderRepo.UpdateStatus(txCtx, orderID, domain.OrderStatusRefunded); err != nil {
				return err
			}
		}

		if err := u.orderRepo.CreateOrderHistory(txCtx, history); err != nil {
			return err
		}

		return nil
	})
}

// GetOrderHistory retrieves the history logs for an order
func (u *OrderUsecase) GetOrderHistory(ctx context.Context, orderID string) ([]domain.OrderHistory, error) {
	return u.orderRepo.GetOrderHistory(ctx, orderID)
}
