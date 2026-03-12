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

	// 2. Calculate Total & Prepare Order Items & Determine Payment Policy
	var total float64
	var orderItems []domain.OrderItem

	// Track total deposit for pre-orders
	var isPreorder bool
	var totalDepositRequired float64

	for _, item := range processItems {
		product, err := u.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found", item.ProductID)
		}

		if product.IsPreorder {
			isPreorder = true
			totalDepositRequired += product.PreorderDepositAmount * float64(item.Quantity)
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
					break
				}
			}
		} else {
			if targetVariantID == "" {
				return nil, fmt.Errorf("product %s has no inventory variants", product.Name)
			}
		}

		if !foundVariant && len(product.Variants) > 0 {
			return nil, fmt.Errorf("variant %s not found for product %s", targetVariantID, product.Name)
		}

		itemTotal := price * float64(item.Quantity)
		total += itemTotal

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

	// 3. Shipping Calculation
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

	// 4. Payment Policy Enforcement
	paymentDetails := domain.JSONB{}
	paidAmount := 0.0

	var requiredDeposit float64
	if isPreorder {
		requiredDeposit = totalDepositRequired
		if requiredDeposit > 0 {
			// L9: Require payment info for any non-zero deposit
			if req.PaymentTrxID == "" || req.PaymentProvider == "" || req.PaymentPhone == "" {
				return nil, fmt.Errorf("Pre-order requires payment info (TrxID, Provider, Phone) — deposit: %.2f BDT", requiredDeposit)
			}
			paidAmount = requiredDeposit

			paymentDetails = domain.JSONB{
				"provider":         req.PaymentProvider,
				"transaction_id":   req.PaymentTrxID,
				"sender_number":    req.PaymentPhone,
				"deposit_required": requiredDeposit,
				"shipping_fee":     shippingFee,
			}
		}
	} else if req.Payment == domain.PaymentMethodAdvance {
		// L9: Advance payment — require trx info
		paidAmount = total

		paymentDetails = domain.JSONB{
			"provider":       req.PaymentProvider,
			"transaction_id": req.PaymentTrxID,
			"sender_number":  req.PaymentPhone,
			"shipping_fee":   shippingFee,
		}
	}

	// 5. Build Initial State based on Payment Method & Pre-order
	order := &domain.Order{
		ID:              utils.GenerateUUID(),
		UserID:          userID,
		TotalAmount:     total,
		ShippingFee:     shippingFee,
		ShippingAddress: req.Address,
		PaymentMethod:   req.Payment,
		PaidAmount:      paidAmount,
		IsPreorder:      isPreorder,
		PaymentDetails:  paymentDetails,
		Items:           orderItems,
	}

	// L9: Centralized Initial Status (Single Source of Truth from constants.go)
	initialOrderStatus, initialPaymentStatus := domain.DetermineInitialStatus(
		req.Payment, isPreorder, requiredDeposit, req.PaymentTrxID != "",
	)
	order.Status = initialOrderStatus
	order.PaymentStatus = initialPaymentStatus

	// 6. Transaction: Create Order, Update Stock, Increment Coupon, Clear Cart
	err = u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.CreateOrder(txCtx, order); err != nil {
			return err
		}

		// 6a. Lock and check stock for each item (L9 Pessimistic Locking)
		for _, item := range order.Items {
			if item.VariantID == nil {
				return fmt.Errorf("item %s has no variant ID", item.ProductID)
			}
			// Lock row for update to prevent race conditions
			variant, err := u.productRepo.GetVariantByIDForUpdate(txCtx, *item.VariantID)
			if err != nil {
				return fmt.Errorf("failed to lock stock for variant %s: %v", *item.VariantID, err)
			}
			if variant.Stock < item.Quantity {
				return fmt.Errorf("insufficient stock for item %s (requested: %d, available: %d)", variant.Name, item.Quantity, variant.Stock)
			}

			// Update Stock
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
		} else if zip, ok := order.ShippingAddress["postalCode"].(string); ok && zip != "" {
			userData.Zip = zip
		}
		// Use thana for more granular city matching if available
		if thana, ok := order.ShippingAddress["thana"].(string); ok && thana != "" {
			userData.City = thana
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

// L9: Strict State Transition Validation
// Uses IsValidTransition from constants.go — the single source of truth.
// Unknown statuses are REJECTED, not silently passed.
func (u *OrderUsecase) validateOrderTransition(order *domain.Order, newStatus string) error {
	if !domain.IsValidTransition(order.Status, newStatus) {
		return fmt.Errorf("forbidden transition: cannot move order from '%s' to '%s'", order.Status, newStatus)
	}
	return nil
}

// L9: Declarative Side-Effect Engine
// Reads from GetSideEffects() in constants.go — no inline business rules.
func (u *OrderUsecase) handleOrderStateSideEffects(ctx context.Context, order *domain.Order, newStatus, actorID string) error {
	effects := domain.GetSideEffects(order.Status, newStatus)
	if len(effects) == 0 {
		return nil
	}

	for _, effect := range effects {
		switch effect {

		case domain.SideEffectRestoreStock:
			// Restore all order items back to inventory
			slog.Info("L9 Side-Effect: Restoring stock", "order_id", order.ID, "trigger", newStatus)
			if err := u.restoreOrderStock(ctx, order, "auto_restore_"+newStatus); err != nil {
				return err
			}

		case domain.SideEffectDeductStock:
			// Recovery path: re-deduct stock (cancelled/fake → processing)
			slog.Info("L9 Side-Effect: Re-deducting stock (recovery)", "order_id", order.ID)
			if err := u.deductOrderStock(ctx, order, "recovery_rededuct"); err != nil {
				return err
			}

		case domain.SideEffectSyncPaymentPaid:
			slog.Info("L9 Side-Effect: Syncing payment → paid", "order_id", order.ID)
			if err := u.orderRepo.UpdatePaymentStatus(ctx, order.ID, domain.PaymentStatusPaid); err != nil {
				return fmt.Errorf("failed to sync payment status to paid: %w", err)
			}
			if err := u.orderRepo.UpdatePaidAmount(ctx, order.ID, order.TotalAmount); err != nil {
				return fmt.Errorf("failed to sync paid amount: %w", err)
			}

		case domain.SideEffectSyncPaymentRefund:
			slog.Info("L9 Side-Effect: Syncing payment → refunded", "order_id", order.ID)
			if err := u.orderRepo.UpdatePaymentStatus(ctx, order.ID, domain.PaymentStatusRefunded); err != nil {
				return fmt.Errorf("failed to sync payment status to refunded: %w", err)
			}
		}
	}
	return nil
}

// restoreOrderStock adds stock back to inventory for all items in an order.
func (u *OrderUsecase) restoreOrderStock(ctx context.Context, order *domain.Order, reason string) error {
	for _, item := range order.Items {
		targetID := item.ProductID
		if item.VariantID != nil {
			targetID = *item.VariantID
		}
		if err := u.productRepo.UpdateStock(ctx, targetID, item.Quantity, reason, order.ID); err != nil {
			return fmt.Errorf("failed to restore stock for item %s: %w", targetID, err)
		}
	}
	return nil
}

// deductOrderStock removes stock from inventory for all items in an order (recovery path).
func (u *OrderUsecase) deductOrderStock(ctx context.Context, order *domain.Order, reason string) error {
	for _, item := range order.Items {
		if item.VariantID == nil {
			return fmt.Errorf("item %s has no variant ID, cannot re-deduct stock", item.ProductID)
		}
		if err := u.productRepo.UpdateStock(ctx, *item.VariantID, -item.Quantity, reason, order.ID); err != nil {
			return fmt.Errorf("failed to re-deduct stock for variant %s: %w", *item.VariantID, err)
		}
	}
	return nil
}

// VerifyOrderPayment verifies an advance/pre-order payment.
// L9: Uses constants, validates both order and payment FSM transitions.
func (u *OrderUsecase) VerifyOrderPayment(ctx context.Context, orderID, adminID string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// L9: Reject COD orders
	if order.PaymentMethod == domain.PaymentMethodCOD {
		return fmt.Errorf("order is COD — no advance payment to verify")
	}

	// L9: Validate order status transition
	if !domain.IsValidTransition(order.Status, domain.OrderStatusProcessing) {
		return fmt.Errorf("cannot verify payment: order status is '%s', expected '%s'",
			order.Status, domain.OrderStatusPendingVerification)
	}

	// L9: Validate payment status transition
	newPaymentStatus := domain.PaymentStatusPartialPaid
	if !order.IsPreorder {
		// Full advance payment → mark as paid
		newPaymentStatus = domain.PaymentStatusPaid
	}
	if !domain.IsValidPaymentTransition(order.PaymentStatus, newPaymentStatus) {
		return fmt.Errorf("cannot verify payment: payment status transition '%s' → '%s' is forbidden",
			order.PaymentStatus, newPaymentStatus)
	}

	oldStatus := order.Status

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdateStatus(txCtx, orderID, domain.OrderStatusProcessing); err != nil {
			return err
		}
		if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, newPaymentStatus); err != nil {
			return err
		}

		reason := fmt.Sprintf("Payment verified by admin. Payment: %s → %s", order.PaymentStatus, newPaymentStatus)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &oldStatus,
			NewStatus:      domain.OrderStatusProcessing,
			Reason:         &reason,
			CreatedBy:      &adminID,
		}
		return u.orderRepo.CreateOrderHistory(txCtx, history)
	})
}

