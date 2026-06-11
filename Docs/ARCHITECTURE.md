# Architecture

เอกสารนี้สรุป design ปัจจุบันของระบบตามโค้ดจริง

## โครงสร้างหลัก

flow หลักของระบบคือ:

```text
HTTP Handler -> Service -> Repository -> MySQL
                       -> Promotion Engine
```

ความหมายของแต่ละชั้น:

- [internal/handler](../internal/handler) รับ request, parse body, map error เป็น HTTP response
- [internal/service](../internal/service) รวม business flow เช่น pricing, promotion lifecycle, confirm order
- [internal/repository](../internal/repository) query ข้อมูลด้วย Gorm
- [internal/promotion](../internal/promotion) คำนวณ promotion จริง
- [internal/model](../internal/model) นิยาม model ที่ map กับ table

## Design Pattern ที่ใช้

- `Service Layer` แยก orchestration ของ use case ออกจาก HTTP
- `Repository Pattern` แยก query DB ออกจาก business logic
- `Rule Engine` ให้ promotion ถูกเก็บเป็น data แทน hardcode
- `Strategy + Registry` ให้ action/condition ขยายเพิ่มได้โดยไม่ต้อง rewrite engine loop

## โครงสร้าง promotion

promotion 1 ตัวถูกแยกข้อมูลเป็น 4 ส่วน:

- `promotions` header ของ promo
- `promotion_targets` สิ่งที่ promo เล็งไปหา เช่น product, category, cart
- `promotion_conditions` เงื่อนไขที่ต้องผ่านก่อน apply
- `promotion_actions` วิธีลดราคา

แนวคิดนี้ทำให้เพิ่ม promotion ใหม่แบบเดิมได้โดยเพิ่ม data ผ่าน API/DB

## ลำดับการคำนวณ

ระบบ sort promotion ก่อนคำนวณด้วย:

1. `scope`
2. `priority`
3. `created_at`
4. `id`

scope order ที่ engine ใช้จริง:

1. `ITEM`
2. `CART`
3. `COUPON`
4. `SHIPPING`

promotion ถัดไปจะคำนวณบนยอดที่ถูกลดจาก promotion ก่อนหน้าแล้ว

## กติกาที่ใช้จริงตอนนี้

- ใช้เฉพาะ promotion ที่ `status = ACTIVE` และอยู่ในช่วงเวลาใช้งาน
- เช็ก `target` ก่อน
- เช็ก `condition` ถัดมา
- enforce `exclusive` และ `stackable` ก่อน apply promotion
- ถ้า `conflict_group` ซ้ำกับ promo ที่ apply ไปแล้ว จะถูก skip
- ถ้า `stop_processing = true` จะหยุดคำนวณหลัง promo นั้น
- ผลลัพธ์ถูกเก็บเป็น `appliedPromotions`, `skippedPromotions`, item totals และ calculation log

## Action และ Condition ที่รองรับจริง

action ที่ registry รองรับตอนนี้:

- `PERCENTAGE_DISCOUNT`
- `FIXED_AMOUNT_DISCOUNT`
- `CART_PERCENTAGE_DISCOUNT`
- `CART_FIXED_AMOUNT_DISCOUNT`
- `FREE_SHIPPING`

condition ที่ registry รองรับตอนนี้:

- `PRODUCT_ID`
- `CATEGORY_ID`
- `MIN_ORDER_AMOUNT`
- `MAX_ORDER_AMOUNT`
- `COUPON_CODE`
- `USER_SEGMENT`
- `FIRST_ORDER`
- `PAYMENT_METHOD`
- `DATE_RANGE`

## ข้อจำกัดของ implementation ปัจจุบัน

มี field บางตัวใน schema/model ที่ยังไม่ได้ถูกใช้เต็มรูปแบบใน engine:

- `stackable` ถูกใช้ตัดสินใจจริงแล้ว โดย enforce ด้วย `NON_STACKABLE_CANNOT_STACK` และ `NON_STACKABLE_ALREADY_APPLIED`
- `exclusive` ถูกใช้ตัดสินใจจริงแล้ว โดย enforce ด้วย `EXCLUSIVE_CANNOT_STACK` และ `EXCLUSIVE_ALREADY_APPLIED`
- `operator` ใน condition ยังมีไว้รองรับ design มากกว่า logic เต็มรูปแบบ
- `logical_operator` ยังไม่ได้ evaluate แบบ AND/OR group จริง
- `FREE_SHIPPING` ยังไม่ลดราคาจริงใน engine ปัจจุบัน
- enum ใน schema มีบาง action type ที่ยังไม่มี handler จริง เช่น `BUY_X_GET_Y`, `BUNDLE_DISCOUNT`

## การเก็บ log และ audit

ทุกครั้งที่ `pricing/calculate` และ `pricing/explain` รันสำเร็จ ระบบจะบันทึก:

- `calculation_id`
- `original_total`
- `discount_total`
- `final_total`
- `applied_promotions_json`
- `skipped_promotions_json`
- `calculation_snapshot_json`

ตอน `orders/confirm` ระบบจะ recalculate อีกครั้ง และบันทึก order + promotion usage ภายใน transaction

## Gorm ใช้ตรงไหน

- [internal/database/gorm.go](../internal/database/gorm.go) เปิด connection
- [internal/model/models.go](../internal/model/models.go) นิยาม model และ gorm tags
- [internal/repository](../internal/repository) ใช้ query, preload, save, list
- [internal/service/promotion_service.go](../internal/service/promotion_service.go) และ [internal/service/order_service.go](../internal/service/order_service.go) ใช้ transaction และ locking
