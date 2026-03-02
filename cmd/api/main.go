package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"valancis-backend/config"
	"valancis-backend/internal/delivery/http/middleware"
	v1 "valancis-backend/internal/delivery/http/v1"
	"valancis-backend/internal/infrastructure/cache"
	"valancis-backend/internal/infrastructure/facebook"
	sqlcrepo "valancis-backend/internal/repository/sqlc"
	"valancis-backend/internal/usecase"
	"valancis-backend/pkg/logger"
	"valancis-backend/pkg/storage"
	"valancis-backend/pkg/utils"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
)

func main() {
	cfg := config.LoadConfig()
	utils.SetSecret(cfg.JWTSecret)

	// Initialize Logger
	logger.Init(cfg.Env, cfg.LogLevel)
	log := logger.Get()

	// Initialize Database with pgx/sqlc
	pgxPool, err := sqlcrepo.NewPgxPool(context.Background(), cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	log.Info().Msg("Successfully connected to PostgreSQL via pgx/sqlc")

	// Initialize Repositories
	userRepo := sqlcrepo.NewUserRepository(pgxPool)
	productRepo := sqlcrepo.NewProductRepository(pgxPool)
	orderRepo := sqlcrepo.NewOrderRepository(pgxPool)
	configRepo := sqlcrepo.NewConfigRepository(pgxPool)
	searchRepo := sqlcrepo.NewSearchRepository(pgxPool)
	txManager := sqlcrepo.NewTransactionManager(pgxPool)
	couponRepo := sqlcrepo.NewCouponRepository(pgxPool)

	// Initialize Cache (In-Memory)
	// Default expiration 30m, cleanup every 60m
	memCache := cache.NewMemoryCache(30*time.Minute, 60*time.Minute)

	// Set up Router
	mux := http.NewServeMux()

	// --- Modules Initialization ---

	// Auth Module
	authUC := usecase.NewAuthUsecase(
		userRepo,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleTokenInfoURL,
		cfg.AccessTokenExpiry,
		cfg.RefreshTokenExpiry,
	)
	authHandler := v1.NewAuthHandler(authUC)

	// --- Storage Module (R2) ---
	r2Storage, err := storage.NewR2Storage(
		context.Background(),
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2AccessKeySecret,
		cfg.R2BucketName,
		cfg.R2PublicURL,
		cfg.R2UploadTimeout,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize R2 Storage")
	}
	uploadHandler := v1.NewUploadHandler(r2Storage, cfg.MaxUploadSizeMB)

	// Catalog Module
	catalogUC := usecase.NewCatalogUsecase(productRepo, orderRepo, memCache, r2Storage, cfg)
	catalogHandler := v1.NewCatalogHandler(catalogUC)

	// Admin Catalog Handlers
	adminCatalogHandler := v1.NewAdminCatalogHandler(catalogUC)

	// Facebook CAPI Client (Marketing / Analytics)
	capiClient := facebook.NewCAPIClient(cfg.FacebookPixelID, cfg.FacebookAccessToken, cfg.FacebookAPIVersion)

	// Order Module
	orderUC := usecase.NewOrderUsecase(orderRepo, productRepo, configRepo, couponRepo, txManager, capiClient)
	orderHandler := v1.NewOrderHandler(orderUC, cfg.MaxCartQuantity)
	adminOrderHandler := v1.NewAdminOrderHandler(orderUC)

	// Content Module
	contentRepo := sqlcrepo.NewContentRepository(pgxPool)
	contentUC := usecase.NewContentUsecase(contentRepo)
	contentHandler := v1.NewContentHandler(contentUC)

	// Search Module
	searchUC := usecase.NewSearchUsecase(searchRepo, 5*time.Second)
	searchHandler := v1.NewSearchHandler(searchUC)

	// Sitemap Module
	sitemapUC := usecase.NewSitemapUsecase(productRepo, cfg.FrontendURL, memCache, cfg)
	sitemapHandler := v1.NewSitemapHandler(sitemapUC)

	// Stats Module (Analytics)
	statsUC := usecase.NewStatsUsecase(pgxPool, memCache)
	adminStatsHandler := v1.NewAdminStatsHandler(statsUC)

	// Config Handler
	configHandler := v1.NewConfigHandler(memCache, configRepo)
	adminConfigHandler := v1.NewAdminConfigHandler(memCache, configRepo)

	// Config (Public)
	mux.HandleFunc("GET /api/v1/config/enums", configHandler.GetEnums)

	// Config (Admin)
	mux.Handle("GET /api/v1/admin/config/shipping-zones", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminConfigHandler.GetAllShippingZones))))
	mux.Handle("POST /api/v1/admin/config/shipping-zones", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminConfigHandler.CreateShippingZone))))
	mux.Handle("PATCH /api/v1/admin/config/shipping-zones/{id}", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminConfigHandler.UpdateShippingZone))))
	mux.Handle("DELETE /api/v1/admin/config/shipping-zones/{id}", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminConfigHandler.DeleteShippingZone))))

	// Auth
	mux.HandleFunc("POST /api/v1/auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/v1/auth/me", middleware.AuthMiddleware(http.HandlerFunc(authHandler.Me)))

	// User Profile / Address
	mux.Handle("POST /api/v1/user/addresses", middleware.AuthMiddleware(http.HandlerFunc(authHandler.AddAddress)))
	mux.Handle("GET /api/v1/user/addresses", middleware.AuthMiddleware(http.HandlerFunc(authHandler.GetAddresses)))
	mux.Handle("PUT /api/v1/user/addresses/{id}", middleware.AuthMiddleware(http.HandlerFunc(authHandler.UpdateAddress)))
	mux.Handle("DELETE /api/v1/user/addresses/{id}", middleware.AuthMiddleware(http.HandlerFunc(authHandler.DeleteAddress)))
	mux.Handle("PUT /api/v1/user/profile", middleware.AuthMiddleware(http.HandlerFunc(authHandler.UpdateProfile)))

	// Uploads
	mux.Handle("POST /api/v1/upload", middleware.AuthMiddleware(http.HandlerFunc(uploadHandler.UploadFile)))

	// Content (Public)
	mux.HandleFunc("GET /api/v1/content/{key}", contentHandler.GetContent)

	// Catalog (Public)
	mux.HandleFunc("GET /sitemap.xml", sitemapHandler.ServeHTTP)
	mux.HandleFunc("GET /api/v1/categories", catalogHandler.GetCategories)
	mux.HandleFunc("GET /api/v1/categories/tree", catalogHandler.GetCategories)
	mux.HandleFunc("GET /api/v1/products", catalogHandler.ListProducts)
	mux.HandleFunc("GET /api/v1/product/{id}", catalogHandler.GetProductByID)
	mux.HandleFunc("GET /api/v1/products/{slug}", catalogHandler.GetProductDetails)
	mux.HandleFunc("GET /api/v1/search", searchHandler.Search)
	mux.HandleFunc("GET /api/v1/products/{id}/reviews", catalogHandler.GetReviews)                                          // Public
	mux.Handle("POST /api/v1/products/{id}/reviews", middleware.AuthMiddleware(http.HandlerFunc(catalogHandler.AddReview))) // Protected

	mux.HandleFunc("GET /api/v1/collections", catalogHandler.GetCollections)
	mux.HandleFunc("GET /api/v1/collections/{slug}", catalogHandler.GetCollectionBySlug)

	// Admin (Protected)
	adminMiddleware := func(h http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.AdminMiddleware(h))
	}

	// Admin Content
	mux.Handle("PUT /api/v1/admin/content/{key}", adminMiddleware(contentHandler.UpsertContent))

	// Admin Config
	mux.Handle("GET /api/v1/admin/config/enums", adminMiddleware(configHandler.GetEnums))

	// Admin Product Management
	mux.Handle("GET /api/v1/admin/products", adminMiddleware(adminCatalogHandler.ListProducts))
	mux.Handle("GET /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.GetProduct))
	mux.Handle("POST /api/v1/admin/products", adminMiddleware(adminCatalogHandler.CreateProduct))
	mux.Handle("PUT /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.UpdateProduct))
	mux.Handle("PATCH /api/v1/admin/products/{id}/status", adminMiddleware(adminCatalogHandler.UpdateProductStatus))
	mux.Handle("DELETE /api/v1/admin/products/{id}", adminMiddleware(adminCatalogHandler.DeleteProduct))
	mux.Handle("POST /api/v1/admin/inventory/adjust", adminMiddleware(adminCatalogHandler.AdjustStock))
	mux.Handle("GET /api/v1/admin/inventory/logs", adminMiddleware(adminCatalogHandler.GetInventoryLogs))
	mux.Handle("GET /api/v1/admin/inventory/variants", adminMiddleware(adminCatalogHandler.GetVariantList))
	mux.Handle("GET /api/v1/admin/products/stats", adminMiddleware(adminCatalogHandler.GetProductStats))

	mux.Handle("GET /api/v1/admin/categories", adminMiddleware(adminCatalogHandler.GetAllCategories))
	mux.Handle("GET /api/v1/admin/categories/tree", adminMiddleware(http.HandlerFunc(catalogHandler.GetCategories)))
	mux.Handle("POST /api/v1/admin/categories", adminMiddleware(adminCatalogHandler.CreateCategory))
	mux.Handle("PUT /api/v1/admin/categories/{id}", adminMiddleware(adminCatalogHandler.UpdateCategory))
	mux.Handle("DELETE /api/v1/admin/categories/{id}", adminMiddleware(adminCatalogHandler.DeleteCategory))
	mux.Handle("POST /api/v1/admin/categories/reorder", adminMiddleware(adminCatalogHandler.ReorderCategories))

	// Collections
	mux.Handle("GET /api/v1/admin/collections", adminMiddleware(adminCatalogHandler.GetAllCollections))
	mux.Handle("POST /api/v1/admin/collections", adminMiddleware(adminCatalogHandler.CreateCollection))
	mux.Handle("PUT /api/v1/admin/collections/{id}", adminMiddleware(adminCatalogHandler.UpdateCollection))
	mux.Handle("DELETE /api/v1/admin/collections/{id}", adminMiddleware(adminCatalogHandler.DeleteCollection))
	mux.Handle("POST /api/v1/admin/collections/{id}/products", adminMiddleware(adminCatalogHandler.ManageCollectionProduct))

	mux.Handle("GET /api/v1/admin/orders", adminMiddleware(adminOrderHandler.ListOrders))
	mux.Handle("GET /api/v1/admin/orders/{id}", adminMiddleware(adminOrderHandler.GetOrder))
	mux.Handle("PATCH /api/v1/admin/orders/{id}/status", adminMiddleware(adminOrderHandler.UpdateStatus))
	mux.Handle("PATCH /api/v1/admin/orders/{id}/payment-status", adminMiddleware(adminOrderHandler.UpdatePaymentStatus))
	mux.Handle("POST /api/v1/admin/orders/{id}/verify-payment", adminMiddleware(adminOrderHandler.VerifyPayment))
	mux.Handle("POST /api/v1/admin/orders/{id}/refund", adminMiddleware(adminOrderHandler.RefundOrder))
	mux.Handle("GET /api/v1/admin/orders/{id}/history", adminMiddleware(adminOrderHandler.GetOrderHistory))
	mux.Handle("GET /api/v1/admin/users", adminMiddleware(authHandler.ListUsers))

	// Admin Coupons
	// couponUC := usecase.NewCouponUsecase(couponRepo)
	// adminCouponHandler := v1.NewAdminCouponHandler(couponUC)
	// mux.Handle("GET /api/v1/admin/coupons", adminMiddleware(adminCouponHandler.ListCoupons))
	// mux.Handle("GET /api/v1/admin/coupons/{id}", adminMiddleware(adminCouponHandler.GetCoupon))
	// mux.Handle("POST /api/v1/admin/coupons", adminMiddleware(adminCouponHandler.CreateCoupon))
	// mux.Handle("PUT /api/v1/admin/coupons/{id}", adminMiddleware(adminCouponHandler.UpdateCoupon))
	// mux.Handle("DELETE /api/v1/admin/coupons/{id}", adminMiddleware(adminCouponHandler.DeleteCoupon))

	// Cart & Order (Protected)
	mux.Handle("GET /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetCart)))
	mux.Handle("POST /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.AddToCart)))
	mux.Handle("PUT /api/v1/cart", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.UpdateCart)))
	mux.Handle("DELETE /api/v1/cart/{productId}", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.RemoveFromCart)))
	// mux.Handle("POST /api/v1/cart/coupon", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.ApplyCoupon)))
	mux.Handle("POST /api/v1/checkout", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.Checkout)))
	mux.Handle("GET /api/v1/orders", middleware.AuthMiddleware(http.HandlerFunc(orderHandler.GetMyOrders)))

	// Wishlist Module
	wishlistRepo := sqlcrepo.NewWishlistRepository(pgxPool)
	wishlistUC := usecase.NewWishlistUsecase(wishlistRepo)
	wishlistHandler := v1.NewWishlistHandler(wishlistUC)

	mux.Handle("GET /api/v1/wishlist", middleware.AuthMiddleware(http.HandlerFunc(wishlistHandler.GetMyWishlist)))
	mux.Handle("POST /api/v1/wishlist", middleware.AuthMiddleware(http.HandlerFunc(wishlistHandler.AddToWishlist)))
	mux.Handle("DELETE /api/v1/wishlist/{productId}", middleware.AuthMiddleware(http.HandlerFunc(wishlistHandler.RemoveFromWishlist)))

	// Admin Stats Routes (Analytics)
	// Admin Stats Routes (Analytics)
	// Must chain: AuthMiddleware -> AdminMiddleware -> Handler
	mux.Handle("GET /api/v1/admin/stats/revenue", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetDailySales))))
	mux.Handle("GET /api/v1/admin/stats/kpis", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetRevenueKPIs))))
	mux.Handle("GET /api/v1/admin/stats/inventory/low-stock", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetLowStockProducts))))
	mux.Handle("GET /api/v1/admin/stats/inventory/dead-stock", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetDeadStockProducts))))
	mux.Handle("GET /api/v1/admin/stats/products/top-selling", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetTopSellingProducts))))
	mux.Handle("GET /api/v1/admin/stats/customers/top", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetTopCustomers))))
	mux.Handle("GET /api/v1/admin/stats/customers/retention", middleware.AuthMiddleware(middleware.AdminMiddleware(http.HandlerFunc(adminStatsHandler.GetCustomerRetention))))

	// Health Check
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "db": "connected"}`))
	}
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.HandleFunc("GET /health", healthHandler) // Support root health check for Load Balancers

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Info().Msgf("Server starting on %s", addr)

	// Initialize Rate Limiter with lifecycle management
	// 50 req/s, burst 100, cleanup every minute, TTL 3 minutes
	rateLimiter := middleware.NewRateLimiter(
		context.Background(),
		50,            // requests per second
		100,           // burst
		time.Minute,   // cleanup period
		3*time.Minute, // client TTL
	)

	// Apply CORS (with config injection), Request Logger, Rate Limit, and Gzip
	handler := middleware.NewCORSMiddleware(cfg)(mux)
	handler = middleware.RequestLogger(handler)
	handler = rateLimiter.Middleware()(handler)
	handler = gziphandler.GzipHandler(handler)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Graceful Shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	log.Info().Msgf("Server starting on %s", addr)

	// Wait for interrupt signal via channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Server shutting down...")

	// L9: Graceful shutdown - stop rate limiter cleanup goroutine
	rateLimiter.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}
