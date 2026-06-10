# Project Structure

This project follows the Promotion Engine documents in `Docs/`.

```text
cmd/
  server/                 Application entrypoint
database/
  schema.sql              Canonical MySQL schema with indexes
  seed.sql                Demo data for Product 1 and Product 2 promotions
migrations/
  001_create_promotion_engine_schema.sql
internal/
  config/                 Environment and application configuration
  database/               MySQL/GORM connection and migration bootstrap
  dto/                    Request/response DTOs
  handler/                Fiber handlers
  middleware/             Request ID, auth, validation, logging, recover
  model/                  GORM models
  promotion/              Rule-based Promotion Engine
    strategy/             Promotion strategy implementations
  repository/             GORM repositories
  seed/                   Application seed helpers
  service/                Pricing, order, promotion admin use cases
tests/
  unit/
  integration/
  api/
```

Layer dependency:

```text
Handler -> Service -> Promotion Engine -> Strategy
        -> Repository -> MySQL
```

Rules to keep:

- Product price must be loaded server-side.
- Money uses `BIGINT` minor units.
- Percentage uses `value_basis_points`.
- Calculation order is `ITEM -> CART -> COUPON -> SHIPPING -> ROUNDING`.
- Promotion sort order is `scope_order ASC -> priority ASC -> created_at ASC -> id ASC`.
- Calculate is preview only. Confirm must recalculate and consume usage in a transaction.
