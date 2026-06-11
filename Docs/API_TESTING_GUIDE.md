# API Testing Guide

คู่มือนี้เขียนสำหรับทำ `Postman Test Pipeline` แบบ 15 requests ตาม flow ที่ใช้พิสูจน์โจทย์ของโปรเจกต์นี้โดยตรง

เอกสารนี้ตอบ 3 เรื่องหลัก:

1. promotion ซ้อนกันคำนวณตามลำดับอย่างไร
2. เพิ่ม promotion ใหม่โดยไม่กระทบ logic เดิมอย่างไร
3. คำนวณราคาและ confirm order ได้ถูกต้องอย่างไร

## แนวคิดของ pipeline นี้

pipeline นี้ตั้งใจออกแบบให้เห็นครบทั้ง:

- baseline ของระบบหลัง seed
- การสร้าง promotion ใหม่ผ่าน API
- การ validate และ activate promotion
- การใช้หลาย promotion ใน order เดียว
- การเห็นทั้ง `appliedPromotions` และ `skippedPromotions`
- การยืนยันว่า `pricing/explain`, `pricing/calculate`, `orders/confirm` ต่อกันได้จริง
- การเห็นผลลัพธ์ปลายทางใน `orders` และ `promotion usages`

## Environment ที่แนะนำใน Postman

สร้าง Postman Environment แล้วใส่ค่าต่อไปนี้:

| Variable | Example |
|---|---|
| `baseUrl` | `http://localhost:3000/api/v1` |
| `requestId` | `req-pipeline-001` |
| `idempotencyKey` | `idem-pipeline-001` |
| `userId` | `1001` |

แนะนำให้ใช้ headers มาตรฐานนี้กับทุก request ที่มี body:

```text
Content-Type: application/json
X-Request-ID: {{requestId}}
```

สำหรับ `Request 13: Confirm Order` ต้องเพิ่ม:

```text
Idempotency-Key: {{idempotencyKey}}
```

## ข้อมูลตั้งต้นหลัง reset DB

หลังจาก reset DB และ seed ใหม่ ระบบจะมีข้อมูล baseline สำคัญ:

- `productId = 1`
  - ชื่อ `Product 1`
  - ราคา `100000`
- `productId = 2`
  - ชื่อ `Product 2`
  - ราคา `50000`

promotion seed:

- `ITEM1_10_PERCENT`
- `ITEM2_MINUS_100`

หมายเหตุสำคัญ:

- ถ้าคุณ reset DB ใหม่แล้ว ให้ล้างค่าพวก `promotionId`, `calculationId`, `acceptedFinalTotal`, `orderId` ใน Postman environment ก่อนเริ่มรอบใหม่
- ถ้า `orders/confirm` ขึ้น `INVALID_REQUEST` ส่วนใหญ่เกิดจาก Postman variable ยังไม่ถูกแทนค่าครบ

## โครงสร้างของ 15 Requests

1. List Active Promotions Baseline
2. Create Cart Promotion
3. Create Conflicting Cart Promotion
4. Create Coupon Promotion
5. Validate Cart Promotion
6. Validate Conflict Cart Promotion
7. Validate Coupon Promotion
8. Activate Cart Promotion
9. Activate Conflict Cart Promotion
10. Activate Coupon Promotion
11. Explain Pricing With Stacked Promotions
12. Calculate Pricing Final Check
13. Confirm Order
14. Check Order Detail
15. Check Promotion Usage

---

## Request 1: List Active Promotions Baseline

**Method**

`GET {{baseUrl}}/promotions?status=ACTIVE`

**ใช้ทำอะไร**

- ดู baseline ของระบบหลัง seed
- เช็กว่า promotion เริ่มต้นมีครบ
- ใช้เทียบก่อนสร้าง promo ใหม่ใน pipeline

**ควรเห็นอะไร**

