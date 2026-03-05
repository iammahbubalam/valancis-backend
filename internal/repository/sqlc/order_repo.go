package sqlcrepo

import (
	"context"
	"encoding/json"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewOrderRepository(db *pgxpool.Pool) domain.OrderRepository {
	return &orderRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// --- Mappers ---

func sqlcCartToDomain(c sqlc.Cart, items []sqlc.GetCartItemsRow) *domain.Cart {
	cart := &domain.Cart{
		ID:        uuidToString(c.ID),
		CreatedAt: pgtimeToTime(c.CreatedAt),
		UpdatedAt: pgtimeToTime(c.UpdatedAt),
	}
	if c.UserID.Valid {
		uid := uuidToString(c.UserID)
		cart.UserID = &uid
	}

	cart.Items = make([]domain.CartItem, len(items))
	for i, item := range items {
		cart.Items[i] = domain.CartItem{
			ID:        uuidToString(item.ID),
			CartID:    uuidToString(item.CartID),
			ProductID: uuidToString(item.ProductID),
			Quantity:  int(item.Quantity),
			Product: domain.Product{
				ID:                    uuidToString(item.ProductID),
				Name:                  item.Name,
				Slug:                  item.Slug,
				BasePrice:             numericToFloat64(item.BasePrice),
				SalePrice:             numericToFloat64Ptr(item.SalePrice),
				IsPreorder:            item.IsPreorder,
				PreorderDepositAmount: numericToFloat64(item.PreorderDepositAmount),
			},
		}
		if item.VariantID.Valid {
			vid := uuidToString(item.VariantID)
			cart.Items[i].VariantID = &vid
		}
		// Parse media for images
		if len(item.Media) > 0 {
			cart.Items[i].Product.Media = domain.RawJSON(item.Media)
			mapMediaToImages(&cart.Items[i].Product)
		}
	}
	return cart
}

func sqlcOrderToDomain(o sqlc.Order, items []sqlc.GetOrderItemsRow) *domain.Order {
	order := &domain.Order{
		ID:             uuidToString(o.ID),
		UserID:         uuidToString(o.UserID),
		Status:         o.Status,
		TotalAmount:    numericToFloat64(o.TotalAmount),
		PaymentMethod:  ptrString(o.PaymentMethod),
		PaymentStatus:  ptrString(o.PaymentStatus),
		PaidAmount:     numericToFloat64(o.PaidAmount),
		RefundedAmount: numericToFloat64(o.RefundedAmount),
		ShippingFee:    numericToFloat64(o.ShippingFee),
		IsPreorder:     o.IsPreorder,
		CreatedAt:      pgtimeToTime(o.CreatedAt),
		UpdatedAt:      pgtimeToTime(o.UpdatedAt),
	}

	if len(o.PaymentDetails) > 0 {
		var details domain.JSONB
		json.Unmarshal(o.PaymentDetails, &details)
		order.PaymentDetails = details
	}

	// Parse shipping address
	if len(o.ShippingAddress) > 0 {
		var addr domain.JSONB
		json.Unmarshal(o.ShippingAddress, &addr)
		order.ShippingAddress = addr
	}

	// Parse payment details
	if len(o.PaymentDetails) > 0 {
		var details domain.JSONB
		json.Unmarshal(o.PaymentDetails, &details)
		order.PaymentDetails = details
	}

	order.Items = make([]domain.OrderItem, len(items))
	for i, item := range items {
		order.Items[i] = domain.OrderItem{
			ID:          uuidToString(item.ID),
			OrderID:     uuidToString(item.OrderID),
			ProductID:   uuidToString(item.ProductID),
			Quantity:    int(item.Quantity),
			Price:       numericToFloat64(item.Price),
			VariantName: item.VariantName,
			VariantSKU:  item.VariantSku,
			Product: domain.Product{
				Name: item.Name,
				Slug: item.Slug,
			},
		}
		if item.VariantID.Valid {
			vid := uuidToString(item.VariantID)
			order.Items[i].VariantID = &vid
		}
		if len(item.Media) > 0 {
			order.Items[i].Product.Media = domain.RawJSON(item.Media)
			mapMediaToImages(&order.Items[i].Product)
		}
	}
	return order
}

// ...

// --- Cart Methods ---

func (r *orderRepository) GetCartByUserID(ctx context.Context, userID string) (*domain.Cart, error) {
	cart, err := r.queries.GetCartByUserID(ctx, stringToUUID(userID))
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	items, err := r.queries.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}

	return sqlcCartToDomain(cart, items), nil
}

func (r *orderRepository) CreateCart(ctx context.Context, cart *domain.Cart) error {
	var userID pgtype.UUID
	if cart.UserID != nil {
		userID = stringToUUID(*cart.UserID)
	}

	created, err := r.queries.CreateCart(ctx, userID)
	if err != nil {
		return err
	}
	cart.ID = uuidToString(created.ID)
	cart.CreatedAt = pgtimeToTime(created.CreatedAt)
	cart.UpdatedAt = pgtimeToTime(created.UpdatedAt)
	return nil
}

func (r *orderRepository) GetCartWithItems(ctx context.Context, userID string) ([]domain.CartItem, error) {
	rows, err := r.queries.GetCartWithItems(ctx, stringToUUID(userID))
	if err != nil {
		return nil, err
	}

	items := make([]domain.CartItem, 0, len(rows))
	for _, row := range rows {
		// Skip rows where item_id is null (empty cart)
		if !row.ItemID.Valid {
			continue
		}

		item := domain.CartItem{
			ID:        uuidToString(row.ItemID),
			CartID:    uuidToString(row.CartID),
			ProductID: uuidToString(row.ProductID),
			Quantity:  int(*row.Quantity),
			Product: domain.Product{
				ID:                    uuidToString(row.ProductID),
				Name:                  *row.Name,
				Slug:                  *row.Slug,
				BasePrice:             numericToFloat64(row.BasePrice),
				SalePrice:             numericToFloat64Ptr(row.SalePrice),
				IsPreorder:            row.IsPreorder != nil && *row.IsPreorder,
				PreorderDepositAmount: numericToFloat64(row.PreorderDepositAmount),
			},
		}

		if row.Stock != nil {
			item.Product.Stock = int(*row.Stock)
		}
		if row.StockStatus != nil {
			item.Product.StockStatus = *row.StockStatus
		}

		if row.VariantID.Valid {
			vid := uuidToString(row.VariantID)
			item.VariantID = &vid
			item.VariantName = row.VariantName
			if len(row.VariantImages) > 0 {
				item.VariantImage = &row.VariantImages[0]
			}
		}

		// RESOLVE Effective Price (Variant Price > Product Base)
		item.Price = numericToFloat64(row.BasePrice)
		if row.VariantPrice.Valid {
			item.Price = numericToFloat64(row.VariantPrice)
		}

		// RESOLVE Effective Sale Price (Variant Sale > Product Sale)
		if row.VariantSalePrice.Valid {
			s := numericToFloat64(row.VariantSalePrice)
			item.SalePrice = &s
		} else if row.SalePrice.Valid {
			s := numericToFloat64(row.SalePrice)
			item.SalePrice = &s
		}

		if len(row.Media) > 0 {
			item.Product.Media = domain.RawJSON(row.Media)
			mapMediaToImages(&item.Product)
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *orderRepository) UpsertCartItemAtomic(ctx context.Context, userID, cartID, productID string, variantID *string, quantity int) ([]domain.CartItem, error) {
	var variantUUID pgtype.UUID
	if variantID != nil {
		variantUUID = stringToUUID(*variantID)
	}

	rows, err := r.queries.UpsertCartItemAtomic(ctx, sqlc.UpsertCartItemAtomicParams{
		CartID:    stringToUUID(cartID),
		UserID:    stringToUUID(userID),
		ProductID: stringToUUID(productID),
		VariantID: variantUUID,
		Quantity:  int32(quantity),
	})
	if err != nil {
		return nil, err
	}

	// Map rows to CartItems
	items := make([]domain.CartItem, len(rows))
	for i, row := range rows {
		items[i] = domain.CartItem{
			ID:        uuidToString(row.ID),
			CartID:    uuidToString(row.CartID),
			ProductID: uuidToString(row.ProductID),
			Quantity:  int(row.Quantity),
			Product: domain.Product{
				ID:                    uuidToString(row.ProductID),
				Name:                  row.Name,
				Slug:                  row.Slug,
				BasePrice:             numericToFloat64(row.BasePrice),
				SalePrice:             numericToFloat64Ptr(row.SalePrice),
				IsPreorder:            row.IsPreorder,
				PreorderDepositAmount: numericToFloat64(row.PreorderDepositAmount),
			},
		}
		if row.VariantID.Valid {
			vid := uuidToString(row.VariantID)
			items[i].VariantID = &vid
			items[i].VariantName = &row.VariantName
			if len(row.VariantImages) > 0 {
				items[i].VariantImage = &row.VariantImages[0]
			}
		}

		// RESOLVE Effective Price (Variant Price > Product Base)
		items[i].Price = numericToFloat64(row.BasePrice)
		if row.VariantPrice.Valid {
			items[i].Price = numericToFloat64(row.VariantPrice)
		}

		// RESOLVE Effective Sale Price (Variant Sale > Product Sale)
		if row.VariantSalePrice.Valid {
			s := numericToFloat64(row.VariantSalePrice)
			items[i].SalePrice = &s
		} else if row.SalePrice.Valid {
			s := numericToFloat64(row.SalePrice)
			items[i].SalePrice = &s
		}

		if len(row.Media) > 0 {
			items[i].Product.Media = domain.RawJSON(row.Media)
			mapMediaToImages(&items[i].Product)
		}
	}

	return items, nil
}

func (r *orderRepository) AtomicRemoveCartItem(ctx context.Context, userID, productID, variantID string) error {
	return r.queries.AtomicRemoveCartItem(ctx, sqlc.AtomicRemoveCartItemParams{
		UserID:    stringToUUID(userID),
		ProductID: stringToUUID(productID),
		VariantID: stringToUUID(variantID),
	})
}

func (r *orderRepository) ClearCart(ctx context.Context, cartID string) error {
	return r.queries.ClearCart(ctx, stringToUUID(cartID))
}

// --- Order Methods ---

func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	shippingAddrBytes, _ := json.Marshal(order.ShippingAddress)
	paymentDetailsBytes, _ := json.Marshal(order.PaymentDetails)

	created, err := r.queries.CreateOrder(ctx, sqlc.CreateOrderParams{
		UserID:          stringToUUID(order.UserID),
		Status:          order.Status,
		TotalAmount:     float64ToNumeric(order.TotalAmount),
		ShippingFee:     float64ToNumeric(order.ShippingFee),
		ShippingAddress: shippingAddrBytes,
		PaymentMethod:   strPtr(order.PaymentMethod),
		PaymentStatus:   strPtr(order.PaymentStatus),
		PaidAmount:      float64ToNumeric(order.PaidAmount),
		PaymentDetails:  paymentDetailsBytes,
		IsPreorder:      order.IsPreorder,
	})
	if err != nil {
		return err
	}

	order.ID = uuidToString(created.ID)
	order.CreatedAt = pgtimeToTime(created.CreatedAt)
	order.UpdatedAt = pgtimeToTime(created.UpdatedAt)

	// Create order items
	for i := range order.Items {
		item := &order.Items[i]
		var variantID pgtype.UUID
		if item.VariantID != nil {
			variantID = stringToUUID(*item.VariantID)
		}

		createdItem, err := r.queries.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			OrderID:   created.ID,
			ProductID: stringToUUID(item.ProductID),
			VariantID: variantID,
			Quantity:  int32(item.Quantity),
			Price:     float64ToNumeric(item.Price),
		})
		if err != nil {
			return err
		}
		item.ID = uuidToString(createdItem.ID)
		item.OrderID = order.ID
	}

	return nil
}

