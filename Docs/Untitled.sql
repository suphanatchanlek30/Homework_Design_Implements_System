CREATE TABLE `product_categories` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `parent_id` bigint,
  `status` ENUM ('ACTIVE', 'INACTIVE') NOT NULL DEFAULT 'ACTIVE',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `deleted_at` datetime
);

CREATE TABLE `products` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `sku` varchar(100) UNIQUE NOT NULL,
  `name` varchar(255) NOT NULL,
  `category_id` bigint NOT NULL,
  `price_amount` bigint NOT NULL COMMENT 'Money in satang. 100 THB = 10000',
  `currency` varchar(10) NOT NULL DEFAULT 'THB',
  `status` ENUM ('ACTIVE', 'INACTIVE') NOT NULL DEFAULT 'ACTIVE',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `deleted_at` datetime
);

CREATE TABLE `orders` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `order_no` varchar(100) UNIQUE NOT NULL,
  `user_id` bigint NOT NULL,
  `original_total` bigint NOT NULL DEFAULT 0,
  `discount_total` bigint NOT NULL DEFAULT 0,
  `final_total` bigint NOT NULL DEFAULT 0,
  `currency` varchar(10) NOT NULL DEFAULT 'THB',
  `status` ENUM ('DRAFT', 'CONFIRMED', 'PAID', 'CANCELLED') NOT NULL DEFAULT 'DRAFT',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `deleted_at` datetime
);

CREATE TABLE `order_items` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `order_id` bigint NOT NULL,
  `product_id` bigint NOT NULL,
  `product_name` varchar(255) NOT NULL COMMENT 'Snapshot product name at order time',
  `sku` varchar(100) NOT NULL COMMENT 'Snapshot SKU at order time',
  `quantity` int NOT NULL,
  `unit_price` bigint NOT NULL COMMENT 'Snapshot unit price',
  `original_amount` bigint NOT NULL,
  `discount_amount` bigint NOT NULL DEFAULT 0,
  `final_amount` bigint NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE TABLE `promotions` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `code` varchar(100) UNIQUE,
  `name` varchar(255) NOT NULL,
  `description` text,
  `scope` ENUM ('ITEM', 'CART', 'COUPON', 'SHIPPING') NOT NULL,
  `priority` int NOT NULL DEFAULT 100,
  `stackable` boolean NOT NULL DEFAULT true,
  `exclusive` boolean NOT NULL DEFAULT false,
  `stop_processing` boolean NOT NULL DEFAULT false,
  `status` ENUM ('ACTIVE', 'INACTIVE') NOT NULL DEFAULT 'INACTIVE',
  `starts_at` datetime NOT NULL,
  `ends_at` datetime NOT NULL,
  `max_usage` int,
  `max_usage_per_user` int,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `deleted_at` datetime
);

CREATE TABLE `promotion_conditions` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `promotion_id` bigint NOT NULL,
  `condition_type` ENUM ('PRODUCT_ID', 'CATEGORY_ID', 'MIN_ORDER_AMOUNT', 'MAX_ORDER_AMOUNT', 'COUPON_CODE', 'USER_SEGMENT', 'FIRST_ORDER', 'PAYMENT_METHOD', 'DATE_RANGE') NOT NULL,
  `operator` ENUM ('EQ', 'NEQ', 'IN', 'NOT_IN', 'GTE', 'LTE', 'BETWEEN') NOT NULL,
  `value_json` json NOT NULL,
  `group_key` varchar(50),
  `logical_operator` ENUM ('AND', 'OR') NOT NULL DEFAULT 'AND',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE TABLE `promotion_actions` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `promotion_id` bigint NOT NULL,
  `action_type` ENUM ('PERCENTAGE_DISCOUNT', 'FIXED_AMOUNT_DISCOUNT', 'FREE_SHIPPING', 'BUY_X_GET_Y') NOT NULL,
  `value_amount` bigint COMMENT 'Fixed discount amount in satang',
  `value_percent` decimal(5,2) COMMENT 'Percentage discount, e.g. 10.00',
  `value_json` json COMMENT 'Config for complex promotion such as BUY_X_GET_Y',
  `max_discount_amount` bigint,
  `applies_to` ENUM ('ITEM', 'CART', 'SHIPPING') NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE TABLE `promotion_targets` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `promotion_id` bigint NOT NULL,
  `target_type` ENUM ('PRODUCT', 'CATEGORY', 'CART', 'USER_SEGMENT', 'BRAND') NOT NULL,
  `target_id` bigint,
  `target_value` varchar(255),
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE TABLE `promotion_usages` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `promotion_id` bigint NOT NULL,
  `user_id` bigint,
  `order_id` bigint,
  `usage_count` int NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE TABLE `promotion_calculation_logs` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `order_id` bigint,
  `request_id` varchar(100) NOT NULL,
  `user_id` bigint,
  `original_total` bigint NOT NULL,
  `discount_total` bigint NOT NULL,
  `final_total` bigint NOT NULL,
  `applied_promotions_json` json NOT NULL,
  `skipped_promotions_json` json NOT NULL,
  `calculation_snapshot_json` json NOT NULL,
  `created_at` datetime NOT NULL
);