// UpdatePaymentStatus updates the payment status of an order manually.
// L9: Enforces ValidPaymentTransitions FSM — rejects invalid changes.
func (u *OrderUsecase) UpdatePaymentStatus(ctx context.Context, orderID, newStatus, actorID string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	oldPaymentStatus := order.PaymentStatus
	if oldPaymentStatus == newStatus {
		return nil // Idempotent: no-op if already at target status
	}

	// L9: Enforce Payment FSM
	if !domain.IsValidPaymentTransition(oldPaymentStatus, newStatus) {
		return fmt.Errorf("forbidden payment transition: cannot change payment from '%s' to '%s'",
			oldPaymentStatus, newStatus)
	}

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, newStatus); err != nil {
			return err
		}

		reason := fmt.Sprintf("Payment status: %s → %s", oldPaymentStatus, newStatus)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status,
			NewStatus:      order.Status,
			Reason:         &reason,
			CreatedBy:      &actorID,
		}
		return u.orderRepo.CreateOrderHistory(txCtx, history)
	})
}

// ProcessRefund handles the refund logic.
// L9: Validates FSM before auto-transitioning, uses shared stock helpers.
func (u *OrderUsecase) ProcessRefund(ctx context.Context, orderID string, amount float64, reason string, restock bool, adminID string) error {
	// 1. Get Order
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Validate Refund Amount
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive")
	}
	remainingRefundable := order.PaidAmount - order.RefundedAmount
	if amount > remainingRefundable {
		return fmt.Errorf("cannot refund %.2f (max refundable: %.2f)", amount, remainingRefundable)
	}

	// 3. Determine if this is a full refund (triggers status change)
	isFullRefund := restock && amount >= remainingRefundable

	// L9: If full refund, validate FSM allows the transition BEFORE entering the transaction
	if isFullRefund {
		if !domain.IsValidTransition(order.Status, domain.OrderStatusRefunded) {
			return fmt.Errorf("cannot auto-refund: transition from '%s' to 'refunded' is forbidden by FSM", order.Status)
		}
	}

	// 4. Execute Transaction
	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		// Create Refund record
		if err := u.orderRepo.CreateRefund(txCtx, orderID, amount, reason, restock, &adminID); err != nil {
			return err
		}

		// Restock if requested (using shared helper)
		if restock {
			if err := u.restoreOrderStock(txCtx, order, "refund_restock"); err != nil {
				return err
			}
		}

		// Build history entry
		histReason := fmt.Sprintf("Refunded %.2f BDT: %s", amount, reason)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status,
			NewStatus:      order.Status,
			Reason:         &histReason,
			CreatedBy:      &adminID,
		}

		// L9: Full refund + restock → auto-transition to refunded (FSM already validated above)
		if isFullRefund {
			history.NewStatus = domain.OrderStatusRefunded
			if err := u.orderRepo.UpdateStatus(txCtx, orderID, domain.OrderStatusRefunded); err != nil {
				return err
			}
			// Sync payment status via side-effect engine
			if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, domain.PaymentStatusRefunded); err != nil {
				return err
			}
		}

		return u.orderRepo.CreateOrderHistory(txCtx, history)
	})
}