// ...

func (r *orderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	row, err := r.queries.GetOrderByID(ctx, stringToUUID(id))
	if err != nil {
		return nil, err
	}

	items, err := r.queries.GetOrderItems(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	// Map row to sqlc.Order for the shared mapper
	o := sqlc.Order{
		ID:              row.ID,
		UserID:          row.UserID,
		Status:          row.Status,
		TotalAmount:     row.TotalAmount,
		ShippingAddress: row.ShippingAddress,
		PaymentMethod:   row.PaymentMethod,
		PaymentStatus:   row.PaymentStatus,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		PaidAmount:      row.PaidAmount,
		PaymentDetails:  row.PaymentDetails,
		IsPreorder:      row.IsPreorder,
		RefundedAmount:  row.RefundedAmount,
		ShippingFee:     row.ShippingFee,
	}

	order := sqlcOrderToDomain(o, items)

	// Enrich with User info from the JOIN
	order.User = domain.User{
		Email:     row.Email,
		FirstName: ptrString(row.FirstName),
		LastName:  ptrString(row.LastName),
		Avatar:    ptrString(row.Avatar),
	}

	return order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Order, error) {
	orders, err := r.queries.GetOrdersByUserID(ctx, stringToUUID(userID))
	if err != nil {
		return nil, err
	}

	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		items, _ := r.queries.GetOrderItems(ctx, o.ID)
		order := sqlcOrderToDomain(o, items)
		result[i] = *order
	}
	return result, nil
}

