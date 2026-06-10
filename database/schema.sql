-- Promotion Engine Database Schema
-- Target DB: MySQL 8.0+
-- Money is stored as BIGINT minor units. For THB, 100 THB = 10000 satang.
-- Percentage discounts use basis points. 10% = 1000, 100% = 10000.

CREATE DATABASE IF NOT EXISTS promotion_engine
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE promotion_engine;

CREATE TABLE product_categories (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  parent_id BIGINT UNSIGNED NULL,
  status ENUM('ACTIVE', 'INACTIVE') NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  CONSTRAINT fk_product_categories_parent
    FOREIGN KEY (parent_id) REFERENCES product_categories(id)
    ON DELETE SET NULL
    ON UPDATE CASCADE
) ENGINE=InnoDB;

CREATE TABLE products (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  sku VARCHAR(100) NOT NULL,
  name VARCHAR(255) NOT NULL,
  category_id BIGINT UNSIGNED NOT NULL,
  price_amount BIGINT NOT NULL,
  currency VARCHAR(10) NOT NULL DEFAULT 'THB',
  status ENUM('ACTIVE', 'INACTIVE') NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_products_sku UNIQUE (sku),
  CONSTRAINT fk_products_category
    FOREIGN KEY (category_id) REFERENCES product_categories(id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE,
  CONSTRAINT chk_products_price_non_negative CHECK (price_amount >= 0)
) ENGINE=InnoDB;

CREATE TABLE orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_no VARCHAR(100) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  original_total BIGINT NOT NULL DEFAULT 0,
  discount_total BIGINT NOT NULL DEFAULT 0,
  final_total BIGINT NOT NULL DEFAULT 0,
  currency VARCHAR(10) NOT NULL DEFAULT 'THB',
  status ENUM('DRAFT', 'CONFIRMED', 'PAID', 'CANCELLED') NOT NULL DEFAULT 'DRAFT',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_orders_order_no UNIQUE (order_no),
  CONSTRAINT chk_orders_money_non_negative
    CHECK (original_total >= 0 AND discount_total >= 0 AND final_total >= 0)
) ENGINE=InnoDB;

CREATE TABLE order_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id BIGINT UNSIGNED NOT NULL,
  product_id BIGINT UNSIGNED NOT NULL,
  product_name VARCHAR(255) NOT NULL,
  sku VARCHAR(100) NOT NULL,
  quantity INT NOT NULL,
  unit_price BIGINT NOT NULL,
  original_amount BIGINT NOT NULL,
  discount_amount BIGINT NOT NULL DEFAULT 0,
  final_amount BIGINT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT fk_order_items_order
    FOREIGN KEY (order_id) REFERENCES orders(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT fk_order_items_product
    FOREIGN KEY (product_id) REFERENCES products(id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE,
  CONSTRAINT chk_order_items_quantity_positive CHECK (quantity > 0),
  CONSTRAINT chk_order_items_money_non_negative
    CHECK (unit_price >= 0 AND original_amount >= 0 AND discount_amount >= 0 AND final_amount >= 0)
) ENGINE=InnoDB;

CREATE TABLE promotions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(100) NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT NULL,
  scope ENUM('ITEM', 'CART', 'COUPON', 'SHIPPING') NOT NULL,
  priority INT NOT NULL DEFAULT 100,
  stackable BOOLEAN NOT NULL DEFAULT TRUE,
  exclusive BOOLEAN NOT NULL DEFAULT FALSE,
  stop_processing BOOLEAN NOT NULL DEFAULT FALSE,
  conflict_group VARCHAR(100) NULL,
  status ENUM('DRAFT', 'ACTIVE', 'INACTIVE', 'EXPIRED') NOT NULL DEFAULT 'DRAFT',
  starts_at DATETIME(3) NOT NULL,
  ends_at DATETIME(3) NOT NULL,
  max_usage INT NULL,
  max_usage_per_user INT NULL,
  version INT NOT NULL DEFAULT 1,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  CONSTRAINT uq_promotions_code UNIQUE (code),
  CONSTRAINT chk_promotions_priority_non_negative CHECK (priority >= 0),
  CONSTRAINT chk_promotions_date_range CHECK (starts_at < ends_at),
  CONSTRAINT chk_promotions_usage_positive
    CHECK ((max_usage IS NULL OR max_usage > 0) AND (max_usage_per_user IS NULL OR max_usage_per_user > 0)),
  CONSTRAINT chk_promotions_version_positive CHECK (version > 0)
) ENGINE=InnoDB;

