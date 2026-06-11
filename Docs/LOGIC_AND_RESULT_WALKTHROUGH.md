# Logic And Result Walkthrough

เอกสารนี้อธิบายโค้ดปัจจุบันของโปรเจกต์โดยยึดตาม flow ในภาพ architecture เป็นหลัก และตอบคำถามว่า:

1. แต่ละกล่องในภาพตรงกับโค้ดส่วนไหน
2. โค้ดแต่ละส่วนทำอะไร
3. ข้อมูลไหลจากจุดไหนไปจุดไหน
4. ผลลัพธ์สุดท้ายที่ระบบสร้างคืออะไร
5. การออกแบบและการเขียนโค้ดแต่ละชั้น support กันอย่างไร

เอกสารนี้อ้างอิง “โค้ดปัจจุบันของโปรเจกต์เท่านั้น” ไม่อิงโค้ดสมมติหรือแผนในอนาคต

## จุดสรุปก่อนอ่าน

ระบบปัจจุบันมี flow หลักแบบนี้:

1. `Fiber` รับ HTTP request
2. middleware ใส่ `X-Request-ID`
3. `PricingHandler` parse body แล้วส่งต่อให้ `PricingService`
4. `PricingService` โหลดสินค้าและ promotions ที่ active จาก repository
5. `Promotion Engine` จัดลำดับ promotions, check target, check condition, ใช้ strategy คำนวณ, และสรุปผล
6. `PricingService` แปลงผลเป็น response DTO และบันทึก calculation log
7. handler ส่ง JSON response กลับไป

สรุปเชิง architecture:

- route layer จัดการ HTTP และ response code
- service layer ทำ orchestration
- repository layer ทำ data access ผ่าน GORM
- promotion engine ทำ business logic คำนวณราคา
- audit/calculation log เก็บ snapshot ของผลลัพธ์เพื่อ replay และตรวจย้อนหลัง

## ภาพรวม wiring ของระบบ

