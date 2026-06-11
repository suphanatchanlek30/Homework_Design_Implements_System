# API Testing Guide

คู่มือนี้เน้น “เส้นเทสหลัก” เพื่อพิสูจน์โจทย์ 3 เรื่อง:

1. promotion ซ้อนกันจัดลำดับอย่างไร
2. เพิ่ม promotion ใหม่โดยไม่กระทบ logic เดิมอย่างไร
3. คำนวณราคาและ confirm order ได้ถูกต้องอย่างไร

## Environment ที่แนะนำ

ตั้งค่าใน Postman:

| Variable | Example |
|---|---|
| `baseUrl` | `http://localhost:3000/api/v1` |
| `requestId` | `req-pipeline-001` |
| `idempotencyKey` | `idem-pipeline-001` |
| `userId` | `1001` |

Headers มาตรฐาน:

```text
Content-Type: application/json
X-Request-ID: {{requestId}}
```

## Seed baseline ที่ระบบมีอยู่แล้ว

หลัง reset DB ระบบจะมี:

- `productId = 1` ราคา `100000`
- `productId = 2` ราคา `50000`
- promotion seed:
  - `ITEM1_10_PERCENT`
  - `ITEM2_MINUS_100`

## Pipeline ที่ควรยิง

### 1. Health และ baseline

- `GET {{baseUrl}}/healthz`
- `GET {{baseUrl}}/readyz`
- `GET {{baseUrl}}/promotions?status=ACTIVE`

ใช้เช็กว่า app พร้อม, DB พร้อม, และ seed กลับมาปกติ

### 2. สร้าง promotion สำหรับ proof

สร้าง 3 ตัวนี้:

- `PIPELINE_CART5`
  - scope `CART`
  - ลด 5%
  - ใช้ `MIN_ORDER_AMOUNT`
- `PIPELINE_CART10_CONFLICT`
  - scope `CART`
  - conflict group เดียวกับ `PIPELINE_CART5`
  - มีไว้ให้เห็น `CONFLICT_GROUP_BLOCKED`
- `PIPELINE_COUPON7`
  - scope `COUPON`
  - coupon `SAVE7`
  - ลด 7%

จากนั้นยิง:

- `POST /promotions/{promotionId}/validate`
- `POST /promotions/{promotionId}/activate`

### 3. Explain pricing

ยิง `POST {{baseUrl}}/pricing/explain` ด้วย payload นี้:

```json
{
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["SAVE7"],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 },
    { "productId": 2, "quantity": 2 }
  ]
}
```

สิ่งที่ควรเห็น:

- `appliedPromotions`
  - `ITEM1_10_PERCENT`
  - `PIPELINE_CART5`
  - `PIPELINE_COUPON7`
- `skippedPromotions`
  - `ITEM2_MINUS_100`
  - `PIPELINE_CART10_CONFLICT`

request นี้คือ proof หลักของ promotion stacking และ conflict handling

### 4. Calculate pricing

ยิง payload เดียวกันที่ `POST {{baseUrl}}/pricing/calculate`

ใน Postman `Tests` ให้เก็บค่า:

```javascript
const body = pm.response.json();
pm.environment.set("calculationId", body.calculationId);
pm.environment.set("acceptedFinalTotal", body.finalTotal);
```

ใช้ request นี้เป็น proof ของตัวเลขจริงที่ user เห็นก่อน confirm

### 5. Confirm order

ยิง `POST {{baseUrl}}/orders/confirm`

Header เพิ่ม:

```text
Idempotency-Key: {{idempotencyKey}}
```

Body:

```json
{
  "calculationId": "{{calculationId}}",
  "acceptedFinalTotal": {{acceptedFinalTotal}},
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["SAVE7"],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 },
    { "productId": 2, "quantity": 2 }
  ]
}
```

ใช้ proof ว่า confirm ต้อง recalculate และ final total ต้องตรงกับ calculation

### 6. ตรวจ audit และ usage

- `GET {{baseUrl}}/orders/{{orderId}}`
- `GET {{baseUrl}}/calculation-logs/{{calculationId}}`
- `GET {{baseUrl}}/promotions/{{promotionId}}/usages`

ใช้ proof ว่าระบบเก็บ snapshot และ usage หลังใช้งานจริง

## คำอธิบายสั้นสำหรับ demo

- `pricing/explain` ใช้ตอบคำถามเรื่องลำดับคำนวณและเหตุผลที่ promo ถูก apply/skip
- `POST /promotions` + `validate` + `activate` ใช้ตอบคำถามเรื่อง extensibility
- `pricing/calculate` + `orders/confirm` ใช้ตอบคำถามเรื่อง correctness ใน flow จริง

## คำเตือนที่ควรรู้ตอนเทส

- ถ้า reset DB ใหม่ ให้ล้างค่าพวก `calculationId`, `acceptedFinalTotal`, `orderId`, `promotionId` ใน Postman environment ก่อน
- `orders/confirm` ต้องมี `Idempotency-Key`
- ถ้า body ขึ้น `INVALID_REQUEST` ให้เช็กก่อนว่า Postman แทนค่า `{{...}}` ครบและ JSON ยัง valid
