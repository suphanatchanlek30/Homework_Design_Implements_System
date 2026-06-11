# Database

source of truth ของฐานข้อมูลคือ `database/schema.sql`

## ภาพรวม

- ใช้ MySQL 8.0+ และ InnoDB
- เงินเก็บเป็น `BIGINT` หน่วยย่อย
- เปอร์เซ็นต์เก็บเป็น `value_basis_points`
- promotion ถูกแยกเป็น `promotions` + `promotion_targets` + `promotion_conditions` + `promotion_actions`
- `promotion_calculation_logs` เก็บ applied/skipped/snapshot สำหรับ audit
- schema มีทั้ง `orders.idempotency_key` และ table `idempotency_keys`
- flow `orders/confirm` ในโค้ดปัจจุบันเช็ก idempotency หลักผ่าน `orders.idempotency_key`

## การ init

ถ้าใช้ Docker:

```bash
docker compose up --build
```

MySQL จะ init จาก:

1. `database/schema.sql`
2. `database/seed.sql`

ถ้าต้องการล้าง DB แล้ว seed ใหม่:

```bash
docker compose down -v
docker compose up --build
```

## Query pattern ที่ระบบใช้จริง

โหลดสินค้า:

```sql
SELECT *
FROM products
WHERE id IN (...)
  AND status = 'ACTIVE';
```

โหลด promotion active:

```sql
SELECT *
FROM promotions
WHERE status = 'ACTIVE'
  AND starts_at <= NOW(3)
  AND ends_at >= NOW(3)
ORDER BY scope, priority, created_at, id;
```

จากนั้น preload:

- `promotion_targets`
- `promotion_conditions`
- `promotion_actions`

## จุดที่ควรรู้

- engine ปัจจุบันใช้ `conflict_group` และ `stop_processing` จริง
- field อย่าง `stackable`, `exclusive`, `operator`, `logical_operator` ยังไม่ถูกใช้เต็มรูปแบบใน calculation loop
- schema รองรับ action type บางตัวมากกว่าที่ registry implement จริง
