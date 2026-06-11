# API Testing Guide

คู่มือนี้เขียนเพื่อให้คุณใช้ `Postman` หรือ `Insomnia` ไล่ทดสอบระบบแบบครบ flow ตั้งแต่ baseline หลัง seed ไปจนถึง pricing, order confirmation, usage tracking, calculation log, replay และ policy ใหม่ของ `stackable / exclusive`

เอกสารนี้มี 4 เป้าหมายหลัก:

1. พิสูจน์ว่า promotion หลายตัวทำงานร่วมกันตามลำดับ `ITEM -> CART -> COUPON -> SHIPPING`
2. พิสูจน์ว่า promo ใหม่ถูกเพิ่มผ่าน data/API ได้ โดยไม่ต้องแก้ engine
3. พิสูจน์ว่า `stackable`, `exclusive`, `conflictGroup` ส่งผลกับผลลัพธ์จริง และอธิบาย policy ของ `stopProcessing` ให้ตรงกับ engine
4. พิสูจน์ว่า flow จริงตั้งแต่ `pricing/explain` ไปถึง `orders/confirm`, `promotion usages`, `calculation logs` ใช้งานต่อกันได้

## ภาพรวมการทดสอบ

เอกสารนี้แบ่งเป็น 2 ชุด:

- `Core Flow` ใช้พิสูจน์ flow หลักที่ควรเดโมก่อน
- `Extended Flow` ใช้พิสูจน์ policy และ edge case เพิ่มเติม

แนะนำให้รันตามลำดับนี้:

1. Reset DB
2. รัน `Core Flow`
3. รัน `Extended Flow`

## Reset DB ก่อนเริ่ม

ถ้าต้องการเริ่มจากข้อมูลสะอาด:

```bash
docker compose down -v
docker compose up --build
```

ถ้าต้องการดู log ของ app:

```bash
docker compose logs -f app
```

ถ้าต้องการดู log ของ mysql:

```bash
docker compose logs -f mysql
```

## Environment ที่แนะนำใน Postman

สร้าง Postman Environment แล้วใส่ค่าต่อไปนี้

| Variable | Example |
|---|---|
| `baseUrl` | `http://localhost:3000/api/v1` |
| `requestId` | `req-pipeline-001` |
| `idempotencyKey` | `idem-pipeline-001` |
| `userId` | `1001` |
| `couponCode` | `SAVE7` |

หมายเหตุ:

- ถ้า request body ใช้ `{{couponCode}}` ค่าใน Postman Environment จะถูกแทนก่อนส่งจริง
- ถ้าไม่ต้องการทดสอบ coupon promotion ให้ส่ง `couponCodes: []` หรือไม่ส่ง field นี้

ตัวแปรที่ระบบจะสร้างระหว่างทาง:

| Variable | มาจาก request ไหน |
|---|---|
| `cartPromotionId` | Create Cart Promotion |
| `cartPromotionVersion` | Create/Activate Cart Promotion |
| `cartConflictPromotionId` | Create Conflict Cart Promotion |
| `cartConflictPromotionVersion` | Create/Activate Conflict Cart Promotion |
| `couponPromotionId` | Create Coupon Promotion |
| `couponPromotionVersion` | Create/Activate Coupon Promotion |
| `nonStackablePromotionId` | Create Non-Stackable Promotion |
| `nonStackablePromotionVersion` | Create/Activate Non-Stackable Promotion |
| `exclusivePromotionId` | Create Exclusive Promotion |
| `exclusivePromotionVersion` | Create/Activate Exclusive Promotion |
| `explainCalculationId` | Pricing Explain |
| `calculationId` | Pricing Calculate |
| `acceptedFinalTotal` | Pricing Calculate |
| `orderId` | Confirm Order |

## Headers มาตรฐาน

ใช้กับทุก request ที่มี body:

```text
Content-Type: application/json
X-Request-ID: {{requestId}}
```

สำหรับ `orders/confirm` เพิ่ม:

```text
Idempotency-Key: {{idempotencyKey}}
```

## วิธีจัดการค่าใน Postman ก่อนเริ่มรอบใหม่

ถ้า reset DB ใหม่ แนะนำให้ล้างค่าพวกนี้ใน environment:

- `cartPromotionId`
- `cartPromotionVersion`
- `cartConflictPromotionId`
- `cartConflictPromotionVersion`
- `couponPromotionId`
- `couponPromotionVersion`
- `nonStackablePromotionId`
- `nonStackablePromotionVersion`
- `exclusivePromotionId`
- `exclusivePromotionVersion`
- `explainCalculationId`
- `calculationId`
- `acceptedFinalTotal`
- `orderId`

## ข้อมูล baseline หลัง seed

หลัง reset DB และ seed ใหม่ ระบบควรมี baseline สำคัญ:

- `productId = 1`
  - ชื่อ `Product 1`
  - ราคา `100000`
- `productId = 2`
  - ชื่อ `Product 2`
  - ราคา `50000`

promotion seed สำคัญ:

- `ITEM1_10_PERCENT`
- `ITEM2_MINUS_100`

## นิยาม policy ที่ใช้ในระบบตอนนี้

ก่อนทดสอบ extended flow ควรยึดนิยามนี้:

- `stackable=true` = promo นี้ยอมให้ apply ร่วมกับ promo อื่นได้
- `stackable=false` = promo นี้ apply ไม่ได้ถ้ามี applied promo มาก่อน และถ้า apply สำเร็จแล้ว promo ถัดไปจะถูก block
- `exclusive=true` = promo นี้ apply ไม่ได้ถ้ามี applied promo มาก่อน และถ้า apply สำเร็จให้จบการคำนวณทันที
- `stopProcessing=true` = หลัง apply promo นี้แล้วหยุดประมวลผล promo ถัดไปทันที โดย promo หลังจากนั้นจะไม่ถูกบันทึกเป็น skipped
- `conflictGroup` = กันชนเฉพาะกลุ่ม promo

ใน Request 21-30 ตัวอย่าง promotion ตั้ง `stopProcessing=false` ทั้งหมด ดังนั้นชุดนี้ไม่ได้สร้าง stop-processing promo แยกต่างหาก แต่ behavior `stopProcessing=true` ถูกยืนยันใน unit test ของ engine

skip reasons ใหม่ที่ควรรู้:

- `CONFLICT_GROUP_BLOCKED`
- `NON_STACKABLE_ALREADY_APPLIED`
- `NON_STACKABLE_CANNOT_STACK`
- `EXCLUSIVE_ALREADY_APPLIED`
- `EXCLUSIVE_CANNOT_STACK`

## โครงสร้างการทดสอบทั้งหมด

### Core Flow

1. Health Check
2. Ready Check
3. List Active Promotions Baseline
4. Create Cart Promotion
5. Create Conflicting Cart Promotion
6. Create Coupon Promotion
7. Validate Cart Promotion
8. Validate Conflict Cart Promotion
9. Validate Coupon Promotion
10. Activate Cart Promotion
11. Activate Conflict Cart Promotion
12. Activate Coupon Promotion
13. Explain Pricing With Stacked Promotions
14. Calculate Pricing Final Check
15. Confirm Order
16. Check Order Detail
17. Check Promotion Usage
18. Check Calculation Log List
19. Check Calculation Log Detail
20. Replay Calculation Log

### Extended Flow

21. Create Non-Stackable Promotion
22. Validate and Activate Non-Stackable Promotion
23. Explain Pricing For Non-Stackable Scenario
24. Deactivate Non-Stackable Promotion
25. Create Exclusive Promotion
26. Validate and Activate Exclusive Promotion
27. Explain Pricing For Exclusive Scenario
28. Deactivate Exclusive Promotion
29. Confirm Order With Same Idempotency Key
30. Confirm Order With Wrong Accepted Total

---

## ส่วนที่ 1: Core Flow

## Request 1: Health Check

**Method**

`GET {{baseUrl}}/healthz`

**ใช้ทำอะไร**

- เช็กว่า app process ยังตอบ API ได้

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});
```

**ควรเห็นอะไร**

- response สำเร็จ

---

## Request 2: Ready Check

**Method**

`GET {{baseUrl}}/readyz`

**ใช้ทำอะไร**

- เช็กว่า app พร้อมทำงานกับ DB จริง

**Postman Tests**

```javascript
pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});
```

**ควรเห็นอะไร**

- response พร้อมใช้งาน
- ถ้ามี field mysql ควรสะท้อนว่า DB พร้อม

---

## Request 3: List Active Promotions Baseline

**Method**

`GET {{baseUrl}}/promotions?status=ACTIVE`

**ใช้ทำอะไร**

- ดู baseline หลัง seed
- ยืนยันว่า promotion เริ่มต้นยังอยู่

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

**ควรเห็นอะไร**

- มี `ITEM1_10_PERCENT`
- มี `ITEM2_MINUS_100`

---

## Request 4: Create Cart Promotion

**Method**

`POST {{baseUrl}}/promotions`

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

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 201", function () {
  pm.response.to.have.status(201);
});

pm.environment.set("cartPromotionId", body.promotionId);
pm.environment.set("cartPromotionVersion", body.version);
```