- `ITEM1_10_PERCENT`
- `ITEM2_MINUS_100`

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("response contains items array", function () {
  const body = pm.response.json();
  pm.expect(body.items).to.be.an("array");
});
```

**คำอธิบาย**

request นี้ยังไม่ใช่การพิสูจน์ promotion stacking โดยตรง แต่เป็นจุดเริ่มต้นที่ดีมาก เพราะช่วยยืนยันว่า seed data พร้อมและระบบกำลังใช้ข้อมูล baseline ที่คาดหวังอยู่

---

## Request 2: Create Cart Promotion

**Method**

`POST {{baseUrl}}/promotions`

**ใช้ทำอะไร**

- เพิ่ม promotion ใหม่แบบ `CART`
- ใช้พิสูจน์ว่าเราสามารถเพิ่ม promotion ผ่าน data ได้โดยไม่ต้องแก้ engine

**Body**

```json
{
  "code": "PIPELINE_CART5",
  "name": "Pipeline Cart 5%",
  "description": "Cart discount 5 percent when subtotal reaches threshold",
  "scope": "CART",
  "priority": 20,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "conflictGroup": "PIPELINE_CART_GROUP",
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "targets": [
    { "targetType": "CART" }
  ],
  "conditions": [
    {
      "conditionType": "MIN_ORDER_AMOUNT",
      "operator": "GTE",
      "valueJson": 150000,
      "logicalOperator": "AND"
    }
  ],
  "actions": [
    {
      "actionType": "CART_PERCENTAGE_DISCOUNT",
      "valueBasisPoints": 500,
      "appliesTo": "CART"
    }
  ]
}
```

**ควรเห็นอะไร**

- status `201`
- response มี `promotionId`
- response มี `version = 1`
- status เริ่มต้นเป็น `DRAFT`

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 201", function () {
  pm.response.to.have.status(201);
});

pm.environment.set("cartPromotionId", body.promotionId);
pm.environment.set("cartPromotionVersion", body.version);
```

**คำอธิบาย**

ตัวนี้คือ `cart-level discount` ที่จะถูก apply หลัง item promotion ทำงานแล้ว โดยใช้เงื่อนไข `MIN_ORDER_AMOUNT` เพื่อพิสูจน์ว่า rule ไม่ได้ดูแค่ target แต่ดู condition ด้วย

---

## Request 3: Create Conflicting Cart Promotion

**Method**

`POST {{baseUrl}}/promotions`

**ใช้ทำอะไร**

- สร้าง promo CART อีกตัวที่อยู่ `conflictGroup` เดียวกัน
- ใช้สำหรับพิสูจน์ว่าเวลามี promo ซ้อนกัน ระบบสามารถ skip บางตัวด้วยเหตุผลที่อธิบายได้

**Body**

```json
{
  "code": "PIPELINE_CART10_CONFLICT",
  "name": "Pipeline Cart 10% Conflict",
  "description": "Should be skipped when earlier cart promotion is applied",
  "scope": "CART",
  "priority": 21,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "conflictGroup": "PIPELINE_CART_GROUP",
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "targets": [
    { "targetType": "CART" }
  ],
  "conditions": [
    {
      "conditionType": "MIN_ORDER_AMOUNT",
      "operator": "GTE",
      "valueJson": 150000,
      "logicalOperator": "AND"
    }
  ],
  "actions": [
    {
      "actionType": "CART_PERCENTAGE_DISCOUNT",
      "valueBasisPoints": 1000,
      "appliesTo": "CART"
    }
  ]
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 201", function () {
  pm.response.to.have.status(201);
});

pm.environment.set("cartConflictPromotionId", body.promotionId);
pm.environment.set("cartConflictPromotionVersion", body.version);
```

**คำอธิบาย**

promotion นี้ถูกสร้างมาไม่ใช่เพื่อให้ apply แต่เพื่อให้เห็น `CONFLICT_GROUP_BLOCKED` ตอน explain/calculate ซึ่งเป็นหลักฐานสำคัญของการควบคุม promotion stacking

---

## Request 4: Create Coupon Promotion

**Method**

`POST {{baseUrl}}/promotions`

**ใช้ทำอะไร**

- เพิ่ม promotion แบบ `COUPON`
- ใช้พิสูจน์ว่า engine รองรับหลาย scope ใน order เดียว

**Body**