จุดเริ่มต้นของระบบทั้งหมดอยู่ที่ [cmd/server/main.go](../cmd/server/main.go#L18)

สิ่งที่ไฟล์นี้ทำ:

- โหลด config
- connect MySQL ผ่าน GORM
- สร้าง Fiber app
- mount middleware
- สร้าง repositories
- สร้าง services
- สร้าง handlers
- bind routes

โครงสร้าง wiring หลัก:

- database connection ที่ [cmd/server/main.go](../cmd/server/main.go#L21)
- middleware registration ที่ [cmd/server/main.go](../cmd/server/main.go#L27)
- repositories ที่ [cmd/server/main.go](../cmd/server/main.go#L34)
- services ที่ [cmd/server/main.go](../cmd/server/main.go#L41)
- handlers ที่ [cmd/server/main.go](../cmd/server/main.go#L49)
- pricing routes ที่ [cmd/server/main.go](../cmd/server/main.go#L88)
- audit routes ที่ [cmd/server/main.go](../cmd/server/main.go#L99)

นี่คือหลักฐานสำคัญว่าระบบไม่ได้ผูกทุกอย่างไว้ในไฟล์เดียว แต่แยกเป็น layer ที่ support กันชัดเจน

## ตารางจับคู่ภาพกับโค้ดจริง

| กล่องในภาพ | โค้ดจริง | ทำหน้าที่อะไร | ส่งต่อไปไหน |
|---|---|---|---|
| Customer / Client | ผู้ใช้ภายนอกเรียก API | ส่ง request สำหรับคำนวณราคา | `Fiber API` |
| Go Fiber API Post `/api/v1/pricing/calculate` | [cmd/server/main.go](../cmd/server/main.go#L89), [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L17) | รับ request และ parse body | `PricingService` |
| Middleware Validate/Auth/Request ID/Error | [internal/middleware/request_id.go](../internal/middleware/request_id.go#L8), [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L45), [internal/handler/api_error.go](../internal/handler/api_error.go#L1) | ใส่ request ID และ map error กลับเป็น HTTP response | handler/service |
| Pricing Service | [internal/service/pricing_service.go](../internal/service/pricing_service.go#L32) | orchestration การคำนวณราคา | product repo, promotion repo, promotion engine, calculation log |
| Promotion Repository | [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L66) | โหลด promotions ที่ active พร้อม targets/conditions/actions | `PricingService` |
| Product Repository | [internal/repository/product_repository.go](../internal/repository/product_repository.go#L39) | โหลดข้อมูลสินค้าและราคา | `PricingService` |
| MySQL + GORM | [cmd/server/main.go](../cmd/server/main.go#L21), [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L69), [internal/repository/product_repository.go](../internal/repository/product_repository.go#L45) | เก็บ product/promotion/log data | repositories/services |
| Promotion Engine | [internal/promotion/engine.go](../internal/promotion/engine.go#L94) | ประมวลผล business rules จริง | ส่ง `CalculationResult` กลับ `PricingService` |
| Check Target | [internal/promotion/engine.go](../internal/promotion/engine.go#L169), [internal/promotion/engine.go](../internal/promotion/engine.go#L460) | ตรวจว่าตะกร้าตรง target ของโปรหรือไม่ | ถ้าผ่านไป check condition |
| Check Condition | [internal/promotion/engine.go](../internal/promotion/engine.go#L173), [internal/promotion/engine.go](../internal/promotion/engine.go#L259) | ตรวจ coupon, amount, date, payment, etc. | ถ้าผ่านไป apply action |
| Sort Promotion | [internal/promotion/engine.go](../internal/promotion/engine.go#L101) | เรียงตาม scope, priority, createdAt, id | loop ของ engine |
| Stacking Policy | [internal/model/models.go](../internal/model/models.go#L74), [internal/promotion/engine.go](../internal/promotion/engine.go#L149) | enforce `stackable`, `exclusive`, `stopProcessing`, `conflictGroup` | loop ของ engine |
| Strategy Pattern | [internal/promotion/registry.go](../internal/promotion/registry.go#L24), [internal/promotion/engine.go](../internal/promotion/engine.go#L230) | map action/condition type ไปยัง handler | คำนวณ discount และ skip reason |
| Percentage Strategy | [internal/promotion/engine.go](../internal/promotion/engine.go#L284) | คำนวณส่วนลดแบบเปอร์เซ็นต์ | apply ลง item/cart |
| Fixed Amount Strategy | [internal/promotion/engine.go](../internal/promotion/engine.go#L302) | คำนวณส่วนลดแบบจำนวนเงินคงที่ | apply ลง item/cart |
| Future Strategy | [internal/promotion/registry.go](../internal/promotion/registry.go#L57), [internal/promotion/engine.go](../internal/promotion/engine.go#L323) | เปิดช่องให้เพิ่ม strategy ใหม่ | บางส่วนทำงานจริง, บางส่วนยัง placeholder |
| Calculate Result | [internal/promotion/engine.go](../internal/promotion/engine.go#L220) | สรุป totals และ applied/skipped promotions | `PricingService` |
| Validate Discount | [internal/promotion/engine.go](../internal/promotion/engine.go#L314), [internal/promotion/engine.go](../internal/promotion/engine.go#L223) | กันไม่ให้ discount เกิน base และไม่ให้ final total ติดลบ | final result |
| Calculation Result | [internal/dto/pricing_dto.go](../internal/dto/pricing_dto.go#L50), [internal/service/pricing_service.go](../internal/service/pricing_service.go#L132) | shape ของ response JSON | handler ส่งกลับ client |
| Calculation Log Service | [internal/service/pricing_service.go](../internal/service/pricing_service.go#L184), [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L21) | persist และ replay calculation snapshot | MySQL / audit API |

## 1. Request เข้ามาอย่างไร

### Route layer

route สำหรับ pricing อยู่ที่:

- `POST /api/v1/pricing/calculate` ที่ [cmd/server/main.go](../cmd/server/main.go#L90)
- `POST /api/v1/pricing/explain` ที่ [cmd/server/main.go](../cmd/server/main.go#L91)

สิ่งนี้สอดคล้องกับภาพที่มี `Go Fiber API Post /api/v1/pricing/calculate`

### Handler layer

handler รับ request และ parse body ที่ [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L17)

ลำดับใน `Calculate()`:

1. สร้างตัวแปร `dto.PricingCalculateRequest`
2. ใช้ `c.BodyParser(&req)` parse body
3. เรียก `h.service.Calculate(...)`
4. ถ้า error ให้ map กลับเป็น HTTP response
5. ถ้าสำเร็จให้ `c.JSON(res)`

สิ่งนี้หมายความว่า handler มีหน้าที่ชัดเจน:

- ไม่คำนวณราคาเอง
- ไม่ query DB เอง
- ไม่แตะ business rule เอง
- ทำหน้าที่เป็น HTTP boundary

### Request DTO

request shape ถูกนิยามไว้ที่ [internal/dto/pricing_dto.go](../internal/dto/pricing_dto.go#L14)

field สำคัญ:

- `userId`
- `items`
- `couponCodes`
- `paymentMethod`
- `shipping`
- `currency`

นี่คือข้อมูลที่ภาพต้องการให้ไหลเข้า engine เพื่อเอาไปเช็ก target, condition และ strategy

## 2. Middleware ทำอะไรบ้าง

middleware ที่มีจริงในระบบตอนนี้คือ `RequestID`

โค้ดอยู่ที่ [internal/middleware/request_id.go](../internal/middleware/request_id.go#L8)

สิ่งที่ middleware ตัวนี้ทำ:

1. อ่าน `X-Request-ID` จาก request header
2. ถ้าไม่มี ให้ generate UUID ใหม่
3. set กลับไปที่ response header
4. เก็บไว้ใน `c.Locals("request_id", requestID)`
5. ส่งต่อไป handler ถัดไป

middleware นี้ถูก mount ทั้ง app ที่ [cmd/server/main.go](../cmd/server/main.go#L28)

### ข้อสรุปของ middleware ในบริบทภาพ

ภาพเขียน middleware เป็นกล่องรวม:

- Validate Request
- Auth
- Request ID
- Error Handling

โค้ดปัจจุบันตรงบางส่วน:

- `Request ID` มีจริง
- request validation มีทั้งใน handler และ service แต่ไม่ได้เป็น middleware แยก
- auth ยังไม่มี
- error handling เป็น per-handler function มากกว่า centralized middleware

ดังนั้น design โดยรวมรองรับการแยก concern แล้ว แต่ implementation ตอนนี้ยังไม่ได้แตก middleware ครบทุกหัวข้อในภาพ

## 3. Pricing Service ทำหน้าที่อะไร

`PricingService` คือ orchestration center ของ flow การคำนวณราคา

อ้างอิง:

- type และ constructor ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L26)

service นี้ถือ dependencies 3 ตัว:

- `db`
- `productRepo`
- `promotionRepo`
- `calculator`

### เมทอดหลัก

- `Calculate()` ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L48)
- `Explain()` ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L52)
- `Preview()` ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L56)

ทั้งสามเมทอดวิ่งเข้าฟังก์ชันกลางตัวเดียวคือ `calculate(...)` ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L60)

### ทำไม design นี้ดี

design นี้ support logic ในภาพได้ดีเพราะ:

- route layer ไม่ต้องรู้ business rule
- engine ไม่ต้องรู้เรื่อง HTTP หรือ DB fetch logic
- service เป็นตัวประกอบ context ทั้งหมดก่อนส่งเข้า engine

## 4. Pricing Service เตรียมข้อมูลอย่างไร

### 4.1 ตรวจ request ว่ามี items หรือไม่

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L61)

ถ้าไม่มี items:

- return `ErrEmptyOrderItems`

### 4.2 aggregate items

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L65)

ฟังก์ชัน `aggregateItems()` ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L210) ทำ 2 อย่าง:

- validate ว่า quantity ต้องมากกว่า 0
- รวมสินค้า product เดียวกันให้เป็นรายการเดียว

ผลของ design นี้คือ:

- engine ได้ input ที่สะอาดขึ้น
- ลดความซับซ้อน downstream

### 4.3 โหลดสินค้า

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L70)

service เรียก `productRepo.FindByIDs(...)`

repository implementation อยู่ที่ [internal/repository/product_repository.go](../internal/repository/product_repository.go#L39)

สิ่งที่เกิดขึ้น:

- query `WHERE id IN ?`
- ได้ข้อมูลจริงของสินค้า เช่น SKU, name, category, price, currency, status

### 4.4 ตรวจสถานะสินค้า

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L79)

service ตรวจว่า:

- สินค้าทุกตัวต้องมีอยู่จริง
- สินค้าทุกตัวต้อง `ACTIVE`

นี่ช่วยให้ engine ไม่ต้องคำนวณบนสินค้า invalid

### 4.5 normalize currency

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L86)

กติกาปัจจุบัน:

- ถ้า client ไม่ส่ง currency มา ใช้ `THB`
- ถ้าส่งมาแต่ไม่ใช่ `THB` ให้ error
- ถ้าสินค้าแต่ละตัว currency ไม่ตรงกับ request ก็ error

### 4.6 แปลง product data เป็น `CalculationItem`

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L94)

service สร้าง `promotion.CalculationItem` ซึ่งเป็น input ที่ engine ต้องการ

field สำคัญ:

- `ProductID`
- `SKU`
- `ProductName`
- `CategoryID`
- `Quantity`
- `UnitPrice`

ตรงนี้คือจุดเชื่อมสำคัญระหว่าง repository layer กับ engine layer

## 5. Pricing Service โหลด promotions อย่างไร

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L113)

service เรียก `promotionRepo.FindActivePromotions(ctx, time.Now())`

implementation อยู่ที่ [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L66)

สิ่งที่ repository ทำ:

1. preload `Conditions`
2. preload `Actions`
3. preload `Targets`
4. filter เฉพาะ `status = ACTIVE`
5. filter เฉพาะ promotion ที่เวลาปัจจุบันอยู่ในช่วง `starts_at <= now <= ends_at`
6. sort ตาม `scope`, `priority`, `created_at`, `id`

ผลคือ `PricingService` ได้ promotions ที่พร้อมใช้จริงใน engine แล้ว

## 6. Promotion data structure support logic ในภาพอย่างไร

promotion model อยู่ที่ [internal/model/models.go](../internal/model/models.go#L69)

field ที่สัมพันธ์กับภาพโดยตรง:

- `Scope` รองรับ `ITEM`, `CART`, `COUPON`, `SHIPPING`
- `Priority` ใช้จัดลำดับ
- `Stackable`
- `Exclusive`
- `StopProcessing`
- `ConflictGroup`
- `Targets`
- `Conditions`
- `Actions`

การแยก `header + targets + conditions + actions` ทำให้ design support ภาพได้ดีมาก เพราะ:

- target check ไม่ต้องฝังอยู่ใน route
- condition check ไม่ต้อง hardcode promo รายตัว
- action strategy เปลี่ยนได้จาก data

## 7. Pricing Service ส่ง context เข้า Promotion Engine อย่างไร

ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L118)

service ส่ง `promotion.CalculationContext` เข้า engine โดยประกอบ:

- `Now`
- `UserID`
- `Currency`
- `CouponCodes`
- `PaymentMethod`
- `ShippingMethod`
- `Items`
- `Promotions`

จุดนี้ตรงกับภาพมาก เพราะทุกข้อมูลที่ใช้ในกล่อง:

- Check Target
- Check Condition
- Sort Promotion
- Stacking Policy
- Strategy Pattern

ถูกประกอบเสร็จแล้วก่อนเข้าคำนวณ

## 8. Promotion Engine เริ่มทำงานอย่างไร

entry point ของ engine อยู่ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L94)

ฟังก์ชัน `Calculate(...)` ทำ 4 งานใหญ่:

1. normalize input
2. sort promotions
3. วน promotion ทีละตัวเพื่อ evaluate และ apply
4. สรุปผล totals และผลลัพธ์

### 8.1 ตั้งค่า currency default

ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L97)

### 8.2 sort promotions

ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L101)

engine sort ตาม:

1. scope rank
2. priority
3. createdAt
4. id

scope rank อยู่ที่:

- `ITEM = 1`
- `CART = 2`
- `COUPON = 3`
- `SHIPPING = 4`

อ้างอิง:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L101)

นี่คือ implementation ของกล่อง `3. Sort Promotion`

### 8.3 สร้าง result object

ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L116)

result จะเก็บ:

- `CalculationID`
- `Currency`
- `Items`
- `AppliedPromotions`
- `SkippedPromotions`
- `DecisionTrace`
- `Snapshot`

### 8.4 คำนวณ original total

ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L126)

แต่ละ item ถูกตั้งค่า:

- `OriginalAmount = UnitPrice * Quantity`
- `DiscountAmount = 0`
- `FinalAmount = OriginalAmount`

นี่คือ base ก่อน promo ใด ๆ ถูก apply

## 9. Loop หลักของ engine ทำงานอย่างไร

loop หลักเริ่มที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L141)

ภายใน loop มีการตัดสินใจเป็นลำดับชัดเจนมาก:

1. active check
2. scope check
3. stacking / exclusive policy
4. conflict group check
5. target check
6. condition check
7. apply strategy
8. update applied/skipped result
9. stop policy

นี่คือหัวใจของ logic ทั้งระบบ

## 10. Check 1: Promotion active หรือไม่

ใน loop มีการเรียก `isPromotionActive(...)`

อ้างอิง:

- จุดเรียกที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L142)

ผลลัพธ์:

- ถ้าไม่ active จะไม่ถูก apply
- ไม่ถูกใส่เป็น skipped reason
- แค่ถูกข้ามไปเงียบ ๆ

## 11. Check 2: Scope ถูกต้องหรือไม่

ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L145)

ถ้า scope ไม่อยู่ในชุดที่รองรับ:

- ถูกใส่ `INVALID_SCOPE`
- ถูกบันทึกใน `SkippedPromotions`

จุดนี้ช่วยให้ engine ป้องกัน config ที่ไม่ปลอดภัยใน runtime

## 12. Check 3: Stacking Policy

นี่คือกล่อง `4. Stacking Policy` ในภาพ และเป็นส่วนที่เพิ่งถูกเพิ่ม logic ชัดเจนขึ้น

อ้างอิงหลัก:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L149)
- fields ใน model ที่ [internal/model/models.go](../internal/model/models.go#L74)

### กติกาที่ enforce ตอนนี้

#### `exclusive`

- ถ้ามี exclusive promo apply ไปแล้ว promo ถัดไปจะถูก skip ด้วย `EXCLUSIVE_ALREADY_APPLIED`
- ถ้า promo ตัวใหม่เป็น exclusive แต่มี promo อื่น apply ไปก่อนแล้ว จะถูก skip ด้วย `EXCLUSIVE_CANNOT_STACK`
- ถ้า exclusive promo apply สำเร็จ engine จะ `break` ทันที จึงมักไม่เห็น promo หลังจากนั้นถูกบันทึกเป็น skipped ในรอบเดียวกัน

#### `stackable=false`

- ถ้ามี non-stackable promo apply ไปแล้ว promo ถัดไปจะถูก skip ด้วย `NON_STACKABLE_ALREADY_APPLIED`
- ถ้า promo ตัวใหม่เป็น non-stackable แต่มี promo อื่น apply ไปก่อนแล้ว จะถูก skip ด้วย `NON_STACKABLE_CANNOT_STACK`

#### `stopProcessing`

- ถ้า promo ตัวที่ apply สำเร็จมี `stopProcessing=true` engine จะหยุด loop ทันที
- promo หลังจากนั้นจะไม่ถูกประเมินต่อ และจะไม่ถูกบันทึกเป็น skipped ในผลลัพธ์รอบนั้น

ผลของ design นี้คือ:

- ลำดับการตัดสินใจ deterministic
- ใช้ `priority` และ sort order เป็นตัวตัดสินก่อนหลัง
- ไม่ต้องหา combination ที่ดีที่สุดทุกแบบ

## 13. Check 4: Conflict Group

ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L165)

logic:

- ถ้า promotion อยู่ใน `conflictGroup`
- และ group นั้นเคยมี promo apply ไปแล้ว
- promo ถัดไปจะถูก skip ด้วย `CONFLICT_GROUP_BLOCKED`

ผลเชิง design:

- ใช้สำหรับ promo ที่ไม่ควรอยู่ร่วมกันเฉพาะกลุ่ม
- เป็น policy คนละตัวกับ `exclusive`

## 14. Check 5: Target

จุดเรียกอยู่ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L169)

logic จริงอยู่ที่:

- `evaluateTargets(...)` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L460)
- `matchedItemIndexes(...)` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L491)

target types ที่รองรับจริง:

- `CART`
- `PRODUCT`
- `CATEGORY`

สิ่งที่ target check ทำ:

- ถ้าไม่มี target เลย ให้ถือว่า match
- ถ้า target เป็น `CART` ให้ match ทั้ง order
- ถ้า target เป็น `PRODUCT` ให้หา productId ใน items
- ถ้า target เป็น `CATEGORY` ให้หา categoryId ใน items

ถ้าไม่ match:

- ใส่ `TARGET_MISMATCH` ลง `SkippedPromotions`

## 15. Check 6: Condition

จุดเรียกอยู่ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L173)

condition evaluation function:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L259)

การทำงาน:

1. วนทุก condition ของ promo
2. ใช้ `registry.Condition(condition.ConditionType)` หา handler
3. เรียก handler พร้อม `ConditionContext`
4. ถ้ามีตัวใดไม่ผ่าน ให้ promo นั้นถูก skip ทันที

condition handlers ที่มีจริง:

- `MIN_ORDER_AMOUNT` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L327)
- `MAX_ORDER_AMOUNT` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L335)
- `COUPON_CODE` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L343)
- `PAYMENT_METHOD` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L353)
- `PRODUCT_ID` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L361)
- `CATEGORY_ID` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L371)
- `DATE_RANGE` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L381)

