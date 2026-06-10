USE promotion_engine;

INSERT INTO product_categories (id, name, parent_id, status)
VALUES (1, 'Default', NULL, 'ACTIVE')
ON DUPLICATE KEY UPDATE name = VALUES(name), status = VALUES(status);

INSERT INTO products (id, sku, name, category_id, price_amount, currency, status)
VALUES
  (1, 'PRODUCT-001', 'Product 1', 1, 100000, 'THB', 'ACTIVE'),
  (2, 'PRODUCT-002', 'Product 2', 1, 50000, 'THB', 'ACTIVE')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  category_id = VALUES(category_id),
  price_amount = VALUES(price_amount),
  currency = VALUES(currency),
  status = VALUES(status);

INSERT INTO promotions (
  id, code, name, description, scope, priority, stackable, exclusive, stop_processing,
  conflict_group, status, starts_at, ends_at, version
)
VALUES
  (1, 'ITEM1_10_PERCENT', 'Product 1 Discount 10%', 'Product 1 gets 10% off', 'ITEM', 10, TRUE, FALSE, FALSE, 'PRODUCT_DISCOUNT', 'ACTIVE', '2026-01-01 00:00:00.000', '2026-12-31 23:59:59.999', 1),
  (2, 'ITEM2_MINUS_100', 'Product 2 Discount 100 THB', 'Product 2 gets 100 THB off', 'ITEM', 10, TRUE, FALSE, FALSE, 'PRODUCT_DISCOUNT', 'ACTIVE', '2026-01-01 00:00:00.000', '2026-12-31 23:59:59.999', 1)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  scope = VALUES(scope),
  priority = VALUES(priority),
  stackable = VALUES(stackable),
  exclusive = VALUES(exclusive),
  stop_processing = VALUES(stop_processing),
  conflict_group = VALUES(conflict_group),
  status = VALUES(status),
  starts_at = VALUES(starts_at),
  ends_at = VALUES(ends_at),
  version = version + 1;

DELETE FROM promotion_targets WHERE promotion_id IN (1, 2);
DELETE FROM promotion_actions WHERE promotion_id IN (1, 2);

INSERT INTO promotion_targets (promotion_id, target_type, target_id, target_value)
VALUES
  (1, 'PRODUCT', 1, NULL),
  (2, 'PRODUCT', 2, NULL);

INSERT INTO promotion_actions (promotion_id, action_type, value_amount, value_basis_points, value_json, max_discount_amount, applies_to)
VALUES
  (1, 'PERCENTAGE_DISCOUNT', NULL, 1000, NULL, NULL, 'ITEM'),
  (2, 'FIXED_AMOUNT_DISCOUNT', 10000, NULL, NULL, NULL, 'ITEM');