```json
{
  "code": "PIPELINE_COUPON7",
  "name": "Pipeline Coupon 7%",
  "description": "Coupon SAVE7 gives 7 percent off",
  "scope": "COUPON",
  "priority": 30,
  "stackable": true,
  "exclusive": false,
  "stopProcessing": false,
  "conflictGroup": "PIPELINE_COUPON_GROUP",
  "startsAt": "2026-01-01T00:00:00Z",
  "endsAt": "2026-12-31T23:59:59Z",
  "targets": [
    { "targetType": "CART" }
  ],
  "conditions": [
    {
      "conditionType": "COUPON_CODE",
      "operator": "EQ",
      "valueJson": "SAVE7",
      "logicalOperator": "AND"
    }
  ],
  "actions": [
    {
      "actionType": "CART_PERCENTAGE_DISCOUNT",
      "valueBasisPoints": 700,
      "appliesTo": "CART"
    }
  ]
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 201", function () {
  pm.response.to.have.status(201);
});

pm.environment.set("couponPromotionId", body.promotionId);
pm.environment.set("couponPromotionVersion", body.version);
```

**คำอธิบาย**

request นี้แสดงให้เห็นว่า promotion ไม่จำเป็นต้องผูกกับ product เสมอไป แต่สามารถผูกกับ `coupon code` และทำงานหลัง `ITEM`/`CART` ได้

---

## Request 5: Validate Cart Promotion

**Method**

`POST {{baseUrl}}/promotions/{{cartPromotionId}}/validate`

**Body**

```json
{
  "expectedVersion": {{cartPromotionVersion}}
}
```

**ใช้ทำอะไร**

- ตรวจ config ของ promo ก่อน activate
- ใช้พิสูจน์ว่า promotion lifecycle ไม่ใช่ create แล้ว active ทันที

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("promotion is valid", function () {
  pm.expect(pm.response.json().valid).to.eql(true);
});
```

**คำอธิบาย**

การมี `validate` step แยกจาก `create` ช่วยให้ระบบรองรับ promotion ที่ซับซ้อนขึ้นในอนาคตได้ดีขึ้น เพราะสามารถ reject config ที่ไม่ถูกต้องก่อนใช้งานจริง

---

## Request 6: Validate Conflict Cart Promotion

**Method**

`POST {{baseUrl}}/promotions/{{cartConflictPromotionId}}/validate`

**Body**

```json
{
  "expectedVersion": {{cartConflictPromotionVersion}}
}
```

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("promotion is valid", function () {
  pm.expect(pm.response.json().valid).to.eql(true);
});
```

**คำอธิบาย**

ถึงแม้ promo นี้ถูกสร้างมาเพื่อให้ conflict ตอนคำนวณ แต่มันควร validate ผ่าน เพราะ config เองไม่ได้ผิด

---

## Request 7: Validate Coupon Promotion

**Method**

`POST {{baseUrl}}/promotions/{{couponPromotionId}}/validate`

**Body**