condition ที่ยังเป็น passthrough:

- `USER_SEGMENT`
- `FIRST_ORDER`

registry mapping อยู่ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L89)

## 16. Check 7: Strategy Pattern

นี่คือกล่อง `5. Strategy Pattern`

registry definition อยู่ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L24)

แนวคิด:

- engine ไม่รู้รายละเอียดของทุก action type
- engine แค่ถาม registry ว่า `ActionType` นี้ใช้ handler ไหน

จุดเรียก action handler อยู่ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L233)

ถ้าไม่มี action handler:

- return `ACTION_STRATEGY_NOT_SUPPORTED`

นี่ทำให้ design รองรับการเพิ่ม strategy ใหม่ได้โดยไม่ต้องแก้ flow หลักของ engine มาก

## 17. Percentage Strategy ทำงานอย่างไร

`percentageActionHandler` อยู่ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L284)

logic:

- ถ้าเป็น `ITEM` scope จะเอาฐานจาก matched items เท่านั้น
- ถ้าเป็น `CART` scope จะเอาฐานจาก `CartBase`
- คำนวณ `discount = base * basisPoints / 10000`
- ถ้ามี `MaxDiscountAmount` ก็ cap ไม่ให้เกิน

ผลเชิง business:

- ใช้ได้ทั้ง item-level และ cart-level percentage