// GetOrderHistory retrieves the history logs for an order
func (u *OrderUsecase) GetOrderHistory(ctx context.Context, orderID string) ([]domain.OrderHistory, error) {
	return u.orderRepo.GetOrderHistory(ctx, orderID)
}

// UpdateShippingZone updates the deliveryLocation inside the shipping_address JSON and recalculates shipping fee
func (u *OrderUsecase) UpdateShippingZone(ctx context.Context, orderID string, newZoneKey string, adminID string) error {
	// 1. Fetch current order
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Fetch new zone cost
	zone, zoneErr := u.configRepo.GetShippingZoneByKey(ctx, newZoneKey)
	if zoneErr != nil {
		return fmt.Errorf("shipping zone %s not found: %w", newZoneKey, zoneErr)
	}

	// 3. Skip if zone is unchanged
	currentZone := ""
	if loc, ok := order.ShippingAddress["deliveryLocation"].(string); ok {
		currentZone = loc
	}

	if currentZone == newZoneKey {
		return nil // No change needed
	}

	// 4. Calculate Financial Impact
	oldShippingFee := order.ShippingFee
	newShippingFee := zone.Cost

	diff := newShippingFee - oldShippingFee
	newTotal := order.TotalAmount + diff

	// 5. Update Address JSON
	if order.ShippingAddress == nil {
		order.ShippingAddress = domain.JSONB{}
	}
	order.ShippingAddress["deliveryLocation"] = newZoneKey

	// 6. DB Update in Transaction
	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		err := u.orderRepo.UpdateOrderShippingDetails(txCtx, orderID, order.ShippingAddress, newShippingFee, newTotal)
		if err != nil {
			return err
		}

		// Build history entry
		histReason := fmt.Sprintf("Shipping zone changed from %s to %s (Fee difference: %.2f)", currentZone, newZoneKey, diff)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status,
			NewStatus:      order.Status,
			Reason:         &histReason,
			CreatedBy:      &adminID,
		}
		return u.orderRepo.CreateOrderHistory(txCtx, history)
	})
}