```json
{
  "expectedVersion": {{couponPromotionVersion}}
}
```

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("promotion is valid", function () {
  pm.expect(pm.response.json().valid).to.eql(true);
});
```

**คำอธิบาย**

request นี้พิสูจน์ว่า promotion แบบ coupon-based ถูกมองเป็น rule ปกติในระบบ ไม่ได้มี flow พิเศษแยกออกจาก promotion อื่น

---

## Request 8: Activate Cart Promotion

**Method**

`POST {{baseUrl}}/promotions/{{cartPromotionId}}/activate`

**Body**

```json
{
  "expectedVersion": {{cartPromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.environment.set("cartPromotionVersion", body.version);
```

**คำอธิบาย**

หลัง activate แล้ว promo นี้จะถูก query ผ่าน `FindActivePromotions` และเข้าสู่ calculation engine จริง

---

## Request 9: Activate Conflict Cart Promotion

**Method**

`POST {{baseUrl}}/promotions/{{cartConflictPromotionId}}/activate`

**Body**

```json
{
  "expectedVersion": {{cartConflictPromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.environment.set("cartConflictPromotionVersion", body.version);
```

**คำอธิบาย**

promotion นี้ต้อง active ด้วย เพื่อให้ตอน pricing engine วิ่งจริง เราจะได้เห็นมันถูก skip ด้วยเหตุผล `CONFLICT_GROUP_BLOCKED`

---

## Request 10: Activate Coupon Promotion

**Method**

`POST {{baseUrl}}/promotions/{{couponPromotionId}}/activate`

**Body**

```json
{
  "expectedVersion": {{couponPromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.environment.set("couponPromotionVersion", body.version);
```

**คำอธิบาย**

หลัง request นี้ ระบบพร้อมสำหรับการยิง pricing flow แบบ stacked promotions เต็มรูปแบบ

---

## Request 11: Explain Pricing With Stacked Promotions

**Method**

`POST {{baseUrl}}/pricing/explain`

**ใช้ทำอะไร**

- เป็น request สำคัญที่สุดของ pipeline นี้
- ใช้พิสูจน์ promotion stacking, deterministic order, conflict handling และ skip reason

**Body**

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

**ควรเห็นอะไร**

ในเคสที่ run ตาม pipeline นี้:

- `ITEM1_10_PERCENT` ควร apply
- `PIPELINE_CART5` ควร apply
- `PIPELINE_COUPON7` ควร apply
- `ITEM2_MINUS_100` มีโอกาสถูก skip ด้วย `CONFLICT_GROUP_BLOCKED`
- `PIPELINE_CART10_CONFLICT` ควรถูก skip ด้วย `CONFLICT_GROUP_BLOCKED`

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.environment.set("explainCalculationId", body.calculationId);

pm.test("has applied promotions", function () {
  pm.expect(body.appliedPromotions.length).to.be.greaterThan(0);
});

pm.test("has skipped promotions", function () {
  pm.expect(body.skippedPromotions.length).to.be.greaterThan(0);
});

pm.test("final total is consistent", function () {
  pm.expect(body.finalTotal).to.eql(body.originalTotal - body.discountTotal);
});
```

**คำอธิบายละเอียด**

payload นี้ตั้งใจออกแบบเพื่อให้เห็นหลายชั้นของระบบพร้อมกัน:

- `productId: 1` ทำให้ item promotion ของสินค้า 1 ทำงาน
- `productId: 2` ทำให้เห็น interaction กับ seed promotion ของสินค้า 2
- quantity รวมทำให้ subtotal ผ่าน `MIN_ORDER_AMOUNT`
- `couponCodes: ["SAVE7"]` ทำให้ coupon promotion ทำงาน
- มี cart promotion 2 ตัวใน conflict group เดียวกันเพื่อให้เห็น skip reason

request นี้คือหลักฐานสำคัญที่สุดเวลาจะอธิบายโจทย์เรื่อง “promotion ซ้อนกัน”

---

## Request 12: Calculate Pricing Final Check

**Method**

`POST {{baseUrl}}/pricing/calculate`

**ใช้ทำอะไร**

- ยืนยันว่าตัวเลขจริงของ calculation ตรงกับ logic ที่ explain อธิบายไว้
- ใช้เก็บ `calculationId` และ `acceptedFinalTotal` ไปต่อกับ confirm order

**Body**

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

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.environment.set("calculationId", body.calculationId);
pm.environment.set("acceptedFinalTotal", body.finalTotal);

pm.test("final total is non-negative", function () {
  pm.expect(body.finalTotal).to.be.at.least(0);
});
```

**คำอธิบาย**

`pricing/calculate` คือ version ที่ใกล้กับ production preview มากกว่า `explain` เพราะเอาไว้ดูราคาสุดท้ายที่ user จะเห็นจริงก่อน confirm

---

## Request 13: Confirm Order

**Method**

`POST {{baseUrl}}/orders/confirm`

**Headers เพิ่ม**

```text
Idempotency-Key: {{idempotencyKey}}
```

**Body**

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

**ใช้ทำอะไร**

- พิสูจน์ว่า order flow จริงจะ recalculate ราคาอีกครั้ง
- พิสูจน์ว่า final total ที่ client ยอมรับต้องตรงกับที่ระบบคำนวณ
- พิสูจน์ว่า confirm สำเร็จแล้วจะบันทึก order และ promotion usage

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 201", function () {
  pm.response.to.have.status(201);
});

pm.environment.set("orderId", body.orderId);

pm.test("confirmed total matches accepted total", function () {
  pm.expect(body.finalTotal).to.eql(Number(pm.environment.get("acceptedFinalTotal")));
});
```

**คำอธิบายละเอียด**

request นี้เป็น proof ของ `business correctness` ไม่ใช่แค่ calculation correctness เพราะระบบต้อง:

- เช็ก idempotency
- recalculate ราคา
- เช็ก usage limit ของ promotion
- บันทึก order
- บันทึก order items
- บันทึก promotion usages

ถ้า `acceptedFinalTotal` ไม่ตรง ระบบควรตอบ `ORDER_PRICE_CHANGED`

---

## Request 14: Check Order Detail

**Method**

`GET {{baseUrl}}/orders/{{orderId}}`

**ใช้ทำอะไร**

- ตรวจว่าหลัง confirm แล้ว order ถูกเก็บครบ
- ใช้ดู `items`, `appliedPromotions`, `skippedPromotions`, `calculationSnapshot`

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});
```

**คำอธิบาย**

request นี้ช่วยยืนยันว่าระบบไม่ได้แค่คำนวณได้ แต่ยังเก็บผลคำนวณและ snapshot สำหรับ audit ย้อนหลังได้ด้วย

---

## Request 15: Check Promotion Usage

**Method**

`GET {{baseUrl}}/promotions/{{cartPromotionId}}/usages`

**ใช้ทำอะไร**

- ตรวจว่า promotion usage ถูกบันทึกหลัง confirm order แล้ว
- พิสูจน์ว่า order confirmation ส่งผลต่อ runtime data ของ promotion จริง

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("response contains usage summary", function () {
  const body = pm.response.json();
  pm.expect(body).to.have.property("promotionId");
  pm.expect(body).to.have.property("totalUsage");
});
```

**คำอธิบาย**

ถ้า request นี้เห็น usage เพิ่มขึ้น แปลว่าระบบไม่ได้แค่ preview promotion แต่ได้ consume usage จริงใน transaction ของ confirm order แล้ว

---

## สรุปว่า 15 Requests นี้ตอบโจทย์อะไร

### ตอบเรื่อง promotion ซ้อนกัน

requests ที่สำคัญที่สุด:

- `Request 11: Explain Pricing With Stacked Promotions`
- `Request 12: Calculate Pricing Final Check`

เพราะจะเห็น:

- ลำดับ `ITEM -> CART -> COUPON -> SHIPPING`
- promo ที่ apply
- promo ที่ skip
- เหตุผลของการ skip

### ตอบเรื่องเพิ่ม promotion โดยไม่กระทบ logic เดิม

requests ที่สำคัญที่สุด:

- `Request 2`
- `Request 3`
- `Request 4`
- `Request 5-10`

เพราะทั้งหมดนี้เป็นการเพิ่มและเปิดใช้ promotion ใหม่ผ่าน API โดยไม่ต้องแก้ core flow

### ตอบเรื่องคำนวณถูกต้อง

requests ที่สำคัญที่สุด:

- `Request 11`
- `Request 12`
- `Request 13`
- `Request 14`
- `Request 15`

เพราะครอบคลุมตั้งแต่ preview, explain, confirm, order snapshot และ usage tracking

## ทิปเวลา demo

ถ้าจะ demo ให้ชัดที่สุด:

1. เปิด `Request 11` เป็นจุดหลัก
2. อธิบาย `appliedPromotions` และ `skippedPromotions`
3. ชี้ว่า promo ใหม่ถูกสร้างผ่าน API ไม่ได้ hardcode
4. ปิดท้ายที่ `Request 13-15` เพื่อยืนยันว่า flow จริงถึง order และ usage

## ปัญหาที่เจอบ่อย

### `INVALID_REQUEST`

มักเกิดจาก Postman variable ยังไม่ถูกแทนค่าครบ เช่น:

- `{{calculationId}}`
- `{{acceptedFinalTotal}}`
- `{{userId}}`

### `IDEMPOTENCY_KEY_REQUIRED`

เกิดจากลืมส่ง header:

```text
Idempotency-Key: {{idempotencyKey}}
```

### `ORDER_PRICE_CHANGED`

เกิดจาก `acceptedFinalTotal` ไม่ตรงกับค่าที่ calculate ล่าสุด

### reset DB แล้ว request หลัง ๆ พัง

เพราะ environment variable เก่าค้างอยู่ ให้ล้าง:

- `cartPromotionId`
- `cartConflictPromotionId`
- `couponPromotionId`
- `calculationId`
- `acceptedFinalTotal`
- `orderId`