## 18. Fixed Amount Strategy ทำงานอย่างไร

`fixedAmountActionHandler` อยู่ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L302)

logic:

- อ่าน `ValueAmount`
- ถ้าเป็น item scope ใช้ฐานจาก matched items
- ถ้าเป็น cart scope ใช้ฐานจาก `CartBase`
- cap ไม่ให้ discount เกิน base
- ถ้ามี `MaxDiscountAmount` ก็ cap ซ้ำอีกชั้น

ผลเชิง business:

- กันไม่ให้ส่วนลดมากกว่ายอดที่ลดได้จริง

## 19. Future Strategy ในโค้ดปัจจุบัน

`FREE_SHIPPING` ถูก register แล้วที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L81)

แต่ handler ปัจจุบันคืน `0` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L323)

แปลว่า:

- architecture รองรับ future strategy
- แต่ runtime behavior ของ strategy นี้ยังเป็น placeholder

ถ้าต้องการรายละเอียดเชิง gap analysis เฉพาะส่วนนี้ ดูเพิ่มได้ที่ [Docs/FUTURE_STRATEGY_GAP_ANALYSIS.md](./FUTURE_STRATEGY_GAP_ANALYSIS.md#L1)

## 20. Apply action ลง item/cart อย่างไร

หลัง handler คืน discount amount มาแล้ว engine จะเอา discount ไป apply จริง

จุดหลัก:

- `applyPromotion(...)` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L230)
- `applyToItems(...)` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L399)
- `applyToCart(...)` อยู่ถัดจากนั้นในไฟล์เดียวกัน