**ควรเห็นอะไร**

- status `201`
- ได้ `promotionId`
- `status = DRAFT`
- `version = 1`

---

## Request 5: Create Conflicting Cart Promotion

**Method**

`POST {{baseUrl}}/promotions`

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

**ควรเห็นอะไร**

- promo สร้างสำเร็จ
- ตัวนี้ยังไม่ควร apply ทันที เพราะยังอยู่ `DRAFT`

**อธิบาย**

ตัวนี้สร้างมาเพื่อพิสูจน์ `CONFLICT_GROUP_BLOCKED`

---

## Request 6: Create Coupon Promotion

**Method**

`POST {{baseUrl}}/promotions`

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

**ควรเห็นอะไร**

- promo แบบ `COUPON` ถูกสร้างผ่าน API ได้

---

## Request 7: Validate Cart Promotion

**Method**

`POST {{baseUrl}}/promotions/{{cartPromotionId}}/validate`

**Body**

```json
{
  "expectedVersion": {{cartPromotionVersion}}
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

---

## Request 8: Validate Conflict Cart Promotion

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

---

## Request 9: Validate Coupon Promotion

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

---

## Request 10: Activate Cart Promotion

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

---

## Request 11: Activate Conflict Cart Promotion

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

---

## Request 12: Activate Coupon Promotion

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

---

## Request 13: Explain Pricing With Stacked Promotions

**Method**

`POST {{baseUrl}}/pricing/explain`

**Body**

```json
{
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["{{couponCode}}"],
  "paymentMethod": "PROMPTPAY",
  "shipping": { "method": "STANDARD" },
  "items": [
    { "productId": 1, "quantity": 1 },
    { "productId": 2, "quantity": 2 }
  ]
}
```

body ชุดนี้คือกรณี "มีคูปอง" เพราะ `{{couponCode}}` จะถูกแทนด้วยค่าจาก environment เช่น `SAVE7`

ถ้าต้องการทดสอบกรณี "ไม่มีคูปอง" ให้ใช้ body นี้แทน:

```json
{
  "userId": {{userId}},
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

**Postman Tests**

```javascript
const body = pm.response.json();
const appliedCodes = body.appliedPromotions.map(p => p.code);
const skippedReasons = body.skippedPromotions.map(p => p.reason);

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.environment.set("explainCalculationId", body.calculationId);

pm.test("has applied promotions", function () {
  pm.expect(body.appliedPromotions.length).to.be.greaterThan(0);
});

pm.test("final total is consistent", function () {
  pm.expect(body.finalTotal).to.eql(body.originalTotal - body.discountTotal);
});

pm.test("cart promotion is applied", function () {
  pm.expect(appliedCodes).to.include("PIPELINE_CART5");
});

pm.test("coupon promotion is applied", function () {
  pm.expect(appliedCodes).to.include("PIPELINE_COUPON7");
});

pm.test("conflict promotion is skipped", function () {
  pm.expect(skippedReasons).to.include("CONFLICT_GROUP_BLOCKED");
});
```

**ควรเห็นอะไร**

- seed item promotions มีโอกาส apply ตาม target
- `PIPELINE_CART5` apply
- `PIPELINE_CART10_CONFLICT` skip ด้วย `CONFLICT_GROUP_BLOCKED`
- `PIPELINE_COUPON7` apply
- ถ้าใช้ `couponCodes: []` โปร `PIPELINE_COUPON7` ไม่ควร apply และมักจะไปอยู่ใน `skippedPromotions` ด้วย `COUPON_CODE_MISMATCH`

**อธิบายละเอียด**

request นี้เป็นจุดสำคัญที่สุดของทั้งระบบ เพราะคุณจะเห็นพร้อมกันว่า:

- item promotion ทำงานก่อน
- cart promotion ทำงานหลัง item
- coupon promotion ทำงานหลัง cart
- conflict group ทำให้บาง promo ถูก skip
- `appliedPromotions` และ `skippedPromotions` อธิบาย flow ได้จริง

---

## Request 14: Calculate Pricing Final Check

**Method**

`POST {{baseUrl}}/pricing/calculate`

**Body**

```json
{
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["{{couponCode}}"],
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

**ใช้ทำอะไร**

- เก็บ `calculationId` ไปต่อกับ confirm order
- เก็บ `acceptedFinalTotal` ที่ client จะยอมรับ

---

## Request 15: Confirm Order

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
  "couponCodes": ["{{couponCode}}"],
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

pm.test("status is 201", function () {
  pm.response.to.have.status(201);
});

pm.environment.set("orderId", body.orderId);

pm.test("confirmed total matches accepted total", function () {
  pm.expect(body.finalTotal).to.eql(Number(pm.environment.get("acceptedFinalTotal")));
});
```

**ควรเห็นอะไร**

- order ถูกสร้าง
- final total ตรงกับ calculate ล่าสุด
- order มี `appliedPromotions` และ `skippedPromotions`

---

## Request 16: Check Order Detail

**Method**

`GET {{baseUrl}}/orders/{{orderId}}`

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("order has items", function () {
  pm.expect(body.items).to.be.an("array");
  pm.expect(body.items.length).to.be.greaterThan(0);
});

pm.test("order includes promotions arrays", function () {
  pm.expect(body.appliedPromotions).to.be.an("array");
  pm.expect(body.skippedPromotions).to.be.an("array");
});
```

**ใช้ทำอะไร**

- ยืนยันว่า snapshot ถูกเก็บกับ order
- ดูผลคำนวณจริงย้อนหลัง

---

## Request 17: Check Promotion Usage

**Method**

`GET {{baseUrl}}/promotions/{{cartPromotionId}}/usages`

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("response contains usage summary", function () {
  pm.expect(body).to.have.property("promotionId");
  pm.expect(body).to.have.property("totalUsage");
  pm.expect(body).to.have.property("items");
});
```

**ใช้ทำอะไร**

- ยืนยันว่า confirm order consume usage จริง

---

## Request 18: Check Calculation Log List

**Method**

`GET {{baseUrl}}/calculation-logs?userId={{userId}}`

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("contains log items", function () {
  pm.expect(body.items).to.be.an("array");
  pm.expect(body.items.length).to.be.greaterThan(0);
});
```

**ใช้ทำอะไร**

- ดูว่าระบบเก็บ calculation log ทุกครั้งที่ calculate/explain สำเร็จ

---

## Request 19: Check Calculation Log Detail

**Method**

`GET {{baseUrl}}/calculation-logs/{{calculationId}}`

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("contains applied and skipped promotions", function () {
  pm.expect(body.appliedPromotions).to.be.an("array");
  pm.expect(body.skippedPromotions).to.be.an("array");
});
```

**ใช้ทำอะไร**

- ดู snapshot ของ calculation ตาม `calculationId` ล่าสุด

---

## Request 20: Replay Calculation Log

**Method**

`POST {{baseUrl}}/calculation-logs/{{calculationId}}/replay`

**Body**

```json
{
  "mode": "SNAPSHOT_CONFIG"
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("replay returns comparison result", function () {
  pm.expect(body).to.have.property("matched");
  pm.expect(body).to.have.property("differences");
});
```

**ใช้ทำอะไร**

- พิสูจน์ว่า audit log replay ได้
- ถ้า config และ data ยังเหมือนเดิม `matched` มีโอกาสเป็น `true`

---

## ส่วนที่ 2: Extended Flow

## Request 21: Create Non-Stackable Promotion

**Method**

`POST {{baseUrl}}/promotions`

**Body**

```json
{
  "code": "PIPELINE_NONSTACK_CART3",
  "name": "Pipeline Non-Stackable Cart 3%",
  "description": "A non-stackable cart promo for policy testing",
  "scope": "CART",
  "priority": 25,
  "stackable": false,
  "exclusive": false,
  "stopProcessing": false,
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
      "valueBasisPoints": 300,
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

pm.test("non-stackable policy is persisted", function () {
  pm.expect(body.stackable).to.eql(false);
  pm.expect(body.exclusive).to.eql(false);
  pm.expect(body.stopProcessing).to.eql(false);
});

pm.environment.set("nonStackablePromotionId", body.promotionId);
pm.environment.set("nonStackablePromotionVersion", body.version);
```

**จุดประสงค์**

- สร้าง promo สำหรับพิสูจน์ `NON_STACKABLE_CANNOT_STACK` หรือ `NON_STACKABLE_ALREADY_APPLIED`
- หลังแก้ persistence ของ `stackable=false` แล้ว response ของ request นี้ควรสะท้อน `stackable: false` ตรงตาม request

---

## Request 22: Validate and Activate Non-Stackable Promotion

ให้ยิง 2 requests ต่อกัน

### 22.1 Validate

`POST {{baseUrl}}/promotions/{{nonStackablePromotionId}}/validate`

```json
{
  "expectedVersion": {{nonStackablePromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("validation succeeds", function () {
  pm.response.to.have.status(200);
  pm.expect(body.valid).to.eql(true);
});
```

### 22.2 Activate

`POST {{baseUrl}}/promotions/{{nonStackablePromotionId}}/activate`

```json
{
  "expectedVersion": {{nonStackablePromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("activation succeeds", function () {
  pm.response.to.have.status(200);
  pm.expect(body.status).to.eql("ACTIVE");
  pm.expect(body.stackable).to.eql(false);
});

pm.environment.set("nonStackablePromotionVersion", body.version);
```

**สิ่งที่ควรเห็น**

- ทั้ง validate และ activate ผ่าน

---

## Request 23: Explain Pricing For Non-Stackable Scenario

**Method**

`POST {{baseUrl}}/pricing/explain`

**Body**

```json
{
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["{{couponCode}}"],
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
const skippedReasons = body.skippedPromotions.map(p => p.reason);

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("non-stackable reason appears", function () {
  pm.expect(skippedReasons).to.include("NON_STACKABLE_CANNOT_STACK");
});
```

**อธิบาย**

สำหรับ body และ priority ในคู่มือนี้ (`ITEM=10`, `CART5=20`, `NONSTACK=25`, `COUPON=30`) ควรเห็น `NON_STACKABLE_CANNOT_STACK` เพราะมี promo อื่น apply ไปก่อนแล้ว

ถ้าคุณเปลี่ยน priority ของ non-stackable ให้มาก่อน promo อื่นและมัน apply สำเร็จ ค่อยจะเห็น promo หลังถูก skip ด้วย `NON_STACKABLE_ALREADY_APPLIED`

---

## Request 24: Deactivate Non-Stackable Promotion

**Method**

`POST {{baseUrl}}/promotions/{{nonStackablePromotionId}}/deactivate`

**Body**

```json
{
  "expectedVersion": {{nonStackablePromotionVersion}},
  "reason": "cleanup after non-stackable scenario"
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("deactivation succeeds", function () {
  pm.response.to.have.status(200);
  pm.expect(body.status).to.eql("INACTIVE");
});

pm.environment.set("nonStackablePromotionVersion", body.version);
```

**ใช้ทำอะไร**

- ปิด promo เพื่อไม่ให้กระทบ exclusive scenario ถัดไป

---

## Request 25: Create Exclusive Promotion

**Method**

`POST {{baseUrl}}/promotions`

**Body**

```json
{
  "code": "PIPELINE_EXCLUSIVE_CART8",
  "name": "Pipeline Exclusive Cart 8%",
  "description": "An exclusive cart promo for policy testing",
  "scope": "CART",
  "priority": 22,
  "stackable": true,
  "exclusive": true,
  "stopProcessing": false,
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
      "valueBasisPoints": 800,
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

pm.test("exclusive policy is persisted", function () {
  pm.expect(body.stackable).to.eql(true);
  pm.expect(body.exclusive).to.eql(true);
  pm.expect(body.stopProcessing).to.eql(false);
});

pm.environment.set("exclusivePromotionId", body.promotionId);
pm.environment.set("exclusivePromotionVersion", body.version);
```

---

## Request 26: Validate and Activate Exclusive Promotion

ให้ยิง 2 requests ต่อกัน

### 26.1 Validate

`POST {{baseUrl}}/promotions/{{exclusivePromotionId}}/validate`

```json
{
  "expectedVersion": {{exclusivePromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("validation succeeds", function () {
  pm.response.to.have.status(200);
  pm.expect(body.valid).to.eql(true);
});
```

### 26.2 Activate

`POST {{baseUrl}}/promotions/{{exclusivePromotionId}}/activate`

```json
{
  "expectedVersion": {{exclusivePromotionVersion}}
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("activation succeeds", function () {
  pm.response.to.have.status(200);
  pm.expect(body.status).to.eql("ACTIVE");
  pm.expect(body.exclusive).to.eql(true);
});

pm.environment.set("exclusivePromotionVersion", body.version);
```

---

## Request 27: Explain Pricing For Exclusive Scenario

**Method**

`POST {{baseUrl}}/pricing/explain`

**Body**

```json
{
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["{{couponCode}}"],
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
const appliedCodes = body.appliedPromotions.map(p => p.code);
const skippedReasons = body.skippedPromotions.map(p => p.reason);

pm.test("status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("exclusive promo is applied or blocks later promos", function () {
  pm.expect(skippedReasons).to.include("EXCLUSIVE_CANNOT_STACK");
});
```

**อธิบาย**

สำหรับ body และ priority ในคู่มือนี้ (`ITEM=10`, `CART5=20`, `EXCLUSIVE=22`, `COUPON=30`) ควรเห็น `EXCLUSIVE_CANNOT_STACK` เพราะมี promo อื่น apply ไปก่อนแล้ว

ถ้าคุณเปลี่ยน priority ของ exclusive ให้มาก่อน promo อื่นและมัน apply สำเร็จ engine จะหยุด loop ทันที และ promo หลังจากนั้นจะไม่ถูก apply ต่อ

---

## Request 28: Deactivate Exclusive Promotion

**Method**

`POST {{baseUrl}}/promotions/{{exclusivePromotionId}}/deactivate`

**Body**

```json
{
  "expectedVersion": {{exclusivePromotionVersion}},
  "reason": "cleanup after exclusive scenario"
}
```

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("deactivation succeeds", function () {
  pm.response.to.have.status(200);
  pm.expect(body.status).to.eql("INACTIVE");
});

pm.environment.set("exclusivePromotionVersion", body.version);
```

---

## Request 29: Confirm Order With Same Idempotency Key

**Method**

ยิง `POST {{baseUrl}}/orders/confirm` ซ้ำอีกครั้งด้วย body เดิมและ `Idempotency-Key` เดิม

**ควรเห็นอะไร**

- request สำเร็จ
- ได้ order เดิมกลับมา
- ไม่ควรสร้าง order ใหม่อีกตัว

**Postman Tests**

```javascript
const body = pm.response.json();

pm.test("request still succeeds", function () {
  pm.expect([200, 201]).to.include(pm.response.code);
});

pm.test("same order id is returned", function () {
  pm.expect(body.orderId).to.eql(Number(pm.environment.get("orderId")));
});
```

---

## Request 30: Confirm Order With Wrong Accepted Total

**Method**

`POST {{baseUrl}}/orders/confirm`

**Headers เพิ่ม**

```text
Idempotency-Key: idem-wrong-total-001
```

**ตั้งค่าก่อนยิง request นี้**

เพิ่มใน Postman Tests ของ `Request 14` หรือสร้างค่าเองใน environment:

```javascript
pm.environment.set(
  "wrongAcceptedFinalTotal",
  Number(pm.environment.get("acceptedFinalTotal")) - 1
);
```

**Body**

ใช้ body เดิม แต่เปลี่ยน `acceptedFinalTotal` เป็น `{{wrongAcceptedFinalTotal}}`

```json
{
  "calculationId": "{{calculationId}}",
  "acceptedFinalTotal": {{wrongAcceptedFinalTotal}},
  "userId": {{userId}},
  "currency": "THB",
  "couponCodes": ["{{couponCode}}"],
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
pm.test("status is 409", function () {
  pm.response.to.have.status(409);
});

pm.test("response is ORDER_PRICE_CHANGED", function () {
  const body = pm.response.json();
  pm.expect(body.error.code).to.eql("ORDER_PRICE_CHANGED");
});
```

**ควรเห็นอะไร**

- error `ORDER_PRICE_CHANGED`

---

## ตารางคาดหวังของแต่ละ scenario

### Main stacking scenario

- item promotions ควรทำงานก่อน
- `PIPELINE_CART5` ควร apply
- `PIPELINE_CART10_CONFLICT` ควรถูก skip ด้วย `CONFLICT_GROUP_BLOCKED`
- `PIPELINE_COUPON7` ควร apply
- `finalTotal = originalTotal - discountTotal`

### Non-stackable scenario

- สำหรับ priority ในคู่มือนี้: `NON_STACKABLE_CANNOT_STACK`
- ถ้า non-stackable apply ก่อน: promo หลังจะได้ `NON_STACKABLE_ALREADY_APPLIED`

### Exclusive scenario

- สำหรับ priority ในคู่มือนี้: `EXCLUSIVE_CANNOT_STACK`
- ถ้า exclusive apply สำเร็จ: engine จะหยุด loop ทันที

## วิธีอ่านผล `appliedPromotions` และ `skippedPromotions`

ดู 3 อย่างพร้อมกัน:

1. `code` ว่าตัวไหน apply หรือ skip
2. `scope` ว่าโปรอยู่ชั้นไหนของ flow
3. `reason` ว่าถูก block เพราะอะไร

ตัวอย่างที่ควรอธิบายได้:

- `CONFLICT_GROUP_BLOCKED` = มี promo จากกลุ่มเดียวกัน apply ไปแล้ว
- `NON_STACKABLE_CANNOT_STACK` = promo ตัวนี้ไม่ยอมมาหลัง promo อื่น
- `NON_STACKABLE_ALREADY_APPLIED` = มี promo non-stackable apply ไปแล้ว จึงกัน promo หลัง
- `EXCLUSIVE_CANNOT_STACK` = exclusive promo ไม่ยอมมาในรอบที่มี applied promo แล้ว
- `EXCLUSIVE_ALREADY_APPLIED` = มี exclusive promo apply ไปแล้ว จึงกัน promo ถัดไป
- ถ้าเห็น `stop_processing=true` ใน trace แปลว่า engine หยุด loop หลัง promo นั้น และ promo หลังจากนั้นจะไม่ถูกบันทึกเป็น skipped

## ถ้าจะเดโมสั้น

ถ้าคุณมีเวลาน้อย แนะนำให้ยิงอย่างน้อย:

1. Request 3
2. Request 4-12
3. Request 13
4. Request 14
5. Request 15
6. Request 18-20
7. Request 23
8. Request 27

ชุดนี้ครอบคลุม:

- baseline
- create/validate/activate
- explain/calculate
- confirm order
- usage tracking
- audit replay
- non-stackable
- exclusive

## ปัญหาที่เจอบ่อย

### `INVALID_REQUEST`

มักเกิดจาก:

- variable ใน Postman ยังไม่ถูกแทนค่าครบ
- `{{calculationId}}` หรือ `{{acceptedFinalTotal}}` ยังว่าง
- body ใส่ JSON expression ตรง ๆ แล้ว Postman ไม่แทนให้

### `IDEMPOTENCY_KEY_REQUIRED`

เกิดจากลืมส่ง:

```text
Idempotency-Key: {{idempotencyKey}}
```

### `ORDER_PRICE_CHANGED`

เกิดจาก:

- `acceptedFinalTotal` ไม่ตรงกับค่าที่ calculate ล่าสุด
- เปลี่ยนโปรโมชั่นหรือสินค้าแล้วใช้ calculation เก่า

### explain แล้วไม่เห็น skip reason ที่คาดไว้

ให้ตรวจ:

- promo ถูก activate แล้วหรือยัง
- `startsAt` / `endsAt` ครอบเวลาปัจจุบันหรือยัง
- `priority` ทำให้ลำดับไม่เป็นอย่างที่คิดหรือไม่
- `conflictGroup`, `stackable`, `exclusive` ถูกตั้งถูกหรือไม่

### confirm order แล้ว usage ไม่ขึ้น

ให้ตรวจ:

- request 15 สำเร็จจริงหรือไม่
- request 17 ใช้ promotionId ถูกตัวหรือไม่
- ใช้ order เดิมซ้ำด้วย idempotency key เดิมหรือไม่

## บทสรุป

ถ้าคุณไล่ครบทั้ง `Core Flow` และ `Extended Flow` เอกสารนี้จะช่วยพิสูจน์ได้ครบว่า:

- promotion engine รองรับหลาย scope
- rule ถูกขับด้วย data ไม่ใช่ hardcode
- conflict, non-stackable, exclusive มีผลกับ runtime จริง
- order confirmation ใช้งานต่อจาก pricing ได้
- calculation logs และ replay ใช้งานได้จริง