CREATE TABLE promotion_targets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  promotion_id BIGINT UNSIGNED NOT NULL,
  target_type ENUM('PRODUCT', 'CATEGORY', 'CART', 'USER_SEGMENT', 'BRAND') NOT NULL,
  target_id BIGINT UNSIGNED NULL,
  target_value VARCHAR(255) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT fk_promotion_targets_promotion
    FOREIGN KEY (promotion_id) REFERENCES promotions(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
) ENGINE=InnoDB;

CREATE TABLE promotion_conditions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  promotion_id BIGINT UNSIGNED NOT NULL,
  condition_type ENUM(
    'PRODUCT_ID',
    'CATEGORY_ID',
    'MIN_ORDER_AMOUNT',
    'MAX_ORDER_AMOUNT',
    'COUPON_CODE',
    'USER_SEGMENT',
    'FIRST_ORDER',
    'PAYMENT_METHOD',
    'DATE_RANGE'
  ) NOT NULL,
  operator ENUM('EQ', 'NEQ', 'IN', 'NOT_IN', 'GTE', 'LTE', 'BETWEEN') NOT NULL,
  value_json JSON NOT NULL,
  group_key VARCHAR(50) NULL,
  logical_operator ENUM('AND', 'OR') NOT NULL DEFAULT 'AND',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT fk_promotion_conditions_promotion
    FOREIGN KEY (promotion_id) REFERENCES promotions(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT chk_promotion_conditions_value_json CHECK (JSON_VALID(value_json))
) ENGINE=InnoDB;

CREATE TABLE promotion_actions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  promotion_id BIGINT UNSIGNED NOT NULL,
  action_type ENUM(
    'PERCENTAGE_DISCOUNT',
    'FIXED_AMOUNT_DISCOUNT',
    'CART_PERCENTAGE_DISCOUNT',
    'CART_FIXED_AMOUNT_DISCOUNT',
    'FREE_SHIPPING',
    'BUY_X_GET_Y',
    'BUNDLE_DISCOUNT'
  ) NOT NULL,
  value_amount BIGINT NULL,
  value_basis_points INT NULL,
  value_json JSON NULL,
  max_discount_amount BIGINT NULL,
  applies_to ENUM('ITEM', 'CART', 'SHIPPING') NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT fk_promotion_actions_promotion
    FOREIGN KEY (promotion_id) REFERENCES promotions(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT chk_promotion_actions_value_amount CHECK (value_amount IS NULL OR value_amount > 0),
  CONSTRAINT chk_promotion_actions_basis_points CHECK (value_basis_points IS NULL OR (value_basis_points > 0 AND value_basis_points <= 10000)),
  CONSTRAINT chk_promotion_actions_max_discount CHECK (max_discount_amount IS NULL OR max_discount_amount > 0),
  CONSTRAINT chk_promotion_actions_value_json CHECK (value_json IS NULL OR JSON_VALID(value_json))
) ENGINE=InnoDB;

CREATE TABLE promotion_usages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  promotion_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NULL,
  order_id BIGINT UNSIGNED NULL,
  usage_count INT NOT NULL DEFAULT 1,
  discount_amount BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT fk_promotion_usages_promotion
    FOREIGN KEY (promotion_id) REFERENCES promotions(id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE,
  CONSTRAINT fk_promotion_usages_order
    FOREIGN KEY (order_id) REFERENCES orders(id)
    ON DELETE SET NULL
    ON UPDATE CASCADE,
  CONSTRAINT chk_promotion_usages_count_positive CHECK (usage_count > 0),
  CONSTRAINT chk_promotion_usages_discount_non_negative CHECK (discount_amount >= 0)
) ENGINE=InnoDB;

CREATE TABLE promotion_calculation_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  calculation_id VARCHAR(100) NOT NULL,
  order_id BIGINT UNSIGNED NULL,
  request_id VARCHAR(100) NOT NULL,
  user_id BIGINT UNSIGNED NULL,
  original_total BIGINT NOT NULL,
  discount_total BIGINT NOT NULL,
  final_total BIGINT NOT NULL,
  applied_promotions_json JSON NOT NULL,
  skipped_promotions_json JSON NOT NULL,
  calculation_snapshot_json JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT uq_promotion_calculation_logs_calculation_id UNIQUE (calculation_id),
  CONSTRAINT fk_calculation_logs_order
    FOREIGN KEY (order_id) REFERENCES orders(id)
    ON DELETE SET NULL
    ON UPDATE CASCADE,
  CONSTRAINT chk_calculation_logs_money_non_negative
    CHECK (original_total >= 0 AND discount_total >= 0 AND final_total >= 0),
  CONSTRAINT chk_calculation_logs_applied_json CHECK (JSON_VALID(applied_promotions_json)),
  CONSTRAINT chk_calculation_logs_skipped_json CHECK (JSON_VALID(skipped_promotions_json)),
  CONSTRAINT chk_calculation_logs_snapshot_json CHECK (JSON_VALID(calculation_snapshot_json))
) ENGINE=InnoDB;

