CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

CREATE TABLE "addresses" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"user_id" uuid NOT NULL,
	"label" varchar(100),
	"contact_email" varchar(255),
	"phone" varchar(50),
	"first_name" varchar(255),
	"last_name" varchar(255),
	"delivery_zone" varchar(255),
	"division" varchar(255),
	"district" varchar(255),
	"thana" varchar(255),
	"address_line" text,
	"landmark" text,
	"postal_code" varchar(20),
	"is_default" boolean DEFAULT false NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE "cart_items" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"cart_id" uuid NOT NULL,
	"product_id" uuid NOT NULL,
	"variant_id" uuid NOT NULL,
	"quantity" integer DEFAULT 1 NOT NULL,
	CONSTRAINT "cart_items_quantity_check" CHECK ((quantity > 0))
);
CREATE TABLE "carts" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"user_id" uuid CONSTRAINT "carts_user_id_key" UNIQUE,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE "categories" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"name" text,
	"slug" text CONSTRAINT "categories_slug_key" UNIQUE,
	"parent_id" uuid,
	"order_index" integer DEFAULT 0 NOT NULL,
	"icon" varchar(255),
	"image" text,
	"is_active" boolean DEFAULT true NOT NULL,
	"show_in_nav" boolean DEFAULT true NOT NULL,
	"meta_title" varchar(255),
	"meta_description" text,
	"keywords" text,
	"is_featured" boolean DEFAULT false NOT NULL,
	"og_image" text,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE "collections" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"title" varchar(255) NOT NULL,
	"slug" varchar(255) NOT NULL CONSTRAINT "collections_slug_key" UNIQUE,
	"description" text,
	"image" text,
	"story" text,
	"is_active" boolean DEFAULT true NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"meta_title" varchar(255),
	"meta_description" text,
	"meta_keywords" text,
	"og_image" text
);
CREATE TABLE "content_blocks" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"section_key" varchar(100) NOT NULL CONSTRAINT "content_blocks_section_key_key" UNIQUE,
	"content" jsonb DEFAULT '{}' NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	"start_at" timestamp,
	"end_at" timestamp,
	"is_active" boolean DEFAULT true
);
CREATE TABLE "coupons" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"code" varchar(50) NOT NULL CONSTRAINT "coupons_code_key" UNIQUE,
	"type" varchar(20) NOT NULL,
	"value" numeric(12, 2) NOT NULL,
	"min_spend" numeric(12, 2) DEFAULT '0',
	"usage_limit" integer DEFAULT 0,
	"used_count" integer DEFAULT 0,
	"start_at" timestamp,
	"expires_at" timestamp,
	"is_active" boolean DEFAULT true,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT "coupons_type_check" CHECK (((type)::text = ANY ((ARRAY['percentage'::character varying, 'fixed'::character varying])::text[]))),
	CONSTRAINT "coupons_value_check" CHECK ((value > (0)::numeric))
);
CREATE TABLE "daily_sales_stats" (
	"date" date PRIMARY KEY,
	"total_revenue" numeric(15, 2) DEFAULT '0' NOT NULL,
	"total_orders" integer DEFAULT 0 NOT NULL,
	"total_items_sold" integer DEFAULT 0 NOT NULL,
	"avg_order_value" numeric(12, 2) DEFAULT '0' NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE "inventory_logs" (
	"id" serial PRIMARY KEY,
	"product_id" uuid NOT NULL,
	"variant_id" uuid,
	"change_amount" integer NOT NULL,
	"reason" varchar(100) NOT NULL,
	"reference_id" varchar(50) NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE "order_history" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	"order_id" uuid NOT NULL,
	"previous_status" varchar(50),
	"new_status" varchar(50) NOT NULL,
	"reason" text,
	"created_by" uuid,
	"created_at" timestamp with time zone DEFAULT now()
);
CREATE TABLE "order_items" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"order_id" uuid NOT NULL,
	"product_id" uuid NOT NULL,
	"variant_id" uuid,
	"quantity" integer NOT NULL,
	"price" numeric(12, 2) NOT NULL,
	CONSTRAINT "order_items_price_check" CHECK ((price > (0)::numeric)),
	CONSTRAINT "order_items_quantity_check" CHECK ((quantity > 0))
);
CREATE TABLE "orders" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"user_id" uuid NOT NULL,
	"status" varchar(50) DEFAULT 'pending' NOT NULL,
	"total_amount" numeric(12, 2) NOT NULL,
	"shipping_address" jsonb NOT NULL,
	"payment_method" varchar(100),
	"payment_status" varchar(50),
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"paid_amount" numeric(10, 2) DEFAULT '0' NOT NULL,
	"payment_details" jsonb DEFAULT '{}' NOT NULL,
	"is_preorder" boolean DEFAULT false NOT NULL,
	"refunded_amount" numeric(12, 2) DEFAULT '0' NOT NULL,
	"shipping_fee" numeric(12, 2) DEFAULT '0' NOT NULL,
	CONSTRAINT "orders_total_amount_check" CHECK ((total_amount >= (0)::numeric))
);
CREATE TABLE "product_categories" (
	"product_id" uuid,
	"category_id" uuid,
	CONSTRAINT "product_categories_pkey" PRIMARY KEY("product_id","category_id")
);
CREATE TABLE "product_collections" (
	"product_id" uuid,
	"collection_id" uuid,
	CONSTRAINT "product_collections_pkey" PRIMARY KEY("product_id","collection_id")
);
CREATE TABLE "products" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"name" varchar(255) NOT NULL,
	"slug" varchar(255) NOT NULL CONSTRAINT "products_slug_key" UNIQUE,
	"description" text,
	"base_price" numeric(12, 2) NOT NULL,
	"sale_price" numeric(12, 2),
	"stock_status" varchar(50),
	"is_featured" boolean DEFAULT false NOT NULL,
	"is_active" boolean DEFAULT true NOT NULL,
	"media" jsonb,
	"attributes" jsonb,
	"specifications" jsonb,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"search_vector" tsvector,
	"meta_title" varchar(255),
	"meta_description" text,
	"meta_keywords" text,
	"og_image" text,
	"brand" text,
	"tags" text[] DEFAULT '{}',
	"warranty_info" jsonb,
	CONSTRAINT "products_base_price_check" CHECK ((base_price > (0)::numeric)),
	CONSTRAINT "products_sale_price_check" CHECK (((sale_price IS NULL) OR (sale_price > (0)::numeric)))
);
CREATE TABLE "refresh_tokens" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"token" varchar(500) NOT NULL CONSTRAINT "refresh_tokens_token_key" UNIQUE,
	"user_id" uuid NOT NULL,
	"expires_at" timestamp NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"revoked" boolean DEFAULT false NOT NULL,
	"device" varchar(255)
);
CREATE TABLE "refunds" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"order_id" uuid NOT NULL,
	"amount" numeric(12, 2) NOT NULL,
	"reason" text,
	"restock_items" boolean DEFAULT false NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"created_by" uuid,
	CONSTRAINT "refunds_amount_check" CHECK ((amount > (0)::numeric))
);
CREATE TABLE "reviews" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"product_id" uuid NOT NULL UNIQUE,
	"user_id" uuid NOT NULL UNIQUE,
	"rating" integer NOT NULL,
	"comment" text,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT "reviews_product_id_user_id_key" UNIQUE("product_id","user_id"),
	CONSTRAINT "reviews_rating_check" CHECK (((rating >= 1) AND (rating <= 5)))
);

