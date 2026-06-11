# Promotion Engine Homework

โปรเจกต์นี้เป็นระบบคำนวณราคาสุทธิของคำสั่งซื้อด้วย `Go`, `Fiber`, `Gorm`, `MySQL` และ promotion engine แบบ `data-driven`

ระบบถูกออกแบบมาเพื่อรองรับโจทย์หลัก 3 เรื่อง:

1. รองรับโปรโมชั่นหลายแบบและโปรโมชั่นซ้อนกัน
2. เพิ่มโปรโมชั่นใหม่โดยไม่กระทบ logic เดิม
3. ขยาย action/condition ใหม่ได้โดยไม่ต้องรื้อ flow คำนวณหลัก

## ภาพรวมระบบ

แนวคิดหลักของระบบนี้คือ:

- เก็บ promotion เป็นข้อมูลในฐานข้อมูล ไม่ hardcode promotion รายตัวใน controller
- แยก `promotion` ออกเป็น `header + targets + conditions + actions`
- ใช้ `pricing service` เป็นตัว orchestration
- ใช้ `promotion engine` เป็นตัวคำนวณจริง
- ใช้ `registry` สำหรับ map `actionType` และ `conditionType` ไปยัง handler
- เก็บผล calculation เป็น audit log เพื่อ replay และ debug ได้

flow หลักของระบบ:

```text
HTTP Handler -> Service -> Repository -> MySQL
                       -> Promotion Engine
```

## Tech Stack

- `Go 1.24.5`
- `Fiber` สำหรับ HTTP API
- `Gorm` สำหรับ ORM และ transaction
- `MySQL 8.0+`
- `Docker Compose` สำหรับ local runtime

## สิ่งที่โปรเจกต์นี้ทำได้

- จัดการ `categories`
- จัดการ `products`
- จัดการ `promotions`
- preview ราคาด้วย `pricing/calculate`
- explain ลำดับการ apply/skip promotion ด้วย `pricing/explain`
- confirm order ด้วย idempotency
- เก็บ `promotion usages`
- เก็บ `calculation logs` สำหรับ audit และ replay

## Design Pattern ที่ใช้

- `Service Layer`
- `Repository Pattern`
- `Rule Engine`
- `Strategy Pattern`
- `Registry Pattern`
- `Audit Log / Snapshot`

## โครงสร้างข้อมูล promotion

promotion 1 ตัวถูกเก็บแยกเป็น:

- `promotions` ข้อมูลหลักของ promo
- `promotion_targets` เป้าหมาย เช่น product, category, cart
- `promotion_conditions` เงื่อนไข เช่น coupon code, minimum order
- `promotion_actions` วิธีลดราคา เช่น fixed amount, percentage

แนวทางนี้ทำให้:

- เพิ่ม promotion แบบเดิมได้ด้วยการเพิ่ม data
- ไม่ต้องแก้ core engine ทุกครั้งที่ business ขอ promo ใหม่
- รองรับ promotion ที่ซับซ้อนขึ้นในอนาคตผ่าน `value_json`

## ลำดับการคำนวณ promotion

engine คำนวณ promotion ตามลำดับนี้:

1. `ITEM`
2. `CART`
3. `COUPON`
4. `SHIPPING`

ถ้าอยู่ใน scope เดียวกัน จะ sort ต่อด้วย:

1. `priority`
2. `created_at`
3. `id`

กติกาที่มีผลจริงใน implementation ปัจจุบัน:

- ใช้เฉพาะ promotion ที่ `ACTIVE` และอยู่ในช่วงเวลาใช้งาน
- เช็ก `target` ก่อน
- เช็ก `condition` ถัดมา
- ถ้า `conflict_group` ชนกับ promo ที่ apply ไปแล้ว จะถูก skip
- ถ้า `stop_processing = true` จะหยุดคำนวณหลัง promo นั้น

หมายเหตุ:

- field อย่าง `stackable` และ `exclusive` มีอยู่ใน schema/model แต่ยังไม่ได้ถูกใช้เต็มรูปแบบใน engine ปัจจุบัน
- `operator` และ `logical_operator` ใน condition ยังมีไว้รองรับ design มากกว่า behavior เต็มรูปแบบ

## เอกสารที่ควรอ่าน

- [Docs/README.md](Docs/README.md) ดัชนีเอกสารทั้งหมด
- [Docs/ARCHITECTURE.md](Docs/ARCHITECTURE.md) โครงสร้างระบบ, design pattern, flow คำนวณ, table design
- [Docs/API_TESTING_GUIDE.md](Docs/API_TESTING_GUIDE.md) เส้นเทสหลักสำหรับ Postman และ proof ตามโจทย์
- [database/README.md](database/README.md) รายละเอียด schema และ query pattern ของฐานข้อมูล