CREATE TABLE idempotency_keys (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  idempotency_key VARCHAR(160) NOT NULL,
  request_hash CHAR(64) NOT NULL,
  resource_type VARCHAR(50) NOT NULL,
  resource_id BIGINT UNSIGNED NULL,
  response_json JSON NULL,
  status ENUM('PROCESSING', 'SUCCEEDED', 'FAILED') NOT NULL DEFAULT 'PROCESSING',
  expires_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT uq_idempotency_keys_key UNIQUE (idempotency_key),
  CONSTRAINT chk_idempotency_keys_response_json CHECK (response_json IS NULL OR JSON_VALID(response_json))
) ENGINE=InnoDB;

-- Product/category lookup indexes.
CREATE INDEX idx_product_categories_parent_status ON product_categories(parent_id, status);
CREATE INDEX idx_product_categories_status_name ON product_categories(status, name);
CREATE INDEX idx_products_status_id ON products(status, deleted_at, id);
CREATE INDEX idx_products_category_status ON products(category_id, status, deleted_at);
CREATE INDEX idx_products_currency_status ON products(currency, status, deleted_at);

-- Order read/query indexes.
CREATE INDEX idx_orders_user_status_created ON orders(user_id, status, created_at);
CREATE INDEX idx_orders_status_created ON orders(status, created_at);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_order_product ON order_items(order_id, product_id);

-- Promotion loading indexes.
-- Main query: active promotions for one request, ordered deterministically.
CREATE INDEX idx_promotions_active_window
  ON promotions(status, deleted_at, starts_at, ends_at);
CREATE INDEX idx_promotions_active_sort
  ON promotions(status, deleted_at, scope, priority, created_at, id);
CREATE INDEX idx_promotions_scope_priority
  ON promotions(scope, priority, created_at, id);
CREATE INDEX idx_promotions_status_updated
  ON promotions(status, updated_at);
CREATE INDEX idx_promotions_conflict
  ON promotions(scope, conflict_group, priority);

-- Promotion relation/preload indexes.
CREATE INDEX idx_promotion_targets_promotion_id
  ON promotion_targets(promotion_id);
CREATE INDEX idx_promotion_targets_lookup
  ON promotion_targets(target_type, target_id, promotion_id);
CREATE INDEX idx_promotion_targets_value
  ON promotion_targets(target_type, target_value, promotion_id);

CREATE INDEX idx_promotion_conditions_promotion_id
  ON promotion_conditions(promotion_id);
CREATE INDEX idx_promotion_conditions_type
  ON promotion_conditions(condition_type, operator);
CREATE INDEX idx_promotion_conditions_group
  ON promotion_conditions(promotion_id, group_key, logical_operator);

CREATE INDEX idx_promotion_actions_promotion_id
  ON promotion_actions(promotion_id);
CREATE INDEX idx_promotion_actions_type_applies
  ON promotion_actions(action_type, applies_to);

-- Usage limit indexes for confirm-order transaction and support reports.
CREATE INDEX idx_promotion_usages_promotion_id
  ON promotion_usages(promotion_id);
CREATE INDEX idx_promotion_usages_promo_user
  ON promotion_usages(promotion_id, user_id);
CREATE INDEX idx_promotion_usages_order_id
  ON promotion_usages(order_id);
CREATE INDEX idx_promotion_usages_created
  ON promotion_usages(created_at);

-- Audit/search indexes.
CREATE INDEX idx_calculation_logs_request_id
  ON promotion_calculation_logs(request_id);
CREATE INDEX idx_calculation_logs_order_id
  ON promotion_calculation_logs(order_id);
CREATE INDEX idx_calculation_logs_user_created
  ON promotion_calculation_logs(user_id, created_at);
CREATE INDEX idx_calculation_logs_created_at
  ON promotion_calculation_logs(created_at);

CREATE INDEX idx_idempotency_keys_resource
  ON idempotency_keys(resource_type, resource_id);
CREATE INDEX idx_idempotency_keys_status_expires
  ON idempotency_keys(status, expires_at);
