# API Testing Guide

คู่มือนี้อธิบายวิธีทดสอบ API ที่มีอยู่จริงในโปรเจกต์นี้แบบละเอียด โดยอ้างอิงจากโค้ดปัจจุบัน ไม่ใช่แค่สเปกในเอกสาร

Endpoints ที่มีตอนนี้:
- `GET /api/v1/healthz`
- `GET /api/v1/readyz`
- `POST /api/v1/categories`
- `GET /api/v1/categories`
- `GET /api/v1/categories/{categoryId}`
- `PATCH /api/v1/categories/{categoryId}`
- `POST /api/v1/products`
- `GET /api/v1/products`
- `GET /api/v1/products/{productId}`
- `PATCH /api/v1/products/{productId}`
- `POST /api/v1/promotions`
- `GET /api/v1/promotions`
- `GET /api/v1/promotions/{promotionId}`
- `PUT /api/v1/promotions/{promotionId}`
- `PATCH /api/v1/promotions/{promotionId}`
- `POST /api/v1/promotions/{promotionId}/validate`
- `POST /api/v1/promotions/{promotionId}/activate`
- `POST /api/v1/promotions/{promotionId}/deactivate`
- `GET /api/v1/promotions/{promotionId}/usages`
- `POST /api/v1/pricing/calculate`
- `POST /api/v1/pricing/explain`
- `POST /api/v1/orders/confirm`
- `GET /api/v1/orders`
- `GET /api/v1/orders/{orderId}`
- `GET /api/v1/calculation-logs`
- `GET /api/v1/calculation-logs/{calculationId}`
- `POST /api/v1/calculation-logs/{calculationId}/replay`

---

## 0) Coverage Matrix

ตารางนี้ตอบคำถามว่า “API ไหนช่วยพิสูจน์เงื่อนไขโจทย์ข้อไหน”

| เงื่อนไขโจทย์ | API ที่ใช้เทส | สิ่งที่พิสูจน์ได้ |
|---|---|---|
| promotion ซ้อนกันต้องเรียงยังไง | `POST /pricing/calculate`, `POST /pricing/explain` | ลำดับ `ITEM -> CART -> COUPON -> SHIPPING`, sort by `priority` / `createdAt` / `id` |
| เพิ่ม promotion โดยไม่กระทบ logic เดิม | `POST /promotions`, `PUT /promotions/{id}`, `PATCH /promotions/{id}`, `POST /promotions/{id}/validate` | เพิ่ม rule data หรือ strategy ใหม่โดย core flow ไม่ต้องแก้ |
| เพิ่ม promotion ใหม่ที่ไม่เคยมีมาก่อน | `POST /promotions`, `POST /pricing/explain` | เพิ่ม action/condition ใหม่ผ่าน registry / rule data |
| design pattern | โค้ด + `POST /pricing/explain` | Rule Engine + Strategy + Repository |
| table design ยืดหยุ่น | `POST /promotions`, `GET /promotions/{id}` | data แยกเป็น promotion / target / condition / action |
| คำนวณโปรโมชั่นได้ถูกต้อง | `POST /pricing/calculate`, `POST /orders/confirm` | price server-side, stacking ถูกลำดับ, final total ไม่ติดลบ, confirm ต้องตรงกับ calculation |

---

## 1) วิธีเตรียม Postman

### Environment variables ที่แนะนำ
สร้าง Environment แล้วใส่ค่าเหล่านี้:

| Variable | Example |
|---|---|
| `baseUrl` | `http://localhost:3000/api/v1` |
| `requestId` | `7b0c8f35-0d3b-4d71-9b0b-9c7f4e3d2a11` |
| `idempotencyKey` | `e9a2d0f7-5e5f-4c9d-8b54-47f4d6f9f0b8` |
| `productId` | `1` |
| `categoryId` | `1` |

### Headers ที่ควรส่ง
| Header | Required | ใช้ทำอะไร |
|---|---:|---|
| `Content-Type: application/json` | Yes | บอกว่า body เป็น JSON |
| `X-Request-ID` | No | ใช้ trace request ระหว่าง debug |
| `Idempotency-Key` | Order confirm | ใช้กัน confirm ซ้ำและ payload mismatch |

> หมายเหตุ: ใน codebase นี้ยังไม่มี auth middleware จริง ดังนั้น `Authorization` ยังไม่ถูกบังคับใช้

### รูปแบบ Error Response
ทุก endpoint ใช้รูปแบบนี้เมื่อเกิด error:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "invalid request body"
  }
}
```

---

## 2) Health APIs

### 2.1 `GET /api/v1/healthz`
ใช้ตรวจว่า application process ยังรันอยู่ ไม่ได้ตรวจ dependency ภายนอก

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/healthz`
- Headers: ไม่จำเป็นต้องมี body

#### Expected Response
Status: `200 OK`

```json
{
  "status": "UP"
}
```

#### ใช้เมื่อไร
- ตรวจว่า container / process ยังตอบสนองอยู่
- ใช้เป็น liveness probe

#### สิ่งที่ API นี้ไม่ทำ
- ไม่ ping MySQL
- ไม่ตรวจ Redis
- ไม่ตรวจ auth

---

