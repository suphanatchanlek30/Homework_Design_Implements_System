# Database

The canonical schema is `database/schema.sql`.

Main design decisions:

- MySQL 8.0+ with InnoDB.
- Money is `BIGINT` in minor units.
- Percentage is `value_basis_points`, not decimal percent.
- Promotions are rule-based: `promotions` + `promotion_targets` + `promotion_conditions` + `promotion_actions`.
- `promotion_calculation_logs` stores applied/skipped/snapshot JSON for audit and replay.
- `idempotency_keys` supports safe confirm-order requests.

Important indexes:

- `idx_promotions_active_window`: filter active promotions by status/date.
- `idx_promotions_active_sort`: sort active promotions deterministically.
- `idx_promotion_targets_lookup`: match product/category/cart targets quickly.
- `idx_promotion_conditions_promotion_id`: preload conditions per promotion.
- `idx_promotion_actions_promotion_id`: preload actions per promotion.
- `idx_promotion_usages_promo_user`: validate per-user usage limits.
- `idx_calculation_logs_request_id`: trace and debug by request ID.
- `idx_orders_user_status_created`: list customer orders efficiently.

Recommended runtime query pattern:

```sql
-- Load products once.
SELECT *
FROM products
WHERE id IN (...)
  AND status = 'ACTIVE';

-- Load active promotions once, then preload/attach targets, conditions, actions.
SELECT *
FROM promotions
WHERE status = 'ACTIVE'
  AND starts_at <= NOW(3)
  AND ends_at >= NOW(3)
ORDER BY scope, priority, created_at, id;
```

Avoid querying promotions inside item loops or conditions/actions inside promotion loops.