## โครงสร้างโปรเจกต์

- `cmd/server` จุดเริ่มต้นของ API server
- `cmd/seed` คำสั่งสำหรับรัน seed/schema
- `database/schema.sql` schema หลักของระบบ
- `database/seed.sql` seed data เริ่มต้น
- `internal/config` โหลด config และ environment
- `internal/database` connection ของ MySQL และ Gorm
- `internal/dto` request/response DTO
- `internal/handler` HTTP handlers
- `internal/middleware` middleware เช่น `X-Request-ID`
- `internal/model` Gorm models
- `internal/promotion` promotion engine และ registry
- `internal/repository` DB access layer
- `internal/service` business logic
- `internal/seed` helper สำหรับรัน SQL seed
- `Docs/` เอกสารใช้งานและออกแบบ

## Prerequisites

ต้องมีอย่างน้อย:

- `Docker Desktop` หรือ `Docker Engine + Docker Compose`
- `Go 1.24.5` ถ้าจะรันแบบ local

## Environment Variables

สร้างไฟล์ `.env` จาก `.env.example`

ค่าตั้งต้นใน `.env.example`:

```env
APP_PORT=3000

MYSQL_ROOT_PASSWORD=rootpassword
MYSQL_DATABASE=promotion_engine
MYSQL_USER=promotion
MYSQL_PASSWORD=promotion123
MYSQL_HOST_PORT=3307

DB_HOST=mysql
DB_PORT=3307
DB_NAME=promotion_engine
DB_USER=promotion
DB_PASSWORD=promotion123

TZ=Asia/Bangkok
```

ความหมายหลัก:

- `APP_PORT` พอร์ตของ API
- `MYSQL_*` ใช้กับ Docker Compose
- `DB_*` ใช้กับ Go app/seed command
- `TZ` timezone ของ container/runtime

## วิธีรันแบบ Docker

นี่คือวิธีที่แนะนำที่สุดสำหรับโปรเจกต์นี้

1. สร้าง `.env` จาก `.env.example`
2. รัน:

```bash
docker compose up --build
```

3. เมื่อ container พร้อม:

- API จะอยู่ที่ `http://localhost:3000`
- MySQL จะอยู่ที่ `localhost:3307`

4. เช็ก health:

```bash
curl http://localhost:3000/api/v1/healthz
curl http://localhost:3000/api/v1/readyz
```

ถ้าพร้อมควรได้:

```json
{ "status": "UP" }
```

และ

```json
{ "status": "READY", "mysql": "UP" }
```

## วิธีรันแบบ Local

ถ้ามี MySQL พร้อมอยู่แล้วและค่าฐานข้อมูลใน `.env` ถูกต้อง:

```bash
go run ./cmd/server
```

หมายเหตุ:

- server จะโหลด `.env` อัตโนมัติ
- ถ้ารัน app บนเครื่อง local ให้แน่ใจว่า `DB_HOST` และ `DB_PORT` ชี้ไปยัง MySQL จริง
- ถ้ารัน app ใน Docker Compose ตัว app container จะ override `DB_HOST=mysql` และ `DB_PORT=3306`

## การ Seed ฐานข้อมูล

### แบบอัตโนมัติผ่าน Docker

MySQL official image จะรันไฟล์ SQL ใน `database/` อัตโนมัติเมื่อ volume ยังว่าง

ลำดับคือ:

1. `database/schema.sql`
2. `database/seed.sql`

### ล้างฐานข้อมูลแล้ว seed ใหม่

ถ้าต้องการ reset ใหม่ทั้งก้อน:

```bash
docker compose down -v
docker compose up --build
```

### Seed ซ้ำบน schema เดิม

```bash
go run ./cmd/seed
```

### รัน schema + seed ผ่าน Go command

```bash
go run ./cmd/seed --schema database/schema.sql --seed database/seed.sql
```

## Seed Data เริ่มต้น

หลัง reset DB ใหม่ ระบบจะมีข้อมูลตั้งต้นอย่างน้อย:

- `Product 1`
- `Product 2`
- promotion seed:
  - `ITEM1_10_PERCENT`
  - `ITEM2_MINUS_100`

ข้อมูลชุดนี้ใช้เป็น baseline สำหรับ test flow ของ pricing และ promotion stacking

## API ที่มีตอนนี้

### Health

- `GET /`
- `GET /api/v1/healthz`
- `GET /api/v1/readyz`

### Categories

- `POST /api/v1/categories`
- `GET /api/v1/categories`
- `GET /api/v1/categories/{categoryId}`
- `PATCH /api/v1/categories/{categoryId}`