### Apply to items

`applyToItems(...)` ทำงานแบบนี้:

1. หา matched items
2. หาฐานรวมของ items เหล่านั้น
3. เฉลี่ย discount ตามสัดส่วน `FinalAmount`
4. กันไม่ให้ลดเกิน `FinalAmount` ของ item
5. ถ้ามีเศษ discount ที่ยังเหลือ ค่อยไล่เติมทีละ item

ผลคือ:

- item แต่ละตัวจะได้ `DiscountAmount` และ `FinalAmount` ใหม่

### Apply to cart

`applyToCart(...)` ใช้หลักการคล้ายกัน แต่ฐานคือยอดรวม cart ณ เวลานั้น

ผลคือ:

- ส่วนลด cart ถูกกระจายลง item เพื่อให้สุดท้าย item-level result และ order-level result ตรงกัน

## 21. Validate Discount / Guardrail ของผลลัพธ์

แม้ในภาพจะมี box `Validate Discount` แยกต่างหาก แต่ในโค้ดปัจจุบันมันกระจายอยู่ใน logic คำนวณ

ตัวอย่าง guardrail สำคัญ:

- fixed amount ห้ามเกิน base ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L314)
- apply to item ห้ามลดเกิน `FinalAmount` ของ item ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L421)
- final total ห้ามติดลบที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L223)