CREATE INDEX `product_categories_index_0` ON `product_categories` (`parent_id`);

CREATE INDEX `product_categories_index_1` ON `product_categories` (`status`);

CREATE UNIQUE INDEX `products_index_2` ON `products` (`sku`);

CREATE INDEX `products_index_3` ON `products` (`category_id`);

CREATE INDEX `products_index_4` ON `products` (`status`);

CREATE UNIQUE INDEX `orders_index_5` ON `orders` (`order_no`);

CREATE INDEX `orders_index_6` ON `orders` (`user_id`);

CREATE INDEX `orders_index_7` ON `orders` (`status`);

CREATE INDEX `orders_index_8` ON `orders` (`created_at`);

CREATE INDEX `order_items_index_9` ON `order_items` (`order_id`);

CREATE INDEX `order_items_index_10` ON `order_items` (`product_id`);

CREATE UNIQUE INDEX `promotions_index_11` ON `promotions` (`code`);

CREATE INDEX `promotions_index_12` ON `promotions` (`status`, `starts_at`, `ends_at`, `priority`);

CREATE INDEX `promotions_index_13` ON `promotions` (`scope`, `priority`);

CREATE INDEX `promotion_conditions_index_14` ON `promotion_conditions` (`promotion_id`);

CREATE INDEX `promotion_conditions_index_15` ON `promotion_conditions` (`condition_type`);

CREATE INDEX `promotion_actions_index_16` ON `promotion_actions` (`promotion_id`);

CREATE INDEX `promotion_actions_index_17` ON `promotion_actions` (`action_type`);

CREATE INDEX `promotion_targets_index_18` ON `promotion_targets` (`promotion_id`);

CREATE INDEX `promotion_targets_index_19` ON `promotion_targets` (`target_type`, `target_id`);

CREATE INDEX `promotion_usages_index_20` ON `promotion_usages` (`promotion_id`);

CREATE INDEX `promotion_usages_index_21` ON `promotion_usages` (`promotion_id`, `user_id`);

CREATE INDEX `promotion_usages_index_22` ON `promotion_usages` (`order_id`);

CREATE INDEX `promotion_calculation_logs_index_23` ON `promotion_calculation_logs` (`order_id`);

CREATE INDEX `promotion_calculation_logs_index_24` ON `promotion_calculation_logs` (`user_id`);

CREATE INDEX `promotion_calculation_logs_index_25` ON `promotion_calculation_logs` (`request_id`);

CREATE INDEX `promotion_calculation_logs_index_26` ON `promotion_calculation_logs` (`created_at`);

ALTER TABLE `product_categories` ADD FOREIGN KEY (`parent_id`) REFERENCES `product_categories` (`id`);

ALTER TABLE `products` ADD FOREIGN KEY (`category_id`) REFERENCES `product_categories` (`id`);

ALTER TABLE `order_items` ADD FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`);

ALTER TABLE `order_items` ADD FOREIGN KEY (`product_id`) REFERENCES `products` (`id`);

ALTER TABLE `promotion_conditions` ADD FOREIGN KEY (`promotion_id`) REFERENCES `promotions` (`id`) ON DELETE CASCADE;

ALTER TABLE `promotion_actions` ADD FOREIGN KEY (`promotion_id`) REFERENCES `promotions` (`id`) ON DELETE CASCADE;

ALTER TABLE `promotion_targets` ADD FOREIGN KEY (`promotion_id`) REFERENCES `promotions` (`id`) ON DELETE CASCADE;

ALTER TABLE `promotion_usages` ADD FOREIGN KEY (`promotion_id`) REFERENCES `promotions` (`id`);

ALTER TABLE `promotion_usages` ADD FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`);

ALTER TABLE `promotion_calculation_logs` ADD FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`);