### 2.2 `GET /api/v1/readyz`
ใช้ตรวจว่า service พร้อมรับ traffic หรือยัง โดยเช็ก MySQL

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/readyz`
- Headers: ไม่จำเป็นต้องมี body

#### Expected Response เมื่อพร้อม
Status: `200 OK`

```json
{
  "status": "READY",
  "dependencies": {
    "mysql": "UP"
  }
}
```

#### Expected Response เมื่อ DB ใช้ไม่ได้
Status: `503 Service Unavailable`

```json
{
  "status": "NOT_READY",
  "dependencies": {
    "mysql": "DOWN"
  }
}
```

#### พฤติกรรมจริงของโค้ด
- ถ้า `db == nil` จะตอบ `503`
- ถ้า `sqlDB.PingContext()` fail จะตอบ `503`
- timeout ของ ping คือ `2s`
- Redis ยังไม่ได้ต่อจริง ดังนั้นจะไม่ถูกส่งกลับมา

#### ใช้เมื่อไร
- ตรวจ readiness ตอน deploy
- ตรวจ dependency ก่อนปล่อย traffic เข้า service

---

## 3) Category APIs

### 3.1 `POST /api/v1/categories`
ใช้สร้าง category ใหม่

#### Request
- Method: `POST`
- Path: `{{baseUrl}}/categories`
- Headers:
  - `Content-Type: application/json`
  - `X-Request-ID` optional
  - `Idempotency-Key` optional

#### Body ตัวอย่าง
```json
{
  "name": "Electronics",
  "parentId": null,
  "status": "ACTIVE"
}
```

#### Validation ที่โค้ดทำจริง
- `name` ห้ามว่าง
- `status` ต้องเป็น `ACTIVE` หรือ `INACTIVE`
- ถ้าใส่ `parentId` ต้องมี category นั้นอยู่จริง
- ถ้า category เดิมชื่อเดียวกันภายใต้ parent เดียวกันมีอยู่แล้ว จะชน conflict

#### Expected Success Response
Status: `201 Created`

```json
{
  "id": 1,
  "name": "Electronics",
  "parentId": null,
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| name ว่าง | `400` | `INVALID_REQUEST` |
| status ไม่ถูกต้อง | `400` | `INVALID_REQUEST` |
| parentId ไม่พบ | `404` | `PARENT_CATEGORY_NOT_FOUND` |
| category ซ้ำ | `409` | `CATEGORY_ALREADY_EXISTS` |

#### Postman tests ที่ควรลอง
1. สร้าง parent category
2. สร้าง child category ด้วย `parentId`
3. ส่ง `name` ว่างเพื่อดู `400`
4. ส่ง `parentId` ที่ไม่มีอยู่จริงเพื่อดู `404`

---

### 3.2 `GET /api/v1/categories`
ใช้ list category แบบ pagination / filter / search

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/categories`

#### Query Parameters
| Parameter | Type | Default | คำอธิบาย |
|---|---|---:|---|
| `status` | string | none | กรอง `ACTIVE` / `INACTIVE` |
| `parentId` | number | none | กรองตาม parent |
| `keyword` | string | none | ค้นหาจากชื่อ |
| `page` | number | `1` | หน้าเริ่มต้น |
| `limit` | number | `10` | จำนวนรายการต่อหน้า |
| `sort` | string | `id desc` | field ที่อนุญาตเท่านั้น |

#### Sort whitelist
- `id`
- `name`
- `parent_id`
- `status`
- `created_at`
- `updated_at`

ตัวอย่าง:
- `sort=name asc`
- `sort=created_at desc`

#### Expected Response
Status: `200 OK`

```json
{
  "items": [
    {
      "id": 1,
      "name": "Electronics",
      "parentId": null,
      "status": "ACTIVE",
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```

#### Postman tests ที่ควรลอง
1. `GET /categories?page=1&limit=10`
2. `GET /categories?status=ACTIVE`
3. `GET /categories?keyword=Elect`
4. `GET /categories?sort=name asc`
5. `GET /categories?page=abc` เพื่อดู `400`

#### Validation ที่โค้ดทำจริง
- `page` ต้องเป็นตัวเลขและมากกว่า 0
- `limit` ต้องเป็นตัวเลขระหว่าง `1..100`
- `sort` ต้องอยู่ใน whitelist

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| query ไม่ถูกต้อง | `400` | `INVALID_QUERY_PARAMETER` |

#### Postman tests ที่ควรลอง
1. `GET /categories?page=1&limit=10`
2. `GET /categories?status=ACTIVE`
3. `GET /categories?keyword=Elect`
4. `GET /categories?sort=name asc`
5. `GET /categories?page=abc` เพื่อดู `400`

---

### 3.3 `GET /api/v1/categories/{categoryId}`
ใช้ดูรายละเอียด category รายตัว

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/categories/1`

#### Expected Response
Status: `200 OK`

```json
{
  "id": 1,
  "name": "Electronics",
  "parentId": null,
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| categoryId ไม่ใช่ตัวเลข | `400` | `INVALID_CATEGORY_ID` |
| ไม่พบ category | `404` | `CATEGORY_NOT_FOUND` |

#### หมายเหตุ
- soft-deleted record ไม่ควรถูกคืนกลับมา

---

### 3.4 `PATCH /api/v1/categories/{categoryId}`
ใช้แก้ไข category แบบ partial update

#### Request
- Method: `PATCH`
- Path: `{{baseUrl}}/categories/1`

#### Body ตัวอย่าง
```json
{
  "name": "Consumer Electronics",
  "parentId": null,
  "status": "ACTIVE"
}
```

#### Validation ที่โค้ดทำจริง
- `name` ถ้าส่งมา ห้ามว่าง
- `status` ถ้าส่งมา ต้องเป็น `ACTIVE` หรือ `INACTIVE`
- `parentId` ห้ามชี้มาที่ตัวเอง
- `parentId` ต้องมีอยู่จริง
- `parentId` ห้ามชี้ไปยังลูกหลานของตัวเอง
- ถ้าเปลี่ยนชื่อ/parent แล้วชนกับ category เดิมที่มีอยู่ จะคืน conflict

#### Expected Success Response
Status: `200 OK`

```json
{
  "id": 1,
  "name": "Consumer Electronics",
  "parentId": null,
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:05:00Z"
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| category ไม่พบ | `404` | `CATEGORY_NOT_FOUND` |
| parent ไม่พบ | `404` | `PARENT_CATEGORY_NOT_FOUND` |
| circular hierarchy | `422` | `INVALID_CATEGORY_HIERARCHY` |
| update conflict | `409` | `CATEGORY_UPDATE_CONFLICT` |
| payload ผิด | `400` | `INVALID_REQUEST` |

#### Postman tests ที่ควรลอง
1. เปลี่ยนชื่อ category
2. เปลี่ยน status เป็น `INACTIVE`
3. ตั้ง `parentId` เป็นตัวเองเพื่อดู `422`
4. ตั้ง `parentId` เป็นลูกหลานของตัวเองเพื่อดู `422`

---

## 4) Product APIs

### 4.1 `POST /api/v1/products`
ใช้สร้าง product ซึ่งเป็น source of truth ของราคา

#### Request
- Method: `POST`
- Path: `{{baseUrl}}/products`

#### Body ตัวอย่าง
```json
{
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 100000,
  "currency": "THB",
  "status": "ACTIVE"
}
```

#### Validation ที่โค้ดทำจริง
- `sku` ห้ามว่าง
- `name` ห้ามว่าง
- `categoryId` ต้องมีอยู่จริง
- `priceAmount` ต้องมากกว่าหรือเท่ากับ 0
- `currency` ต้องเป็น `THB`
- `status` ต้องเป็น `ACTIVE` หรือ `INACTIVE`
- `sku` ต้องไม่ซ้ำ

#### Expected Success Response
Status: `201 Created`

```json
{
  "id": 1,
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 100000,
  "currency": "THB",
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| category ไม่พบ | `404` | `CATEGORY_NOT_FOUND` |
| SKU ซ้ำ | `409` | `SKU_ALREADY_EXISTS` |
| price ติดลบ | `422` | `INVALID_PRICE_AMOUNT` |
| currency ไม่รองรับ | `422` | `UNSUPPORTED_CURRENCY` |
| payload ผิด | `400` | `INVALID_REQUEST` |

#### Postman tests ที่ควรลอง
1. สร้าง product ใหม่ด้วย category ที่มีอยู่
2. ส่ง SKU เดิมซ้ำเพื่อดู `409`
3. ส่ง `categoryId` ที่ไม่อยู่จริงเพื่อดู `404`
4. ส่ง `priceAmount: -1` เพื่อดู `422`
5. ส่ง `currency: "USD"` เพื่อดู `422`

---

### 4.2 `GET /api/v1/products`
ใช้ list product แบบ filter/pagination/search

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/products`

#### Query Parameters
| Parameter | Type | Default | คำอธิบาย |
|---|---|---:|---|
| `status` | string | none | กรอง `ACTIVE` / `INACTIVE` |
| `categoryId` | number | none | กรองตาม category |
| `sku` | string | none | กรอง SKU ตรงตัว |
| `keyword` | string | none | ค้นหาจากชื่อ |
| `page` | number | `1` | หน้าเริ่มต้น |
| `limit` | number | `10` | จำนวนรายการต่อหน้า |
| `sort` | string | `id desc` | field ที่อนุญาตเท่านั้น |

#### Sort whitelist
- `id`
- `sku`
- `name`
- `category_id`
- `price_amount`
- `status`
- `created_at`
- `updated_at`

ตัวอย่าง:
- `sort=price_amount desc`
- `sort=name asc`

#### Expected Response
Status: `200 OK`

```json
{
  "items": [
    {
      "id": 1,
      "sku": "PRODUCT-001",
      "name": "Product 1",
      "categoryId": 1,
      "priceAmount": 100000,
      "currency": "THB",
      "status": "ACTIVE",
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```

#### Validation ที่โค้ดทำจริง
- `page` ต้องเป็นตัวเลขและมากกว่า 0
- `limit` ต้องเป็นตัวเลขระหว่าง `1..100`
- `sort` ต้องอยู่ใน whitelist

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| query ไม่ถูกต้อง | `400` | `INVALID_QUERY_PARAMETER` |

#### Postman tests ที่ควรลอง
1. `GET /products?page=1&limit=10`
2. `GET /products?sku=PRODUCT-001`
3. `GET /products?categoryId=1&status=ACTIVE`
4. `GET /products?sort=price_amount desc`
5. `GET /products?sort=DROP TABLE` เพื่อดู `400`

---

### 4.3 `GET /api/v1/products/{productId}`
ใช้ดูรายละเอียด product รายตัว

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/products/1`

#### Expected Response
Status: `200 OK`

```json
{
  "id": 1,
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 100000,
  "currency": "THB",
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| productId ไม่ใช่ตัวเลข | `400` | `INVALID_PRODUCT_ID` |
| ไม่พบ product | `404` | `PRODUCT_NOT_FOUND` |

---

### 4.4 `PATCH /api/v1/products/{productId}`
ใช้แก้ไข product แบบ partial update

#### Request
- Method: `PATCH`
- Path: `{{baseUrl}}/products/1`

#### Body ตัวอย่าง
```json
{
  "priceAmount": 120000,
  "categoryId": 1,
  "currency": "THB",
  "status": "ACTIVE"
}
```

#### Validation ที่โค้ดทำจริง
- `priceAmount` ถ้าส่งมา ต้องไม่ติดลบ
- `categoryId` ถ้าส่งมา ต้องมีอยู่จริง
- `currency` ถ้าส่งมา ต้องเป็น `THB`
- `status` ถ้าส่งมา ต้องเป็น `ACTIVE` หรือ `INACTIVE`

#### Expected Success Response
Status: `200 OK`

```json
{
  "id": 1,
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 120000,
  "currency": "THB",
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:10:00Z"
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| product ไม่พบ | `404` | `PRODUCT_NOT_FOUND` |
| category ไม่พบ | `404` | `CATEGORY_NOT_FOUND` |
| price ติดลบ | `422` | `INVALID_PRICE_AMOUNT` |
| currency ไม่รองรับ | `422` | `UNSUPPORTED_CURRENCY` |
| payload ผิด | `400` | `INVALID_REQUEST` |

#### Postman tests ที่ควรลอง
1. เปลี่ยนราคา product
2. เปลี่ยน status เป็น `INACTIVE`
3. ส่ง `categoryId` ที่ไม่มีจริงเพื่อดู `404`
4. ส่ง `priceAmount: -1` เพื่อดู `422`

---

## 4) Promotion APIs

> หมายเหตุ: กลุ่ม API นี้คือ admin surface ของ promotion engine ตาม architecture diagram

### 4.1 `POST /api/v1/promotions`
สร้าง promotion ใหม่แบบ rule-based

#### Request
- Method: `POST`
- Path: `{{baseUrl}}/promotions`
- Headers:
  - `Content-Type: application/json`
  - `Idempotency-Key` optional

#### Body ตัวอย่าง
```json
{
  "code": "ITEM1_10_PERCENT",
  "name": "Product 1 Discount 10%",
  "description": "Product 1 gets 10% off",
  "scope": "ITEM",
  "priority": 10,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "conflictGroup": "PRODUCT_DISCOUNT",
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "maxUsage": null,
  "maxUsagePerUser": null,
  "targets": [
    { "targetType": "PRODUCT", "targetId": 1 }
  ],
  "conditions": [],
  "actions": [
    {
      "actionType": "PERCENTAGE_DISCOUNT",
      "valueBasisPoints": 1000,
      "appliesTo": "ITEM"
    }
  ]
}
```

#### Expected Success Response
Status: `201 Created`

```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Product 1 Discount 10%",
  "scope": "ITEM",
  "status": "DRAFT",
  "priority": 10,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 1,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```

#### Validation ที่โค้ดทำจริง
- `code` ต้อง unique
- `scope` ต้องเป็น `ITEM`, `CART`, `COUPON`, หรือ `SHIPPING`
- `priority` ต้องไม่ติดลบ
- `startsAt` ต้องมาก่อน `endsAt`
- `targets` ต้องไม่ว่าง
- `actions` ต้องไม่ว่าง
- action type ต้องอยู่ใน registry ที่รองรับ

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| code ซ้ำ | `409` | `PROMOTION_CODE_ALREADY_EXISTS` |
| config ไม่ถูกต้อง | `422` | `INVALID_PROMOTION_CONFIG` |
| action ไม่รองรับ | `422` | `ACTION_STRATEGY_NOT_SUPPORTED` |
| target ไม่ครบ | `422` | `TARGET_REQUIRED` |

#### Postman tests ที่ควรลอง
1. สร้าง promo ITEM สำหรับสินค้า 1
2. สร้าง promo CART แบบ fixed amount
3. ส่ง action type ที่ไม่รองรับเพื่อดู `422`
4. ส่ง target ว่างเพื่อดู `422`
5. สร้าง promo coupon-based เพื่อพิสูจน์ว่า rule data แยกจาก logic หลัก
6. สร้าง promo ซ้ำ code เดิมเพื่อดู `409`
7. สร้าง promo ใหม่ที่ใช้ target เดิมแต่ action ต่างกัน เพื่อยืนยันว่าเพิ่ม promotion ใหม่ได้โดยไม่กระทบ logic เดิม
8. สร้าง promo แบบซ้อนกันแล้วใช้ `priority` กับ `conflictGroup` ตรวจว่าระบบมีข้อมูลพอให้ engine ตัดสินลำดับได้

---

### 4.2 `GET /api/v1/promotions`
ใช้ค้นหา promotion แบบ summary โดยไม่ preload rule ทั้งหมด

#### Query Parameters
| Parameter | Type | ตัวอย่าง |
|---|---|---|
| `status` | enum | `DRAFT`, `ACTIVE`, `INACTIVE`, `EXPIRED` |
| `scope` | enum | `ITEM`, `CART`, `COUPON`, `SHIPPING` |
| `actionType` | string | `PERCENTAGE_DISCOUNT` |
| `code` | string | `ITEM1_10_PERCENT` |
| `activeAt` | RFC3339 | `2026-06-10T00:00:00Z` |
| `page` | number | `1` |
| `limit` | number | `10` |
| `sort` | string | `priority desc` |

#### Expected Response
Status: `200 OK`

```json
{
  "items": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "status": "ACTIVE",
      "priority": 10,
      "startsAt": "2026-01-01T00:00:00Z",
      "endsAt": "2026-12-31T23:59:59Z",
      "version": 1,
      "stackable": true,
      "exclusive": false,
      "stopProcessing": false,
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```

#### Validation ที่โค้ดทำจริง
- `status` และ `scope` ต้องอยู่ใน enum ที่กำหนด
- `activeAt` ต้องเป็น RFC3339
- `page` และ `limit` ต้องถูกต้อง
- `sort` ต้องอยู่ใน whitelist

#### Postman tests ที่ควรลอง
1. list promotion ที่ `status=ACTIVE`
2. list promotion ที่ `scope=ITEM`
3. list promotion ที่ `actionType=PERCENTAGE_DISCOUNT`
4. list promotion ตามเวลา `activeAt`
5. sort ตาม `priority desc`
6. ส่ง `status=BAD` เพื่อดู `400`
7. ส่ง `page=abc` เพื่อดู `400`
8. ตรวจว่า summary list ไม่คืน `targets`, `conditions`, `actions` เต็มชุด
9. list หลายหน้าเพื่อดู pagination response

---

### 4.3 `GET /api/v1/promotions/{promotionId}`
ดู promotion แบบเต็ม พร้อม `targets`, `conditions`, `actions`

#### Expected Response
Status: `200 OK`

```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Product 1 Discount 10%",
  "scope": "ITEM",
  "status": "ACTIVE",
  "priority": 10,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 1,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "description": "Product 1 gets 10% off",
  "conflictGroup": "PRODUCT_DISCOUNT",
  "targets": [],
  "conditions": [],
  "actions": []
}
```

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| id ไม่ถูกต้อง | `400` | `INVALID_PROMOTION_ID` |
| ไม่พบ promotion | `404` | `PROMOTION_NOT_FOUND` |

#### Postman tests ที่ควรลอง
1. ดู promotion ที่สร้างใหม่หลัง `POST`
2. ดู promotion ที่มี target/condition/action หลายตัว
3. ดู promotion ที่ inactive เพื่อยืนยันข้อมูลยัง load ได้
4. ส่ง id ที่ไม่มีอยู่จริงเพื่อดู `404`
5. compare response กับ list endpoint เพื่อยืนยันว่า detail โหลด relation ครบ
6. ตรวจว่า promotion ที่ soft-delete หรือ inactive ยังแยก state ได้ชัด

---

### 4.4 `PUT /api/v1/promotions/{promotionId}`
replace configuration ทั้งชุด และเพิ่ม version

#### Expected Behavior
- ใช้กับ draft หรือกรณีสร้าง version ใหม่
- ต้องส่ง `expectedVersion`
- ถ้า version ไม่ตรงจะคืน conflict

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| version ไม่ตรง | `409` | `PROMOTION_VERSION_CONFLICT` |
| config ไม่ครบ | `422` | `INVALID_PROMOTION_CONFIG` |

#### Postman tests ที่ควรลอง
1. replace config ทั้งชุดใน draft promotion
2. ส่ง `expectedVersion` ผิดเพื่อดู `409`
3. ลบ `actions` ออกเพื่อดู `422`
4. เปลี่ยน `code` ให้ชนของเดิมเพื่อดู `409`
5. replace config เพื่อเพิ่ม promotion variant ใหม่โดยไม่ต้องเปลี่ยน engine code
6. replace แล้วไปเรียก `pricing/explain` เพื่อตรวจว่า version ใหม่ถูกใช้จริง

---

### 4.5 `PATCH /api/v1/promotions/{promotionId}`
แก้ metadata บางส่วน เช่น `name`, `description`, `priority`, `startsAt`, `endsAt`

#### Constraint
- ห้าม patch `targets`, `conditions`, `actions`
- ต้องส่ง `expectedVersion`

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| field ไม่ patchable | `422` | `FIELD_NOT_PATCHABLE` |
| date range ผิด | `422` | `INVALID_PROMOTION_CONFIG` |
| version ไม่ตรง | `409` | `PROMOTION_VERSION_CONFLICT` |

#### Postman tests ที่ควรลอง
1. patch เฉพาะ `name` หรือ `priority`
2. patch `targets` เพื่อยืนยันว่าถูกบล็อก
3. patch `startsAt` / `endsAt` ให้ย้อนกันเพื่อดู `422`
4. patch ด้วย version ไม่ตรงเพื่อดู `409`
5. patch เฉพาะ metadata แล้วตรวจว่า version เพิ่มและ cache strategy ต้อง invalidate
6. patch เพื่อทดลอง business rule ที่ห้ามแก้ action/condition/target ผ่าน PATCH

---

### 4.6 `POST /api/v1/promotions/{promotionId}/validate`
run validation pipeline โดยไม่แก้ข้อมูล

#### Response
```json
{
  "valid": true,
  "errors": [],
  "warnings": []
}
```

#### Postman tests ที่ควรลอง
1. validate promotion ที่ config ถูกต้อง
2. validate promotion ที่ action ไม่รองรับ
3. validate promotion ที่ target ไม่ครบ
4. validate promotion ที่ช่วงเวลาไม่ถูกต้อง
5. validate ก่อน activate ทุกครั้ง
6. validate promotion ที่มี stacking policy หลายตัวเพื่อดู warnings
7. validate promotion ใหม่ที่ไม่เคยมีมาก่อนเพื่อพิสูจน์ว่า registry เป็นจุดขยาย

---

### 4.7 `POST /api/v1/promotions/{promotionId}/activate`
เปิดใช้งาน promotion หลัง validate ผ่าน

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| version ไม่ตรง | `409` | `PROMOTION_VERSION_CONFLICT` |
| config ไม่ผ่าน | `422` | `PROMOTION_CONFIGURATION_INVALID` |
| หมดอายุแล้ว | `422` | `PROMOTION_ALREADY_EXPIRED` |

#### Postman tests ที่ควรลอง
1. activate promotion ที่ validate ผ่าน
2. activate promotion ที่ config ผิด
3. activate promotion ที่หมดอายุแล้ว
4. activate ด้วย `expectedVersion` ผิด
5. activate แล้ว re-check status ผ่าน `GET /promotions/{promotionId}`
6. activate promotion แล้ว test `pricing/calculate` เพื่อดูว่า promotion ถูกใช้งานจริง

---

### 4.8 `POST /api/v1/promotions/{promotionId}/deactivate`
ปิด promotion เพื่อหยุดใช้กับ calculation ใหม่

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| version ไม่ตรง | `409` | `PROMOTION_VERSION_CONFLICT` |
| inactive อยู่แล้ว | `409` | `PROMOTION_ALREADY_INACTIVE` |

#### Postman tests ที่ควรลอง
1. deactivate promotion ที่ active อยู่
2. deactivate ซ้ำอีกครั้งเพื่อดู `409`
3. ส่ง reason เพื่อเก็บเหตุผลทางธุรกิจ
4. deactivate แล้วไปยิง `pricing/calculate` อีกครั้งเพื่อยืนยันว่า promotion ไม่ active แล้ว

---

### 4.9 `GET /api/v1/promotions/{promotionId}/usages`
ดู usage count และ discount รวมของ promotion

#### Query Parameters
- `userId`
- `from`
- `to`
- `page`
- `limit`

#### Response
```json
{
  "promotionId": 1,
  "totalUsage": 1,
  "totalDiscountAmount": 10000,
  "items": []
}
```

#### Postman tests ที่ควรลอง
1. list usage แบบไม่ filter
2. filter ด้วย `userId`
3. filter ด้วย `from` / `to`
4. filter ด้วย `page` และ `limit`
5. ส่ง `from=bad-date` เพื่อดู `400`
6. filter ด้วย `promotionId` เดียวกันหลาย request เพื่อดู usage สะสม
7. ตรวจว่า result เป็น aggregate จริง ไม่ใช่ mock

---

## 5) Pricing APIs

### 5.1 `POST /api/v1/pricing/calculate`
คำนวณราคาแบบ preview โดยโหลดราคาสินค้าจาก server และ apply promotion ที่ active

#### Request
- Method: `POST`
- Path: `{{baseUrl}}/pricing/calculate`
- Headers:
  - `Content-Type: application/json`
  - `X-Request-ID` optional

#### Body ตัวอย่าง
```json
{
  "userId": 1001,
  "currency": "THB",
  "couponCodes": [],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 },
    { "productId": 2, "quantity": 2 }
  ]
}
```

#### Expected Response
Status: `200 OK`

```json
{
  "calculationId": "calc-...",
  "originalTotal": 200000,
  "discountTotal": 10000,
  "finalTotal": 190000,
  "currency": "THB",
  "items": [
    {
      "productId": 1,
      "sku": "PRODUCT-001",
      "productName": "Product 1",
      "quantity": 1,
      "unitPrice": 100000,
      "originalAmount": 100000,
      "discountAmount": 10000,
      "finalAmount": 90000
    }
  ],
  "appliedPromotions": [],
  "skippedPromotions": []
}
```

#### Validation ที่โค้ดทำจริง
- `items` ห้ามว่าง
- `quantity` ต้องมากกว่า 0
- product ต้องมีอยู่จริง
- product ต้อง active
- currency ต้องตรงกันและตอนนี้รองรับ `THB`

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| items ว่าง | `422` | `EMPTY_ORDER_ITEMS` |
| quantity ผิด | `422` | `INVALID_QUANTITY` |
| product ไม่พบ | `404` | `PRODUCT_NOT_FOUND` |
| product inactive | `422` | `PRODUCT_INACTIVE` |
| currency ไม่ตรง | `422` | `CURRENCY_MISMATCH` |
| calculation fail | `500` | `CALCULATION_FAILED` |

#### Postman tests ที่ควรลอง
1. ใส่สินค้า 1 และสินค้า 2 ตาม seed
2. เปลี่ยน quantity เป็น 0 เพื่อดู `422`
3. ใส่ productId ที่ไม่มีจริงเพื่อดู `404`
4. ส่ง `currency: "USD"` เพื่อดู `422`
5. ใส่ productId เดิมซ้ำหลายแถวเพื่อยืนยันว่า system aggregate quantity ก่อนคำนวณ
6. ส่ง payload ที่มี promotion-enabled order เดียวกันแล้ว compare กับผลใน `pricing/explain`
7. ใช้สินค้าที่ราคาเยอะมากแล้วใส่ fixed discount เพื่อดูว่า final ไม่ติดลบ
8. ใส่สินค้า 1 กับสินค้า 2 เพื่อดูว่า promotion ที่ target ต่างกันถูก apply ตาม item จริง
9. ส่ง items หลายตัวแล้วตรวจลำดับการคำนวณ `ITEM -> CART -> COUPON -> SHIPPING`
10. เปลี่ยน `couponCodes` แล้วดูว่า coupon ถูกอ่านจาก request และไม่ได้ไปกระทบ item promotion
11. compare ผล `pricing/calculate` กับ `orders/confirm` ว่าค่า final ต้องตรงกันก่อน confirm
12. ใช้สินค้าที่ inactive เพื่อดูว่า validate reject ตาม contract

---

### 5.2 `POST /api/v1/pricing/explain`
เหมือน calculate แต่มีไว้ debug flow และ decision trace

#### ใช้เมื่อไร
- อยากดูว่าถูก skip เพราะอะไร
- อยาก debug stacking / target / condition flow

#### หมายเหตุ
- ใช้ logic เดียวกับ calculate
- ควรเปิดให้เฉพาะ admin/support หรือ internal เท่านั้นเมื่อใส่ auth จริง

#### Postman tests ที่ควรลอง
1. ส่ง payload เดียวกับ `pricing/calculate` แล้วเทียบผล output ทุก field
2. ดู decision trace / skipped promotion ที่ไม่เห็นใน `calculate`
3. ใช้ `includeLoadedPromotions` และ `includeSkippedRules` ตอนทดสอบ debug flow
4. ส่ง request จาก customer เพื่อยืนยันว่าในอนาคตจะต้องโดน role guard
5. ใช้ promotion ซ้อนหลายชั้นเพื่อดูว่า trace แสดงเหตุผลที่ถูก apply/skip ครบ
6. compare result กับ `pricing/calculate` เพื่อพิสูจน์ว่า logic เดียวกันแต่ explain เปิดรายละเอียดเพิ่ม

---

## 6) Order APIs

### 6.1 `POST /api/v1/orders/confirm`
ใช้ยืนยันคำสั่งซื้อจริงโดย recalculate ราคาอีกครั้ง แล้วบันทึก order + usage snapshot

#### Request
- Method: `POST`
- Path: `{{baseUrl}}/orders/confirm`
- Headers:
  - `Content-Type: application/json`
  - `Idempotency-Key` required
  - `X-Request-ID` optional

#### Body ตัวอย่าง
```json
{
  "calculationId": "calc-001",
  "acceptedFinalTotal": 135000,
  "userId": 1001,
  "currency": "THB",
  "couponCodes": [],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 }
  ]
}
```

#### พฤติกรรมจริงของโค้ด
- ต้องมี `Idempotency-Key`
- ต้องมี `calculationId`
- `items` ห้ามว่าง
- service จะเรียก pricing engine ซ้ำเพื่อยืนยันราคา
- ถ้า `acceptedFinalTotal` ไม่ตรงกับผลคำนวณ จะตอบ `409 ORDER_PRICE_CHANGED`
- ถ้าใช้ `Idempotency-Key` ซ้ำด้วย payload เดิม จะคืน order เดิม
- ถ้าใช้ key เดิมแต่ payload ต่าง จะถือเป็น confirmation failure
- ระบบจะสร้าง order status `CONFIRMED` พร้อมบันทึก `promotion usages`

#### Expected Response
Status: `201 Created`

```json
{
  "orderId": 1,
  "orderNo": "ORD-...",
  "userId": 1001,
  "status": "CONFIRMED",
  "currency": "THB",
  "originalTotal": 150000,
  "discountTotal": 15000,
  "finalTotal": 135000,
  "calculationId": "calc-...",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z",
  "items": [],
  "appliedPromotions": [],
  "skippedPromotions": [],
  "calculationSnapshot": {}
}
```

#### Postman tests ที่ควรลอง
1. เปิดดู log ที่เกิดจาก calculation จริง
2. compare snapshot กับ output จาก `pricing/calculate`
3. ตรวจว่ามี promotion version ที่ใช้งานจริงตอนคำนวณ ไม่ใช่ current version
4. ตรวจว่าอ่าน snapshot เดิม ไม่ใช่ไปอ่าน config ปัจจุบันของ promotion
5. เปิด log ที่ไม่ใช่ของตัวเองเพื่อเตรียมทดสอบ access control เมื่อ auth middleware ถูกเปิดใช้

#### Error Cases
| Case | Status | Error Code |
|---|---:|---|
| ไม่มี idempotency key | `400` | `IDEMPOTENCY_KEY_REQUIRED` |
| price ที่ยืนยันไม่ตรง | `409` | `ORDER_PRICE_CHANGED` |
| promotion usage เต็ม | `409` | `PROMOTION_USAGE_LIMIT_REACHED` |
| product หาย / inactive / currency mismatch | `404/422` | ตาม error ของ pricing |
| confirm ล้มเหลว | `500` | `ORDER_CONFIRMATION_FAILED` |

#### Postman tests ที่ควรลอง
1. preview ด้วย `POST /api/v1/pricing/calculate` ก่อน
2. ใช้ `calculationId` จากผล preview ไป confirm
3. ส่ง `Idempotency-Key` เดิมซ้ำเพื่อดูว่าได้ผลเดิม
4. เปลี่ยน `acceptedFinalTotal` เพื่อดู `409`
5. เปลี่ยน `items` หลัง preview เพื่อยืนยันว่า confirm ต้อง recalculate และ reject ถ้าราคาเปลี่ยน
6. confirm order ที่ promotion usage ใกล้เต็มเพื่อดู check usage limit
7. confirm แล้วไปดู order detail และ calculation log ว่า snapshot ถูก persist

---

### 6.2 `GET /api/v1/orders`
ใช้ list order แบบ pagination/filter

#### Query Parameters
| Parameter | Type | Default | คำอธิบาย |
|---|---|---:|---|
| `status` | string | none | `DRAFT`, `CONFIRMED`, `PAID`, `CANCELLED` |
| `userId` | number | none | กรองตาม user (ตอนนี้ยังไม่มี auth middleware จริง) |
| `createdFrom` | datetime | none | RFC3339 start time |
| `createdTo` | datetime | none | RFC3339 end time |
| `page` | number | `1` | หน้าเริ่มต้น |
| `limit` | number | `10` | จำนวนรายการต่อหน้า |
| `sort` | string | `id desc` | field ที่อยู่ใน whitelist |

#### Sort whitelist
- `id`
- `order_no`
- `user_id`
- `status`
- `original_total`
- `discount_total`
- `final_total`
- `created_at`
- `updated_at`

#### Validation ที่โค้ดทำจริง
- `status` ต้องอยู่ใน enum
- `createdFrom`/`createdTo` ต้องเป็น RFC3339
- `createdFrom` ต้องไม่มากกว่า `createdTo`
- `page` ต้องมากกว่า 0
- `limit` ต้องอยู่ระหว่าง `1..100`

#### Postman tests ที่ควรลอง
1. list order แบบไม่ใส่ filter เพื่อดู default result
2. filter ด้วย `status=CONFIRMED`
3. filter ด้วย `userId` เพื่อดู scope ตามผู้ใช้
4. filter ด้วยช่วงเวลา `createdFrom` / `createdTo`
5. ส่ง `status=INVALID` เพื่อดู validation
6. ส่ง `createdFrom` มากกว่า `createdTo` เพื่อดู `400`
7. ส่ง `limit=0` และ `limit=101` เพื่อดู reject
8. ส่ง sort ที่ไม่อยู่ whitelist เพื่อดู `400`
9. ถ้ามี auth middleware ภายหลัง ให้เทสว่า customer เห็นเฉพาะ order ของตัวเอง

---

### 6.3 `GET /api/v1/orders/{orderId}`
ใช้ดูรายละเอียด order + items + promotion snapshot

#### Request
- Method: `GET`
- Path: `{{baseUrl}}/orders/{orderId}`

#### Query ที่ใช้กับการทดสอบสิทธิ์
- `userId` optional

> หมายเหตุ: ตอนนี้ยังไม่มี auth middleware จริง จึงใช้ `userId` query เพื่อช่วยทดสอบ access control ในระดับโค้ด

#### Expected Error Cases
| Case | Status | Error Code |
|---|---:|---|
| orderId ไม่ถูกต้อง | `400` | `INVALID_ORDER_ID` |
| order ไม่พบ | `404` | `ORDER_NOT_FOUND` |
| user ไม่ตรงกับเจ้าของ order | `403` | `ORDER_ACCESS_DENIED` |

#### Postman tests ที่ควรลอง
1. ดู order ที่เพิ่ง confirm สำเร็จ
2. ดู order ที่ไม่มีอยู่จริงเพื่อดู `404`
3. ดู order ด้วย `userId` ที่ไม่ตรงกับ owner เพื่อยืนยัน access control logic
4. ตรวจว่า response มี items, totals, applied promotions และ calculation snapshot ครบ
5. ยืนยันว่า endpoint นี้ไม่ recalculate ตอนอ่าน

---

## 7) Error code reference

โค้ด error ที่ใช้จริงใน current implementation:

| Code | ความหมาย |
|---|---|
| `INVALID_REQUEST` | JSON body ผิด หรือ validation ผิด |
| `INVALID_QUERY_PARAMETER` | query string ผิดรูปแบบ |
| `INVALID_CATEGORY_ID` | categoryId path param ผิด |
| `INVALID_PRODUCT_ID` | productId path param ผิด |
| `CATEGORY_NOT_FOUND` | ไม่พบ category |
| `PARENT_CATEGORY_NOT_FOUND` | ไม่พบ parent category |
| `PRODUCT_NOT_FOUND` | ไม่พบ product |
| `CATEGORY_ALREADY_EXISTS` | category ซ้ำ |
| `CATEGORY_UPDATE_CONFLICT` | update แล้วชนกับข้อมูลเดิม |
| `SKU_ALREADY_EXISTS` | SKU ซ้ำ |
| `INVALID_PRICE_AMOUNT` | ราคาติดลบ |
| `UNSUPPORTED_CURRENCY` | currency ไม่รองรับ |
| `INVALID_CATEGORY_HIERARCHY` | hierarchy ผิด เช่น circular reference |
| `PROMOTION_CODE_ALREADY_EXISTS` | promo code ซ้ำ |
| `PROMOTION_VERSION_CONFLICT` | version ไม่ตรง |
| `INVALID_PROMOTION_CONFIG` | promotion config ผิด |
| `ACTION_STRATEGY_NOT_SUPPORTED` | action ที่ระบบยังไม่รองรับ |
| `TARGET_REQUIRED` | promotion ต้องมี target |
| `FIELD_NOT_PATCHABLE` | field นี้ patch ไม่ได้ |
| `PROMOTION_ALREADY_INACTIVE` | promotion inactive อยู่แล้ว |
| `PROMOTION_ALREADY_EXPIRED` | promotion หมดอายุแล้ว |
| `PROMOTION_CONFIGURATION_INVALID` | config ไม่ผ่าน validate ตอน activate |
| `PROMOTION_NOT_FOUND` | ไม่พบ promotion |
| `EMPTY_ORDER_ITEMS` | pricing request ไม่มี items |
| `INVALID_QUANTITY` | quantity ผิด |
| `PRODUCT_INACTIVE` | product inactive |
| `CURRENCY_MISMATCH` | currency ไม่ตรง |
| `CALCULATION_FAILED` | pricing engine ล้มเหลว
| `IDEMPOTENCY_KEY_REQUIRED` | confirm order ต้องมี idempotency key |
| `ORDER_PRICE_CHANGED` | ราคาที่ confirm ไม่ตรงกับผลคำนวณล่าสุด |
| `PROMOTION_USAGE_LIMIT_REACHED` | promotion usage เต็มแล้ว |
| `ORDER_CONFIRMATION_FAILED` | confirm order ล้มเหลวโดยรวม |
| `INVALID_ORDER_ID` | orderId path param ผิด |
| `ORDER_NOT_FOUND` | ไม่พบ order |
| `ORDER_ACCESS_DENIED` | ไม่มีสิทธิ์ดู order นี้ |
| `CALCULATION_LOG_NOT_FOUND` | ไม่พบ calculation log |
| `REPLAY_MODE_NOT_SUPPORTED` | mode replay ยังไม่รองรับ |
| `CALCULATION_REPLAY_FAILED` | replay calculation ล้มเหลว |

---

## 8) Audit APIs

### 8.1 `GET /api/v1/calculation-logs`
ใช้ค้นหา calculation log แบบ pagination และ filter จาก request/order/user/promotion/time

#### Query Parameters
| Parameter | Type | Default | คำอธิบาย |
|---|---|---:|---|
| `requestId` | string | none | request id จาก middleware/log |
| `orderId` | number | none | order ที่สร้างจาก calculation นี้ |
| `userId` | number | none | user ของ calculation |
| `promotionId` | number | none | promotions ที่ถูก apply |
| `createdFrom` | datetime | none | RFC3339 start time |
| `createdTo` | datetime | none | RFC3339 end time |
| `page` | number | `1` | หน้าเริ่มต้น |
| `limit` | number | `10` | จำนวนรายการต่อหน้า |
| `sort` | string | `created_at DESC` | sort field ที่ whitelist |

#### Expected Response
Status: `200 OK`

```json
{
  "items": [
    {
      "calculationId": "calc-001",
      "orderId": 1,
      "requestId": "req-001",
      "userId": 1001,
      "originalTotal": 150000,
      "discountTotal": 15000,
      "finalTotal": 135000,
      "appliedPromotionCount": 1,
      "skippedPromotionCount": 0,
      "createdAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```

### 8.2 `GET /api/v1/calculation-logs/{calculationId}`
ใช้ดู snapshot แบบเต็มของ calculation หนึ่งรายการ

#### Expected Response
Status: `200 OK`

```json
{
  "calculationId": "calc-001",
  "requestId": "req-001",
  "originalTotal": 150000,
  "discountTotal": 15000,
  "finalTotal": 135000,
  "appliedPromotionCount": 1,
  "skippedPromotionCount": 0,
  "appliedPromotions": [],
  "skippedPromotions": [],
  "calculationSnapshot": {}
}
```

### 8.3 `POST /api/v1/calculation-logs/{calculationId}/replay`
ใช้ replay จาก snapshot เดิมเพื่อพิสูจน์ผลโดยไม่สร้าง order และไม่ consume usage

#### Request
- Method: `POST`
- Path: `{{baseUrl}}/calculation-logs/{calculationId}/replay`
- Body ตัวอย่าง:
```json
{ "mode": "SNAPSHOT_CONFIG" }
```

#### Expected Response
Status: `200 OK`

```json
{
  "calculationId": "calc-001",
  "mode": "SNAPSHOT_CONFIG",
  "originalResult": {},
  "replayResult": {},
  "matched": true,
  "differences": []
}
```

#### Postman tests ที่ควรลอง
1. replay ด้วย snapshot เดิมแล้วต้องได้ผลเท่าเดิม
2. replay ด้วย mode ที่ไม่รองรับเพื่อดู `422`
3. compare originalResult กับ replayResult ว่าตรงกัน
4. ยืนยันว่า replay ไม่สร้าง order ใหม่
5. ยืนยันว่า replay ไม่ consume usage เพิ่ม

---

## 9) ค่าที่ควรรู้ก่อนทดสอบ

### Pagination defaults
- `page` default = `1`
- `limit` default = `10`
- `limit` มากกว่า `100` จะถูก reject

### Sort behavior
- sort ต้องเป็น field ที่ whitelist ไว้เท่านั้น
- ถ้าไม่ส่ง sort มา ระบบใช้ `id desc`

### Response field names
ดูชื่อ field ให้ตรงกับ JSON ที่ระบบส่งกลับจริง:
- Category response ใช้ `parentId`, `createdAt`, `updatedAt`
- Product response ใช้ `categoryId`, `priceAmount`, `createdAt`, `updatedAt`

### Health behavior
- `healthz` = liveness
- `readyz` = readiness ตรวจ MySQL

---

## 10) Suggested Postman flow

ถ้าจะเทสครบแบบลำดับที่ใช้งานจริง ให้ทำตามนี้:

1. เรียก `GET /api/v1/healthz`
2. เรียก `GET /api/v1/readyz`
3. สร้าง parent category ด้วย `POST /api/v1/categories`
4. สร้าง child category ด้วย `POST /api/v1/categories`
5. list category ด้วย `GET /api/v1/categories`
6. ดู category รายตัวด้วย `GET /api/v1/categories/{categoryId}`
7. update category ด้วย `PATCH /api/v1/categories/{categoryId}`
8. สร้าง product ด้วย `POST /api/v1/products`
9. list product ด้วย `GET /api/v1/products`
10. ดู product รายตัวด้วย `GET /api/v1/products/{productId}`
11. update product ด้วย `PATCH /api/v1/products/{productId}`
12. สร้าง promotion ด้วย `POST /api/v1/promotions`
13. list promotion ด้วย `GET /api/v1/promotions`
14. ดู promotion รายตัวด้วย `GET /api/v1/promotions/{promotionId}`
15. validate/activate/deactivate promotion
16. ยิง `POST /api/v1/pricing/calculate`
17. ยิง `POST /api/v1/pricing/explain`
18. confirm order ด้วย `POST /api/v1/orders/confirm`
19. list order ด้วย `GET /api/v1/orders`
20. ดู order รายตัวด้วย `GET /api/v1/orders/{orderId}`
21. list calculation log ด้วย `GET /api/v1/calculation-logs`
22. ดู calculation log รายตัวด้วย `GET /api/v1/calculation-logs/{calculationId}`
23. replay calculation ด้วย `POST /api/v1/calculation-logs/{calculationId}/replay`

---

## 11) หมายเหตุสำคัญ

- ปัจจุบันยังไม่มี auth middleware จริงใน codebase
- `Idempotency-Key` ของ order confirm ถูกบังคับใช้แล้ว และใช้ร่วมกับ request hash เพื่อกัน payload mismatch
- คู่มือนี้อิง behavior ที่โค้ดทำจริง ณ ตอนนี้ ไม่ได้เขียนตามสเปกในเอกสารทั้งหมดแบบอุดมคติ
- ถ้าต้องการให้ตรงสเปกมากขึ้น ขั้นต่อไปคือเพิ่ม auth, audit, middleware validation และ scope-based access control
- audit endpoints ใช้ snapshot ที่ persist ไว้ใน calculation log เป็น source of truth สำหรับ replay

---

## 12) สรุปว่าเทสไหนตอบโจทย์โจทย์นี้ได้

### 12.1 Promotion ซ้อนกันต้องเรียงยังไง
พิสูจน์ด้วย:
- `POST /api/v1/pricing/calculate`
- `POST /api/v1/pricing/explain`

สิ่งที่ต้องดู:
- ลำดับการประมวลผล `ITEM -> CART -> COUPON -> SHIPPING`
- promotion ที่ priority สูงกว่าถูก apply ก่อน
- promotion ที่ conflict ถูก skip ตาม `conflictGroup`, `exclusive`, `stopProcessing`
- `explain` ต้องบอกเหตุผลว่าเพราะอะไรตัวหนึ่งถูก apply และอีกตัวถูก skip

### 12.2 ถ้าต้องเพิ่ม promotion ใหม่โดยไม่กระทบ logic เดิม
พิสูจน์ด้วย:
- `POST /api/v1/promotions`
- `PUT /api/v1/promotions/{promotionId}`
- `PATCH /api/v1/promotions/{promotionId}`
- `POST /api/v1/promotions/{promotionId}/validate`

สิ่งที่ต้องดู:
- สร้าง promotion ใหม่ได้จาก data/config
- validate ผ่าน registry โดยไม่ต้องแก้ core pricing flow
- PATCH จำกัดเฉพาะ metadata ไม่แตะ rule logic
- PUT ใช้ version control เพื่อเปลี่ยน config แบบมีกรอบ

### 12.3 ถ้าต้องเพิ่ม promotion ใหม่ที่ไม่เคยมีมาก่อน
พิสูจน์ด้วย:
- `POST /api/v1/promotions`
- `POST /api/v1/promotions/{promotionId}/validate`
- `POST /api/v1/pricing/explain`

สิ่งที่ต้องดู:
- action/condition ใหม่ต้องผ่าน registry หรือ validation pipeline
- design ต้องรองรับ extension แบบ data-driven
- ถ้า registry ยังไม่รองรับ ต้อง reject แบบชัดเจน ไม่ปล่อยให้คำนวณผิดเงียบ ๆ

### 12.4 design pattern และ table design
สิ่งที่คู่มือนี้สะท้อนจากโค้ด:
- Rule Engine / Strategy Pattern: promotion action แยกเป็น registry และ strategy
- Repository Pattern: แยกชั้นอ่าน/เขียนจาก DB
- Versioned config: promotion เปลี่ยนผ่าน version ไม่แก้ของเดิมทิ้ง
- Normalized table design: promotion, target, condition, action, usage, calculation log แยกกัน

### 12.5 ขอบเขตที่เทสชุดนี้พิสูจน์ได้จริง
- คำนวณโปรโมชั่นถูกต้องบน server-side
- ตรวจ stacking และ rejection/skip logic
- ตรวจ idempotency ของ confirm order
- ตรวจ snapshot และ replay จาก audit log
- ตรวจว่า list/detail/use-case แยก payload ตามหน้าที่

---

## 13) Request/Response Reference ตามโค้ดจริง

ภาคผนวกนี้รวบรวมตัวอย่าง `request body`, `response body`, และ `error body` ที่อิงจาก DTO และ handler จริงของโปรเจกต์

### 13.1 Health

#### `GET /api/v1/healthz`
- Request body: ไม่มี
- Success `200 OK`
```json
{ "status": "UP" }
```
- Error: ไม่มีกรณี business error ใน handler นี้

#### `GET /api/v1/readyz`
- Request body: ไม่มี
- Success `200 OK`
```json
{
  "status": "READY",
  "dependencies": {
    "mysql": "UP"
  }
}
```
- Error `503 Service Unavailable`
```json
{
  "status": "NOT_READY",
  "dependencies": {
    "mysql": "DOWN"
  }
}
```

### 13.2 Category

#### `POST /api/v1/categories`
- Request body
```json
{
  "name": "Electronics",
  "parentId": null,
  "status": "ACTIVE"
}
```
- Success `201 Created`
```json
{
  "id": 1,
  "name": "Electronics",
  "parentId": null,
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```
- Error body ตัวอย่าง
```json
{ "error": { "code": "CATEGORY_ALREADY_EXISTS", "message": "..." } }
```

#### `GET /api/v1/categories`
- Request query example: `?status=ACTIVE&keyword=Elect&page=1&limit=10&sort=name asc`
- Success `200 OK`
```json
{
  "items": [
    {
      "id": 1,
      "name": "Electronics",
      "parentId": null,
      "status": "ACTIVE",
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```
- Error `400`
```json
{ "error": { "code": "INVALID_QUERY_PARAMETER", "message": "invalid page parameter" } }
```

#### `GET /api/v1/categories/{categoryId}`
- Request path example: `/categories/1`
- Success `200 OK`
```json
{
  "id": 1,
  "name": "Electronics",
  "parentId": null,
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```
- Error `400`
```json
{ "error": { "code": "INVALID_CATEGORY_ID", "message": "invalid category ID" } }
```
- Error `404`
```json
{ "error": { "code": "CATEGORY_NOT_FOUND", "message": "..." } }
```

#### `PATCH /api/v1/categories/{categoryId}`
- Request body
```json
{
  "name": "Consumer Electronics",
  "parentId": null,
  "status": "ACTIVE"
}
```
- Success `200 OK`
```json
{
  "id": 1,
  "name": "Consumer Electronics",
  "parentId": null,
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:05:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "INVALID_CATEGORY_HIERARCHY", "message": "..." } }
```
```json
{ "error": { "code": "CATEGORY_UPDATE_CONFLICT", "message": "..." } }
```

### 13.3 Product

#### `POST /api/v1/products`
- Request body
```json
{
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 100000,
  "currency": "THB",
  "status": "ACTIVE"
}
```
- Success `201 Created`
```json
{
  "id": 1,
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 100000,
  "currency": "THB",
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "SKU_ALREADY_EXISTS", "message": "..." } }
```
```json
{ "error": { "code": "UNSUPPORTED_CURRENCY", "message": "..." } }
```

#### `GET /api/v1/products`
- Request query example: `?status=ACTIVE&categoryId=1&sku=PRODUCT-001&page=1&limit=10&sort=price_amount desc`
- Success `200 OK`
```json
{
  "items": [
    {
      "id": 1,
      "sku": "PRODUCT-001",
      "name": "Product 1",
      "categoryId": 1,
      "priceAmount": 100000,
      "currency": "THB",
      "status": "ACTIVE",
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```
- Error `400`
```json
{ "error": { "code": "INVALID_QUERY_PARAMETER", "message": "invalid sort parameter" } }
```

#### `GET /api/v1/products/{productId}`
- Success `200 OK`
```json
{
  "id": 1,
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 100000,
  "currency": "THB",
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "INVALID_PRODUCT_ID", "message": "invalid product ID" } }
```
```json
{ "error": { "code": "PRODUCT_NOT_FOUND", "message": "..." } }
```

#### `PATCH /api/v1/products/{productId}`
- Request body
```json
{
  "priceAmount": 120000,
  "categoryId": 1,
  "currency": "THB",
  "status": "ACTIVE"
}
```
- Success `200 OK`
```json
{
  "id": 1,
  "sku": "PRODUCT-001",
  "name": "Product 1",
  "categoryId": 1,
  "priceAmount": 120000,
  "currency": "THB",
  "status": "ACTIVE",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:10:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "INVALID_PRICE_AMOUNT", "message": "..." } }
```
```json
{ "error": { "code": "CATEGORY_NOT_FOUND", "message": "..." } }
```

### 13.4 Promotion

#### `POST /api/v1/promotions`
- Request body
```json
{
  "code": "ITEM1_10_PERCENT",
  "name": "Product 1 Discount 10%",
  "description": "Product 1 gets 10% off",
  "scope": "ITEM",
  "priority": 10,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "conflictGroup": "PRODUCT_DISCOUNT",
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "maxUsage": null,
  "maxUsagePerUser": null,
  "targets": [
    { "targetType": "PRODUCT", "targetId": 1 }
  ],
  "conditions": [],
  "actions": [
    {
      "actionType": "PERCENTAGE_DISCOUNT",
      "valueBasisPoints": 1000,
      "appliesTo": "ITEM"
    }
  ]
}
```
- Success `201 Created`
```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Product 1 Discount 10%",
  "scope": "ITEM",
  "status": "DRAFT",
  "priority": 10,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 1,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "PROMOTION_CODE_ALREADY_EXISTS", "message": "..." } }
```
```json
{ "error": { "code": "TARGET_REQUIRED", "message": "..." } }
```

#### `GET /api/v1/promotions`
- Request query example: `?status=ACTIVE&scope=ITEM&actionType=PERCENTAGE_DISCOUNT&activeAt=2026-06-10T00:00:00Z&page=1&limit=10&sort=priority desc`
- Success `200 OK`
```json
{
  "items": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "status": "ACTIVE",
      "priority": 10,
      "startsAt": "2026-01-01T00:00:00Z",
      "endsAt": "2026-12-31T23:59:59Z",
      "version": 1,
      "stackable": true,
      "exclusive": false,
      "stopProcessing": false,
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```
- Error `400`
```json
{ "error": { "code": "INVALID_QUERY_PARAMETER", "message": "invalid status parameter" } }
```

#### `GET /api/v1/promotions/{promotionId}`
- Success `200 OK`
```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Product 1 Discount 10%",
  "scope": "ITEM",
  "status": "ACTIVE",
  "priority": 10,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 1,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "description": "Product 1 gets 10% off",
  "conflictGroup": "PRODUCT_DISCOUNT",
  "maxUsage": null,
  "maxUsagePerUser": null,
  "targets": [
    { "targetType": "PRODUCT", "targetId": 1, "targetValue": null }
  ],
  "conditions": [],
  "actions": [
    {
      "actionType": "PERCENTAGE_DISCOUNT",
      "valueAmount": null,
      "valueBasisPoints": 1000,
      "valueJson": null,
      "maxDiscountAmount": null,
      "appliesTo": "ITEM"
    }
  ]
}
```
- Error bodies
```json
{ "error": { "code": "INVALID_PROMOTION_ID", "message": "invalid promotion ID" } }
```
```json
{ "error": { "code": "PROMOTION_NOT_FOUND", "message": "..." } }
```

#### `PUT /api/v1/promotions/{promotionId}`
- Request body
```json
{
  "code": "ITEM1_10_PERCENT_V2",
  "name": "Product 1 Discount 10% V2",
  "description": "new config",
  "scope": "ITEM",
  "priority": 10,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "conflictGroup": "PRODUCT_DISCOUNT",
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "maxUsage": null,
  "maxUsagePerUser": null,
  "targets": [{ "targetType": "PRODUCT", "targetId": 1 }],
  "conditions": [],
  "actions": [{ "actionType": "PERCENTAGE_DISCOUNT", "valueBasisPoints": 1000, "appliesTo": "ITEM" }],
  "expectedVersion": 1
}
```
- Success `200 OK`
```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT_V2",
  "name": "Product 1 Discount 10% V2",
  "scope": "ITEM",
  "status": "DRAFT",
  "priority": 10,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 2,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "description": "new config",
  "conflictGroup": "PRODUCT_DISCOUNT",
  "maxUsage": null,
  "maxUsagePerUser": null,
  "targets": [
    { "targetType": "PRODUCT", "targetId": 1, "targetValue": null }
  ],
  "conditions": [],
  "actions": [
    {
      "actionType": "PERCENTAGE_DISCOUNT",
      "valueAmount": null,
      "valueBasisPoints": 1000,
      "valueJson": null,
      "maxDiscountAmount": null,
      "appliesTo": "ITEM"
    }
  ]
}
```
- Error bodies
```json
{ "error": { "code": "PROMOTION_VERSION_CONFLICT", "message": "..." } }
```
```json
{ "error": { "code": "INVALID_PROMOTION_CONFIG", "message": "..." } }
```

#### `PATCH /api/v1/promotions/{promotionId}`
- Request body
```json
{
  "name": "Updated Name",
  "priority": 20,
  "expectedVersion": 2
}
```
- Success `200 OK`
```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Updated Name",
  "scope": "ITEM",
  "status": "ACTIVE",
  "priority": 20,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 3,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "description": "Product 1 gets 10% off",
  "conflictGroup": "PRODUCT_DISCOUNT",
  "maxUsage": null,
  "maxUsagePerUser": null,
  "targets": [
    { "targetType": "PRODUCT", "targetId": 1, "targetValue": null }
  ],
  "conditions": [],
  "actions": [
    {
      "actionType": "PERCENTAGE_DISCOUNT",
      "valueAmount": null,
      "valueBasisPoints": 1000,
      "valueJson": null,
      "maxDiscountAmount": null,
      "appliesTo": "ITEM"
    }
  ]
}
```
- Error bodies
```json
{ "error": { "code": "FIELD_NOT_PATCHABLE", "message": "..." } }
```
```json
{ "error": { "code": "PROMOTION_VERSION_CONFLICT", "message": "..." } }
```

#### `POST /api/v1/promotions/{promotionId}/validate`
- Request body
```json
{ "expectedVersion": 2 }
```
- Success `200 OK`
```json
{
  "valid": true,
  "errors": [],
  "warnings": []
}
```
- Error bodies
```json
{ "error": { "code": "PROMOTION_VERSION_CONFLICT", "message": "..." } }
```
```json
{ "error": { "code": "PROMOTION_NOT_FOUND", "message": "..." } }
```

#### `POST /api/v1/promotions/{promotionId}/activate`
- Request body
```json
{ "expectedVersion": 2 }
```
- Success `200 OK`
```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Updated Name",
  "scope": "ITEM",
  "status": "ACTIVE",
  "priority": 20,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 3,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:10:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "PROMOTION_CONFIGURATION_INVALID", "message": "..." } }
```
```json
{ "error": { "code": "PROMOTION_ALREADY_EXPIRED", "message": "..." } }
```

#### `POST /api/v1/promotions/{promotionId}/deactivate`
- Request body
```json
{
  "expectedVersion": 3,
  "reason": "Campaign ended"
}
```
- Success `200 OK`
```json
{
  "promotionId": 1,
  "code": "ITEM1_10_PERCENT",
  "name": "Updated Name",
  "scope": "ITEM",
  "status": "INACTIVE",
  "priority": 20,
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "version": 4,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:15:00Z"
}
```
- Error bodies
```json
{ "error": { "code": "PROMOTION_ALREADY_INACTIVE", "message": "..." } }
```
```json
{ "error": { "code": "PROMOTION_VERSION_CONFLICT", "message": "..." } }
```

#### `GET /api/v1/promotions/{promotionId}/usages`
- Request query example: `?userId=1001&from=2026-06-01T00:00:00Z&to=2026-06-30T23:59:59Z&page=1&limit=10`
- Success `200 OK`
```json
{
  "promotionId": 1,
  "totalUsage": 1,
  "totalDiscountAmount": 10000,
  "items": [
    {
      "promotionId": 1,
      "userId": 1001,
      "orderId": 1,
      "usageCount": 1,
      "discountAmount": 10000,
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ]
}
```
- Error bodies
```json
{ "error": { "code": "PROMOTION_NOT_FOUND", "message": "..." } }
```
```json
{ "error": { "code": "INVALID_QUERY_PARAMETER", "message": "invalid from parameter" } }
```

### 13.5 Pricing

#### `POST /api/v1/pricing/calculate`
- Request body
```json
{
  "userId": 1001,
  "currency": "THB",
  "couponCodes": [],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 },
    { "productId": 2, "quantity": 2 }
  ]
}
```
- Success `200 OK`
```json
{
  "calculationId": "calc-001",
  "originalTotal": 200000,
  "discountTotal": 10000,
  "finalTotal": 190000,
  "currency": "THB",
  "items": [
    {
      "productId": 1,
      "sku": "PRODUCT-001",
      "productName": "Product 1",
      "quantity": 1,
      "unitPrice": 100000,
      "originalAmount": 100000,
      "discountAmount": 10000,
      "finalAmount": 90000
    },
    {
      "productId": 2,
      "sku": "PRODUCT-002",
      "productName": "Product 2",
      "quantity": 2,
      "unitPrice": 50000,
      "originalAmount": 100000,
      "discountAmount": 0,
      "finalAmount": 100000
    }
  ],
  "appliedPromotions": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "discountAmount": 10000
    }
  ],
  "skippedPromotions": []
}
```
- Error bodies
```json
{ "error": { "code": "EMPTY_ORDER_ITEMS", "message": "..." } }
```
```json
{ "error": { "code": "CURRENCY_MISMATCH", "message": "..." } }
```

#### `POST /api/v1/pricing/explain`
- Request body
```json
{
  "userId": 1001,
  "currency": "THB",
  "couponCodes": [],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 }
  ]
}
```
- Success `200 OK`
```json
{
  "calculationId": "calc-001",
  "originalTotal": 100000,
  "discountTotal": 10000,
  "finalTotal": 90000,
  "currency": "THB",
  "items": [
    {
      "productId": 1,
      "sku": "PRODUCT-001",
      "productName": "Product 1",
      "quantity": 1,
      "unitPrice": 100000,
      "originalAmount": 100000,
      "discountAmount": 10000,
      "finalAmount": 90000
    }
  ],
  "appliedPromotions": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "discountAmount": 10000
    }
  ],
  "skippedPromotions": []
}
```
- Error body: same shape as `pricing/calculate`

### 13.6 Order

#### `POST /api/v1/orders/confirm`
- Required header: `Idempotency-Key`
- Request body
```json
{
  "calculationId": "calc-001",
  "acceptedFinalTotal": 135000,
  "userId": 1001,
  "currency": "THB",
  "couponCodes": [],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 }
  ]
}
```
- Success `201 Created`
```json
{
  "orderId": 1,
  "orderNo": "ORD-000001",
  "userId": 1001,
  "status": "CONFIRMED",
  "currency": "THB",
  "originalTotal": 150000,
  "discountTotal": 15000,
  "finalTotal": 135000,
  "calculationId": "calc-001",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z",
  "items": [
    {
      "productId": 1,
      "sku": "PRODUCT-001",
      "productName": "Product 1",
      "quantity": 1,
      "unitPrice": 100000,
      "originalAmount": 100000,
      "discountAmount": 10000,
      "finalAmount": 90000
    }
  ],
  "appliedPromotions": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "discountAmount": 10000
    }
  ],
  "skippedPromotions": [],
  "calculationSnapshot": {
    "request": {},
    "response": {}
  }
}
```
- Error bodies
```json
{ "error": { "code": "IDEMPOTENCY_KEY_REQUIRED", "message": "idempotency key is required" } }
```
```json
{ "error": { "code": "ORDER_PRICE_CHANGED", "message": "..." } }
```

#### `GET /api/v1/orders`
- Request query example: `?status=CONFIRMED&userId=1001&createdFrom=2026-06-01T00:00:00Z&createdTo=2026-06-30T23:59:59Z&page=1&limit=10&sort=created_at desc`
- Success `200 OK`
```json
{
  "items": [
    {
      "orderId": 1,
      "orderNo": "ORD-000001",
      "userId": 1001,
      "status": "CONFIRMED",
      "currency": "THB",
      "originalTotal": 150000,
      "discountTotal": 15000,
      "finalTotal": 135000,
      "calculationId": "calc-001",
      "createdAt": "2026-06-10T10:00:00Z",
      "updatedAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```
- Error body
```json
{ "error": { "code": "INVALID_QUERY_PARAMETER", "message": "invalid date range" } }
```

#### `GET /api/v1/orders/{orderId}`
- Request path example: `/orders/1?userId=1001`
- Success `200 OK`
```json
{
  "orderId": 1,
  "orderNo": "ORD-000001",
  "userId": 1001,
  "status": "CONFIRMED",
  "currency": "THB",
  "originalTotal": 150000,
  "discountTotal": 15000,
  "finalTotal": 135000,
  "calculationId": "calc-001",
  "createdAt": "2026-06-10T10:00:00Z",
  "updatedAt": "2026-06-10T10:00:00Z",
  "items": [
    {
      "productId": 1,
      "sku": "PRODUCT-001",
      "productName": "Product 1",
      "quantity": 1,
      "unitPrice": 100000,
      "originalAmount": 100000,
      "discountAmount": 10000,
      "finalAmount": 90000,
      "createdAt": "2026-06-10T10:00:00Z"
    }
  ],
  "appliedPromotions": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "discountAmount": 10000
    }
  ],
  "skippedPromotions": [],
  "calculationSnapshot": {
    "request": {},
    "response": {}
  }
}
```
- Error bodies
```json
{ "error": { "code": "INVALID_ORDER_ID", "message": "invalid order ID" } }
```
```json
{ "error": { "code": "ORDER_ACCESS_DENIED", "message": "..." } }
```

### 13.7 Calculation logs

#### `GET /api/v1/calculation-logs`
- Request query example: `?requestId=req-001&orderId=1&userId=1001&promotionId=1&createdFrom=2026-06-01T00:00:00Z&createdTo=2026-06-30T23:59:59Z&page=1&limit=10&sort=created_at desc`
- Success `200 OK`
```json
{
  "items": [
    {
      "calculationId": "calc-001",
      "orderId": 1,
      "requestId": "req-001",
      "userId": 1001,
      "originalTotal": 150000,
      "discountTotal": 15000,
      "finalTotal": 135000,
      "appliedPromotionCount": 1,
      "skippedPromotionCount": 0,
      "createdAt": "2026-06-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "totalItems": 1,
    "totalPages": 1
  }
}
```
- Error body
```json
{ "error": { "code": "INVALID_QUERY_PARAMETER", "message": "invalid createdFrom parameter" } }
```

#### `GET /api/v1/calculation-logs/{calculationId}`
- Success `200 OK`
```json
{
  "calculationId": "calc-001",
  "requestId": "req-001",
  "orderId": 1,
  "userId": 1001,
  "originalTotal": 150000,
  "discountTotal": 15000,
  "finalTotal": 135000,
  "appliedPromotionCount": 1,
  "skippedPromotionCount": 0,
  "appliedPromotions": [
    {
      "promotionId": 1,
      "code": "ITEM1_10_PERCENT",
      "name": "Product 1 Discount 10%",
      "scope": "ITEM",
      "discountAmount": 10000
    }
  ],
  "skippedPromotions": [],
  "calculationSnapshot": {
    "request": {},
    "response": {}
  }
}
```
- Error bodies
```json
{ "error": { "code": "CALCULATION_LOG_NOT_FOUND", "message": "..." } }
```
```json
{ "error": { "code": "INVALID_CALCULATION_ID", "message": "invalid calculation ID" } }
```

#### `POST /api/v1/calculation-logs/{calculationId}/replay`
- Request body
```json
{ "mode": "SNAPSHOT_CONFIG" }
```
- Success `200 OK`
```json
{
  "calculationId": "calc-001",
  "mode": "SNAPSHOT_CONFIG",
  "originalResult": {
    "calculationId": "calc-001",
    "originalTotal": 150000,
    "discountTotal": 15000,
    "finalTotal": 135000,
    "currency": "THB",
    "items": [],
    "appliedPromotions": [],
    "skippedPromotions": []
  },
  "replayResult": {
    "calculationId": "calc-001",
    "originalTotal": 150000,
    "discountTotal": 15000,
    "finalTotal": 135000,
    "currency": "THB",
    "items": [],
    "appliedPromotions": [],
    "skippedPromotions": []
  },
  "matched": true,
  "differences": []
}
```
- Error bodies
```json
{ "error": { "code": "REPLAY_MODE_NOT_SUPPORTED", "message": "..." } }
```
```json
{ "error": { "code": "CALCULATION_REPLAY_FAILED", "message": "..." } }
```