CREATE TABLE "shipping_zones" (
	"id" serial PRIMARY KEY,
	"key" varchar(50) NOT NULL CONSTRAINT "shipping_zones_key_key" UNIQUE,
	"label" varchar(100) NOT NULL,
	"cost" numeric(12, 2) DEFAULT '0' NOT NULL,
	"is_active" boolean DEFAULT true NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE "users" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"email" text NOT NULL CONSTRAINT "users_email_key" UNIQUE,
	"role" varchar(50) DEFAULT 'customer' NOT NULL,
	"first_name" varchar(255),
	"last_name" varchar(255),
	"avatar" text,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"phone" varchar(20)
);
CREATE TABLE "variants" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"product_id" uuid NOT NULL,
	"name" varchar(255) NOT NULL,
	"stock" integer DEFAULT 0 NOT NULL,
	"sku" varchar(100),
	"attributes" jsonb DEFAULT '{}' NOT NULL,
	"price" numeric(10, 2),
	"sale_price" numeric(10, 2),
	"images" text[] DEFAULT '{}',
	"weight" numeric(8, 3),
	"dimensions" jsonb,
	"barcode" text,
	"low_stock_threshold" integer DEFAULT 5 NOT NULL,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT "variants_low_stock_threshold_check" CHECK ((low_stock_threshold >= 0)),
	CONSTRAINT "variants_stock_check" CHECK ((stock >= 0))
);
CREATE TABLE "wishlist_items" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"wishlist_id" uuid NOT NULL UNIQUE,
	"product_id" uuid NOT NULL UNIQUE,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT "wishlist_items_wishlist_id_product_id_key" UNIQUE("wishlist_id","product_id")
);
CREATE TABLE "wishlists" (
	"id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	"user_id" uuid NOT NULL CONSTRAINT "wishlists_user_id_key" UNIQUE,
	"created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL
);
ALTER TABLE "addresses" ADD CONSTRAINT "addresses_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE;
ALTER TABLE "cart_items" ADD CONSTRAINT "cart_items_cart_id_fkey" FOREIGN KEY ("cart_id") REFERENCES "carts"("id") ON DELETE CASCADE;
ALTER TABLE "cart_items" ADD CONSTRAINT "cart_items_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "cart_items" ADD CONSTRAINT "cart_items_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "variants"("id") ON DELETE SET NULL;
ALTER TABLE "carts" ADD CONSTRAINT "carts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE;
ALTER TABLE "categories" ADD CONSTRAINT "categories_parent_id_fkey" FOREIGN KEY ("parent_id") REFERENCES "categories"("id") ON DELETE SET NULL;
ALTER TABLE "inventory_logs" ADD CONSTRAINT "inventory_logs_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "inventory_logs" ADD CONSTRAINT "inventory_logs_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "variants"("id") ON DELETE SET NULL;
ALTER TABLE "order_history" ADD CONSTRAINT "order_history_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "users"("id");
ALTER TABLE "order_history" ADD CONSTRAINT "order_history_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders"("id") ON DELETE CASCADE;
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders"("id") ON DELETE CASCADE;
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE RESTRICT;
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "variants"("id") ON DELETE SET NULL;
ALTER TABLE "orders" ADD CONSTRAINT "orders_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE RESTRICT;
ALTER TABLE "product_categories" ADD CONSTRAINT "product_categories_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE CASCADE;
ALTER TABLE "product_categories" ADD CONSTRAINT "product_categories_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "product_collections" ADD CONSTRAINT "product_collections_collection_id_fkey" FOREIGN KEY ("collection_id") REFERENCES "collections"("id") ON DELETE CASCADE;
ALTER TABLE "product_collections" ADD CONSTRAINT "product_collections_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "refresh_tokens_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE;
ALTER TABLE "refunds" ADD CONSTRAINT "refunds_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "users"("id") ON DELETE SET NULL;
ALTER TABLE "refunds" ADD CONSTRAINT "refunds_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders"("id") ON DELETE CASCADE;
ALTER TABLE "reviews" ADD CONSTRAINT "reviews_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "reviews" ADD CONSTRAINT "reviews_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE;
ALTER TABLE "variants" ADD CONSTRAINT "variants_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "wishlist_items" ADD CONSTRAINT "wishlist_items_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products"("id") ON DELETE CASCADE;
ALTER TABLE "wishlist_items" ADD CONSTRAINT "wishlist_items_wishlist_id_fkey" FOREIGN KEY ("wishlist_id") REFERENCES "wishlists"("id") ON DELETE CASCADE;
ALTER TABLE "wishlists" ADD CONSTRAINT "wishlists_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE;
CREATE INDEX "idx_addresses_user_id" ON "addresses" ("user_id");
CREATE INDEX "idx_cart_items_cart_id" ON "cart_items" ("cart_id");
CREATE INDEX "idx_cart_items_product_id" ON "cart_items" ("product_id");
CREATE INDEX "idx_carts_user_id" ON "carts" ("user_id");
CREATE INDEX "idx_categories_is_active" ON "categories" ("is_active");
CREATE INDEX "idx_categories_parent_id" ON "categories" ("parent_id");
CREATE INDEX "idx_categories_slug" ON "categories" ("slug");
CREATE INDEX "idx_collections_created_at" ON "collections" ("created_at");
CREATE INDEX "idx_collections_is_active" ON "collections" ("is_active");
CREATE INDEX "idx_collections_slug" ON "collections" ("slug");
CREATE INDEX "idx_content_blocks_active" ON "content_blocks" ("section_key");
CREATE INDEX "idx_content_blocks_active_time" ON "content_blocks" ("is_active","start_at","end_at");
CREATE INDEX "idx_content_blocks_key" ON "content_blocks" ("section_key");
CREATE INDEX "idx_coupons_code" ON "coupons" ("code");
CREATE INDEX "idx_coupons_created_at" ON "coupons" ("created_at");
CREATE INDEX "idx_coupons_expires_at" ON "coupons" ("expires_at");
CREATE INDEX "idx_coupons_is_active" ON "coupons" ("is_active");
CREATE INDEX "idx_inventory_logs_created_at" ON "inventory_logs" ("created_at");
CREATE INDEX "idx_inventory_logs_product_id" ON "inventory_logs" ("product_id");
CREATE INDEX "idx_inventory_logs_variant_id" ON "inventory_logs" ("variant_id");
CREATE INDEX "idx_order_history_order_id" ON "order_history" ("order_id");
CREATE INDEX "idx_order_items_order_id" ON "order_items" ("order_id");
CREATE INDEX "idx_order_items_product_id" ON "order_items" ("product_id");
CREATE INDEX "idx_orders_created_at" ON "orders" ("created_at");
CREATE INDEX "idx_orders_id_trgm" ON "orders" USING gin ((id::text) gin_trgm_ops);
CREATE INDEX "idx_orders_is_preorder" ON "orders" ("is_preorder");
CREATE INDEX "idx_orders_payment_phone_trgm" ON "orders" USING gin ((payment_details ->> 'sender_number') gin_trgm_ops);
CREATE INDEX "idx_orders_payment_status" ON "orders" ("payment_status");
CREATE INDEX "idx_orders_payment_trx_trgm" ON "orders" USING gin ((payment_details ->> 'transaction_id') gin_trgm_ops);
CREATE INDEX "idx_orders_status" ON "orders" ("status");
CREATE INDEX "idx_orders_status_created_desc" ON "orders" ("status","created_at");
CREATE INDEX "idx_orders_user_id" ON "orders" ("user_id");
CREATE INDEX "idx_product_categories_category_id" ON "product_categories" ("category_id");
CREATE INDEX "idx_product_categories_product_id" ON "product_categories" ("product_id");
CREATE INDEX "idx_product_collections_collection_id" ON "product_collections" ("collection_id");
CREATE INDEX "idx_product_collections_product_id" ON "product_collections" ("product_id");
CREATE INDEX "idx_products_base_price" ON "products" ("base_price");
CREATE INDEX "idx_products_brand" ON "products" ("brand");
CREATE INDEX "idx_products_category_slug_active" ON "products" ("is_active");
CREATE INDEX "idx_products_created_at" ON "products" ("created_at");
CREATE INDEX "idx_products_featured_active_created" ON "products" ("is_featured","is_active","created_at");
CREATE INDEX "idx_products_is_active" ON "products" ("is_active");
CREATE INDEX "idx_products_is_featured" ON "products" ("is_featured");
CREATE INDEX "idx_products_search_vector" ON "products" USING gin ("search_vector");
CREATE INDEX "idx_products_slug" ON "products" ("slug");
CREATE INDEX "idx_products_tags" ON "products" USING gin ("tags");
CREATE INDEX "idx_refresh_tokens_token" ON "refresh_tokens" ("token");
CREATE INDEX "idx_refresh_tokens_user_id" ON "refresh_tokens" ("user_id");
CREATE INDEX "idx_refunds_order_id" ON "refunds" ("order_id");
CREATE INDEX "idx_reviews_product_id" ON "reviews" ("product_id");
CREATE INDEX "idx_reviews_user_id" ON "reviews" ("user_id");

CREATE INDEX "idx_users_email_trgm" ON "users" USING gin (email gin_trgm_ops);
CREATE INDEX "idx_variants_attributes" ON "variants" USING gin ("attributes");
CREATE INDEX "idx_variants_product_id" ON "variants" ("product_id");
CREATE INDEX "idx_wishlist_items_product_id" ON "wishlist_items" ("product_id");
CREATE INDEX "idx_wishlist_items_wishlist_id" ON "wishlist_items" ("wishlist_id");
CREATE INDEX "idx_wishlists_user_id" ON "wishlists" ("user_id");
