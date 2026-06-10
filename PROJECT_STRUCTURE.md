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
  service/                Category, product, promotion, pricing, order use cases
test/
  unit/
    handler/
    promotion/
    service/
  integration/
  api/
```

Layer dependency:

```text
Handler -> Service -> Promotion Engine -> Strategy
        -> Repository -> MySQL
```

Current diagram-aligned API groups:

```text
Health    -> /api/v1/healthz, /api/v1/readyz
Catalog   -> /api/v1/categories, /api/v1/products
Promotion -> /api/v1/promotions/*
Pricing   -> /api/v1/pricing/*
Order     -> /api/v1/orders/*
Audit     -> /api/v1/calculation-logs/*
```

Rules to keep:

- Product price must be loaded server-side.
- Money uses `BIGINT` minor units.
- Percentage uses `value_basis_points`.
- Calculation order is `ITEM -> CART -> COUPON -> SHIPPING -> ROUNDING`.
- Promotion sort order is `scope_order ASC -> priority ASC -> created_at ASC -> id ASC`.
- Calculate is preview only. Confirm must recalculate and consume usage in a transaction.