### Products

- `POST /api/v1/products`
- `GET /api/v1/products`
- `GET /api/v1/products/{productId}`
- `PATCH /api/v1/products/{productId}`

### Promotions

- `POST /api/v1/promotions`
- `GET /api/v1/promotions`
- `GET /api/v1/promotions/{promotionId}`
- `PUT /api/v1/promotions/{promotionId}`
- `PATCH /api/v1/promotions/{promotionId}`
- `POST /api/v1/promotions/{promotionId}/validate`
- `POST /api/v1/promotions/{promotionId}/activate`
- `POST /api/v1/promotions/{promotionId}/deactivate`
- `GET /api/v1/promotions/{promotionId}/usages`

### Pricing

- `POST /api/v1/pricing/calculate`
- `POST /api/v1/pricing/explain`

### Orders

- `POST /api/v1/orders/confirm`
- `GET /api/v1/orders`
- `GET /api/v1/orders/{orderId}`

### Audit

- `GET /api/v1/calculation-logs`
- `GET /api/v1/calculation-logs/{calculationId}`
- `POST /api/v1/calculation-logs/{calculationId}/replay`

## วิธีทดสอบแบบเร็ว

เส้นทดสอบหลักที่แนะนำ:

1. เช็ก `healthz` และ `readyz`
2. ดู baseline promotions
3. สร้าง promotion ใหม่ 2-3 ตัว
4. `validate` และ `activate`
5. ยิง `pricing/explain` เพื่อดู `appliedPromotions` และ `skippedPromotions`
6. ยิง `pricing/calculate` เพื่อยืนยันตัวเลข
7. ยิง `orders/confirm` เพื่อยืนยัน business flow จริง
8. ดู `orders`, `promotion usages`, `calculation logs`

รายละเอียด Postman pipeline อยู่ใน [Docs/API_TESTING_GUIDE.md](Docs/API_TESTING_GUIDE.md)

## ตัวอย่างแนวคิดการใช้งาน

โจทย์ตัวอย่าง:

- สินค้า 1 ลด 10%
- สินค้า 2 ลด 100 บาท
- ทั้งตะกร้าลด 5%
- ใช้ coupon แล้วลดเพิ่มอีก 7%

ระบบจะ:

1. apply item promotion ก่อน
2. ใช้ยอดใหม่ไปคำนวณ cart promotion
3. ใช้ยอดหลัง cart ไปคำนวณ coupon promotion
4. เก็บ promo ที่ apply และ promo ที่ skip พร้อมเหตุผล

นี่คือจุดสำคัญที่ใช้ตอบโจทย์เรื่อง `promotion stacking` และ `correctness`

## การเชื่อมต่อฐานข้อมูลเพื่อตรวจข้อมูล

ถ้าใช้ Docker Compose:

```bash
docker compose exec mysql mysql -u${MYSQL_USER} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE}
```

ตัวอย่าง query:

```sql
SELECT * FROM products;
SELECT * FROM promotions;
SELECT * FROM promotion_actions;
SELECT * FROM promotion_conditions;
SELECT * FROM promotion_calculation_logs;
```

## จุดที่ควรรู้เกี่ยวกับ implementation ปัจจุบัน

- `Request ID` middleware มีแล้ว
- `Auth middleware` ยังไม่มี
- `pricing/explain` ใช้ logic เดียวกับ `pricing/calculate` แต่เก็บ trace/audit เพิ่ม
- `orders/confirm` จะ recalculate ก่อนบันทึก order ทุกครั้ง
- `Idempotency-Key` เป็น required header ของ `orders/confirm`
- calculation log ถูกเก็บทุกครั้งที่ calculate/explain สำเร็จ

## ถ้าจะขยายระบบต่อ

ถ้าจะเพิ่ม promotion แบบเดิม:

- เพิ่มผ่าน API/DB ได้เลย ถ้าใช้ action/condition ที่ระบบรองรับอยู่แล้ว

ถ้าจะเพิ่ม promotion แบบใหม่จริง:

- เพิ่ม action handler หรือ condition handler ใน `internal/promotion/registry.go`
- เพิ่ม logic ที่เกี่ยวข้องใน engine/validation
- ถ้าต้อง persist type ใหม่ใน DB อาจต้องขยาย enum ใน schema ด้วย

## เอกสารเสริม

- [Docs/README.md](Docs/README.md)
- [Docs/ARCHITECTURE.md](Docs/ARCHITECTURE.md)
- [Docs/API_TESTING_GUIDE.md](Docs/API_TESTING_GUIDE.md)
- [database/README.md](database/README.md)