ดังนั้นแม้จะไม่มี service หรือ function ชื่อ `ValidateDiscount` โดยตรง แต่ behavior ตามภาพมีจริง

## 22. Result สุดท้ายของ engine คืออะไร

ท้าย loop engine สรุปผลที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L220)

field หลัก:

- `OriginalTotal`
- `DiscountTotal`
- `FinalTotal`
- `AppliedPromotions`
- `SkippedPromotions`

และเก็บ metadata เพิ่ม:

- `DecisionTrace`
- `Snapshot["scopeOrder"]`

นี่คือ implementation ของกล่อง `Calculation Result`

## 23. Pricing Service แปลง engine result เป็น API response อย่างไร

mapping อยู่ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L132)

service แปลง:

- `CalculationResult.Items` -> `[]dto.PricingItemResponse`
- `AppliedPromotions` -> `[]dto.PricingPromotionAppliedResponse`
- `SkippedPromotions` -> `[]dto.PricingPromotionSkippedResponse`

response shape จริงอยู่ที่ [internal/dto/pricing_dto.go](../internal/dto/pricing_dto.go#L50)

ดังนั้น client จะได้ result ในรูปที่เสถียรและอ่านง่าย:

- totals
- item breakdown
- applied promotions
- skipped promotions

## 24. Error ถูกแปลงเป็นผลลัพธ์ HTTP อย่างไร

handler map business errors กลับเป็น HTTP status code ที่ [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L45)

ตัวอย่าง:

- `ErrEmptyOrderItems` -> `422 EMPTY_ORDER_ITEMS`
- `ErrInvalidQuantity` -> `422 INVALID_QUANTITY`
- `ErrProductNotFound` -> `404 PRODUCT_NOT_FOUND`
- `ErrProductInactive` -> `422 PRODUCT_INACTIVE`
- `ErrCurrencyMismatch` -> `422 CURRENCY_MISMATCH`
- `ErrCalculationFailed` -> `500 CALCULATION_FAILED`

error response shape จริงอยู่ที่ [internal/handler/api_error.go](../internal/handler/api_error.go#L1)

ผลคือ output ของระบบไม่ได้มีแค่ success result แต่มี error result ที่ deterministic เช่นกัน

## 25. Calculation Log ถูกสร้างอย่างไร

หลังคำนวณเสร็จ `PricingService` จะ persist log ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L175)

ฟังก์ชัน `persistCalculationLog(...)` จะเก็บ:

- request
- response
- `explain`
- `decisionTrace`
- `scopeOrder`

ทั้งหมดถูก marshaled เป็น JSON snapshot ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L184)