// --- Admin Methods ---

func (r *orderRepository) GetAll(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var status *string
	if filter.Status != "" {
		status = &filter.Status
	}
	var paymentStatus *string
	if filter.PaymentStatus != "" {
		paymentStatus = &filter.PaymentStatus
	}

	var search *string
	if filter.Search != "" {
		search = &filter.Search
	}

	orders, err := r.queries.GetAllOrders(ctx, sqlc.GetAllOrdersParams{
		Status:        status,
		PaymentStatus: paymentStatus,
		IsPreorder:    filter.IsPreorder,
		Search:        search,
		Limit:         int32(limit),
		Offset:        int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountOrders(ctx, sqlc.CountOrdersParams{
		Status:        status,
		PaymentStatus: paymentStatus,
		IsPreorder:    filter.IsPreorder,
		Search:        search,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		result[i] = domain.Order{
			ID:             uuidToString(o.ID),
			UserID:         uuidToString(o.UserID),
			Status:         o.Status,
			TotalAmount:    numericToFloat64(o.TotalAmount),
			PaymentMethod:  ptrString(o.PaymentMethod),
			PaymentStatus:  ptrString(o.PaymentStatus),
			PaidAmount:     numericToFloat64(o.PaidAmount),
			RefundedAmount: numericToFloat64(o.RefundedAmount),
			ShippingFee:    numericToFloat64(o.ShippingFee),
			IsPreorder:     o.IsPreorder,
			CreatedAt:      pgtimeToTime(o.CreatedAt),
			UpdatedAt:      pgtimeToTime(o.UpdatedAt),
			User: domain.User{
				Email:     o.Email,
				FirstName: ptrString(o.FirstName),
				LastName:  ptrString(o.LastName),
				Avatar:    ptrString(o.Avatar),
			},
		}
		if len(o.PaymentDetails) > 0 {
			var details domain.JSONB
			json.Unmarshal(o.PaymentDetails, &details)
			result[i].PaymentDetails = details
		}
		if len(o.ShippingAddress) > 0 {
			var addr domain.JSONB
			json.Unmarshal(o.ShippingAddress, &addr)
			result[i].ShippingAddress = addr
		}

		// Fetch items for this order
		items, _ := r.queries.GetOrderItems(ctx, o.ID)
		domainItems := make([]domain.OrderItem, len(items))
		for j, item := range items {
			domainItems[j] = domain.OrderItem{
				ID:          uuidToString(item.ID),
				OrderID:     uuidToString(item.OrderID),
				ProductID:   uuidToString(item.ProductID),
				Quantity:    int(item.Quantity),
				Price:       numericToFloat64(item.Price),
				VariantName: item.VariantName,
				VariantSKU:  item.VariantSku,
				Product: domain.Product{
					Name: item.Name,
					Slug: item.Slug,
				},
			}
			if item.VariantID.Valid {
				vid := uuidToString(item.VariantID)
				domainItems[j].VariantID = &vid
			}
			if len(item.Media) > 0 {
				domainItems[j].Product.Media = domain.RawJSON(item.Media)
				mapMediaToImages(&domainItems[j].Product)
			}
		}
		result[i].Items = domainItems
	}

	return result, count, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.queries.UpdateOrderStatus(ctx, sqlc.UpdateOrderStatusParams{
		ID:     stringToUUID(id),
		Status: status,
	})
}

func (r *orderRepository) UpdatePaymentStatus(ctx context.Context, id, status string) error {
	return r.queries.UpdateOrderPaymentStatus(ctx, sqlc.UpdateOrderPaymentStatusParams{
		ID:            stringToUUID(id),
		PaymentStatus: strPtr(status),
	})
}

func (r *orderRepository) UpdatePaidAmount(ctx context.Context, id string, amount float64) error {
	return r.queries.UpdateOrderPaidAmount(ctx, sqlc.UpdateOrderPaidAmountParams{
		ID:         stringToUUID(id),
		PaidAmount: float64ToNumeric(amount),
	})
}

func (r *orderRepository) HasPurchasedProduct(ctx context.Context, userID, productID string) (bool, error) {
	return r.queries.HasPurchasedProduct(ctx, sqlc.HasPurchasedProductParams{
		UserID:    stringToUUID(userID),
		ProductID: stringToUUID(productID),
	})
}

func (r *orderRepository) CreateRefund(ctx context.Context, orderID string, amount float64, reason string, restock bool, createdBy *string) error {
	var createdByUUID pgtype.UUID
	if createdBy != nil {
		createdByUUID = stringToUUID(*createdBy)
	}

	// 1. Create Refund Record
	_, err := r.queries.CreateRefund(ctx, sqlc.CreateRefundParams{
		OrderID:      stringToUUID(orderID),
		Amount:       float64ToNumeric(amount),
		Reason:       strPtr(reason),
		RestockItems: restock,
		CreatedBy:    createdByUUID,
	})
	if err != nil {
		return err
	}

	// 2. Update Order Totals
	return r.queries.UpdateOrderRefundedAmount(ctx, sqlc.UpdateOrderRefundedAmountParams{
		Amount: float64ToNumeric(amount),
		ID:     stringToUUID(orderID),
	})
}

func (r *orderRepository) CreateOrderHistory(ctx context.Context, history *domain.OrderHistory) error {
	var createdBy pgtype.UUID
	if history.CreatedBy != nil {
		createdBy = stringToUUID(*history.CreatedBy)
	}

	_, err := r.queries.CreateOrderHistory(ctx, sqlc.CreateOrderHistoryParams{
		OrderID:        stringToUUID(history.OrderID),
		PreviousStatus: history.PreviousStatus,
		NewStatus:      history.NewStatus,
		Reason:         history.Reason,
		CreatedBy:      createdBy,
	})
	return err
}

func (r *orderRepository) GetOrderHistory(ctx context.Context, orderID string) ([]domain.OrderHistory, error) {
	rows, err := r.queries.GetOrderHistory(ctx, stringToUUID(orderID))
	if err != nil {
		return nil, err
	}

	var history []domain.OrderHistory
	for _, row := range rows {
		h := domain.OrderHistory{
			ID:        uuidToString(row.ID),
			OrderID:   uuidToString(row.OrderID),
			NewStatus: row.NewStatus,
			CreatedAt: row.CreatedAt.Time, // pgtype.Timestamptz has Time field
		}
		if row.PreviousStatus != nil {
			h.PreviousStatus = row.PreviousStatus
		}
		if row.Reason != nil {
			h.Reason = row.Reason
		}
		if row.CreatedBy.Valid {
			uid := uuidToString(row.CreatedBy)
			h.CreatedBy = &uid
		}

		if row.FirstName != nil || row.LastName != nil {
			fname := ""
			if row.FirstName != nil {
				fname = *row.FirstName
			}
			lname := ""
			if row.LastName != nil {
				lname = *row.LastName
			}
			name := fname + " " + lname
			h.CreatedName = &name
		} else if row.Email != nil {
			h.CreatedName = row.Email
		}

		history = append(history, h)
	}
	return history, nil
}

func (r *orderRepository) UpdateOrderShippingDetails(ctx context.Context, id string, address domain.JSONB, shippingFee, totalAmount float64) error {
	addrBytes, err := json.Marshal(address)
	if err != nil {
		return err
	}
	return r.queries.UpdateOrderShippingDetails(ctx, sqlc.UpdateOrderShippingDetailsParams{
		ID:              stringToUUID(id),
		ShippingAddress: addrBytes,
		ShippingFee:     float64ToNumeric(shippingFee),
		TotalAmount:     float64ToNumeric(totalAmount),
	})
}