row ที่บันทึกลง DB คือ `model.PromotionCalculationLog` ที่ [internal/model/models.go](../internal/model/models.go#L137)

field สำคัญ:

- `CalculationID`
- `RequestID`
- `UserID`
- `OriginalTotal`
- `DiscountTotal`
- `FinalTotal`
- `AppliedPromotionsJSON`
- `SkippedPromotionsJSON`
- `CalculationSnapshotJSON`

นี่คือ implementation จริงของกล่อง `Calculation Log Service สร้าง Snapshot การคำนวณ`

## 26. Calculation Log API ใช้ทำอะไรต่อ

นอกจาก persist ตอน pricing แล้ว ระบบยังมี `CalculationLogService` สำหรับ query และ replay

อ้างอิง:

- service definition ที่ [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L21)
- repository ที่ [internal/repository/calculation_log_repository.go](../internal/repository/calculation_log_repository.go#L21)

### List

`List(...)` ที่ [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L45)

ใช้ดู history ของ calculation logs

### Detail

`GetByCalculationID(...)` ที่ [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L81)

ใช้ดู:

- summary
- applied promotions
- skipped promotions
- snapshot เต็ม

### Replay

`Replay(...)` ที่ [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L104)

logic:

1. โหลด snapshot เดิม
2. เอา request เดิมมายิง `pricing.Preview(...)`
3. compare กับผลเดิม
4. บอกว่า `Matched` หรือไม่ พร้อม `Differences`

จุดนี้ทำให้ design ของระบบ support เรื่อง audit/debug ได้ดีมาก

## 27. Product Repository support flow อย่างไร

`ProductRepository` ทำหน้าที่เป็น source ของ “ราคาสินค้าจริง”

อ้างอิง:

- [internal/repository/product_repository.go](../internal/repository/product_repository.go#L39)

ในบริบทภาพ repository นี้ support flow โดยตรงเพราะ:

- handler ไม่รู้ DB schema
- service ไม่ต้องเขียน SQL เอง
- engine ไม่ต้อง query DB เอง

data ที่ repo โหลดมา support logic ต่อ:

- target check ผ่าน `ProductID`, `CategoryID`
- base price ผ่าน `PriceAmount`
- item response ผ่าน `SKU`, `Name`

## 28. Promotion Repository support flow อย่างไร

`PromotionRepository` เป็น source ของ “promotion configuration จริง”

อ้างอิง:

- [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L66)

สิ่งที่ repo นี้ support ให้ engine:

- status filtering
- active window filtering
- preload complete rule graph
- order เดียวกับที่ engine ใช้ตัดสินใจต่อ

นี่คือเหตุผลที่ design ตอบโจทย์ `data-driven promotion engine` ได้จริง

## 29. โค้ดแต่ละชั้น support กันอย่างไร

### Handler support Service

- handler parse request
- handler map error -> HTTP
- handler ไม่ทำ business logic

### Service support Engine

- service normalize request
- service load products + promotions
- service build calculation context
- service persist result log

### Repository support Service

- repository แยก concern ของ DB access ออกจาก service
- service จึงโฟกัส orchestration ได้

### Engine support Result

- engine เป็นจุดรวมของ target, condition, policy, strategy
- ผลลัพธ์ที่ได้ถูก format ต่อได้ทั้งเป็น API response และ log snapshot

### Calculation log support Debug / Replay

- explain/calculate ไม่ได้ให้แค่ผลตัวเลข
- แต่ให้ evidence สำหรับ audit ย้อนหลังด้วย

## 30. ผลลัพธ์สุดท้ายของระบบคืออะไร

ถ้ามองตามภาพ ระบบไม่ได้จบแค่ “มีส่วนลด” แต่จบที่ “ได้ final price ที่อธิบายได้”

ผลลัพธ์สุดท้ายของโค้ดปัจจุบันมี 3 ระดับ:

### ระดับ 1: HTTP response

client ได้:

- `calculationId`
- `originalTotal`
- `discountTotal`
- `finalTotal`
- `items`
- `appliedPromotions`
- `skippedPromotions`

อ้างอิง:

- [internal/dto/pricing_dto.go](../internal/dto/pricing_dto.go#L50)

### ระดับ 2: Engine trace

ระบบเก็บ:

- `DecisionTrace`
- `scopeOrder`

อ้างอิง:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L226)

### ระดับ 3: Audit snapshot

ระบบ persist:

- request
- response
- applied/skipped promotions
- trace context

อ้างอิง:

- [internal/service/pricing_service.go](../internal/service/pricing_service.go#L184)

## 31. สรุปเชิง design และ logic

ถ้าสรุปจากโค้ดปัจจุบัน:

- design ของระบบแยก layer ชัดเจน
- flow หลักในภาพถูกสะท้อนในโค้ดจริงเกือบครบ
- logic ของ pricing อยู่รวมศูนย์ใน promotion engine
- service layer ทำหน้าที่ orchestration ได้ถูกบทบาท
- repository layer ทำหน้าที่เป็น data boundary ได้ดี
- result ไม่ได้มีแค่ total แต่มี applied/skipped breakdown และ audit log ด้วย

### จุดที่โค้ดสอดคล้องกับภาพชัดที่สุด

- `Pricing Service -> Repository -> Engine -> Result`
- target/condition/action แบบ data-driven
- sort by scope/priority
- stacking policy
- result breakdown
- calculation log + replay

### จุดที่ควรรู้เวลาพูดเรื่อง logic/result

- `FREE_SHIPPING` ยังเป็น placeholder
- `USER_SEGMENT` และ `FIRST_ORDER` ยังเป็น passthrough condition
- middleware ในภาพยังไม่ได้แตกครบทุกหัวข้อเป็น component แยก
- request ID middleware มีแล้ว แต่ calculation log ตอน persist ใช้ request id ที่ generate ใหม่ใน service

## 32. ประโยคสรุปที่ใช้ตอบพี่ได้เลย

ถ้าจะตอบในมุม design:

> ระบบแยก layer ชัดเจนและ support ภาพ architecture ได้ดี โดยใช้ service เป็น orchestration layer, repository เป็น data access layer และ promotion engine เป็น business rule core

ถ้าจะตอบในมุม logic:

> flow จริงของการคำนวณเริ่มจาก parse request, โหลดสินค้าและ promotions ที่ active, sort promotions ตาม scope และ priority, check target/condition, enforce stacking policy, apply strategy, แล้วสรุปออกมาเป็น final total พร้อม applied/skipped breakdown

ถ้าจะตอบในมุม result:

> ผลลัพธ์สุดท้ายไม่ได้มีแค่ราคาสุทธิ แต่มี item breakdown, appliedPromotions, skippedPromotions และ calculation snapshot สำหรับ audit/replay ทำให้ผลลัพธ์อธิบายย้อนกลับได้
