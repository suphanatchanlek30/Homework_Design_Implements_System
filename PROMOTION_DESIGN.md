# Promotion Design Notes

เอกสารนี้อธิบายการออกแบบระบบคำนวณราคาสุทธิจากโค้ดในโปรเจกต์นี้เท่านั้น โดยโฟกัส 3 คำถามหลัก:

1. ถ้ารองรับ promotion แบบซ้อนกัน ต้องจัดลำดับคำนวณยังไง
2. ถ้าต้องการเพิ่ม promotion จะทำยังไงเพื่อไม่ให้กระทบ logic เดิม
3. ถ้าต้องการเพิ่ม promotion ใหม่ที่ไม่เคยมีมาก่อน จะ design ยังไงให้รองรับ

---

## 1) ภาพรวมการออกแบบจากโค้ดจริง

โครงหลักของระบบคำนวณราคาอยู่ที่:

- `internal/service/pricing_service.go`
- `internal/promotion/engine.go`
- `internal/promotion/registry.go`
- `internal/model/models.go`
- `internal/repository/promotion_repository.go`

แนวคิดที่ใช้จริงในโค้ดคือ:

- แยก **pricing service** ออกจาก **promotion engine**
- ให้ **promotion engine** เป็นตัวคำนวณจริง
- ให้ **registry** เป็นตัวจับคู่ `actionType` / `conditionType` กับ handler
- ให้ข้อมูล promotion อยู่ในฐานข้อมูลเป็น rule data ไม่ hardcode ไว้ใน controller

สรุปสั้น ๆ: ระบบนี้ใช้แนวทาง **Rule Engine + Strategy Pattern + Repository Pattern**

---

## 2) ถ้า promotion ซ้อนกัน ต้องจัดลำดับคำนวณยังไง

### ลำดับหลัก

ใน `internal/promotion/engine.go` ระบบ sort promotion ตามลำดับนี้ก่อนคำนวณ:

1. `scope`
2. `priority`
3. `created_at`
4. `id`

โค้ดใช้ scope order เป็น:

- `ITEM`
- `CART`
- `COUPON`
- `SHIPPING`

อ้างอิง:

- `internal/promotion/engine.go:94`
- `internal/promotion/engine.go:101`
- `internal/promotion/engine.go:189`

### ความหมายของลำดับนี้

- `ITEM` ต้องมาก่อน เพราะเป็น promotion ที่ผูกกับสินค้ารายชิ้น
- `CART` ค่อยตามหลัง เพราะใช้ยอดรวมที่เหลือหลัง item discount
- `COUPON` และ `SHIPPING` อยู่ถัดไปตามลำดับ scope

### วิธีคิดตอนคำนวณจริง

engine ใช้ `FinalAmount` ของ item ที่เปลี่ยนไปแล้วเป็นฐานของรอบถัดไป

ดังนั้น promotion ที่ซ้อนกันจะไม่ได้คำนวณจากราคาเริ่มต้นทั้งหมดทุกครั้ง
แต่คำนวณจากยอดที่เหลือหลัง promotion ก่อนหน้า

อ้างอิง:

- `internal/promotion/engine.go:138`
- `internal/promotion/engine.go:186`

### ตัวอย่างตามโจทย์

ถ้ามี:

- สินค้า 1 ลด 10%
- สินค้า 2 ลด 100 บาท

ระบบจะ:

1. สร้าง `CalculationItem` จากราคาสินค้าจริง
2. หา promotion ที่ target สินค้า 1
3. apply discount ของสินค้า 1 ก่อน
4. ค่อย apply promotion ของสินค้า 2
5. สรุป `OriginalTotal`, `DiscountTotal`, `FinalTotal`

อ้างอิง:

- `internal/service/pricing_service.go:70`
- `internal/service/pricing_service.go:118`
- `internal/promotion/engine.go:159`

---

## 3) ถ้าต้องการเพิ่ม promotion โดยไม่กระทบ logic เดิม ต้องทำยังไง

### ใช้ registry แทน hardcode

ใน `internal/promotion/registry.go` มี registry สำหรับ:

- `ActionHandler`
- `ConditionHandler`

handler แต่ละตัวถูก register ด้วย `actionType` หรือ `conditionType`

อ้างอิง:

- `internal/promotion/registry.go:20`
- `internal/promotion/registry.go:22`
- `internal/promotion/registry.go:29`

### ผลที่ได้

ถ้าจะเพิ่ม promotion ใหม่ที่ยังใช้ primitive เดิมได้
เช่น:

- target ใหม่
- condition ใหม่
- action ใหม่

ให้เพิ่ม handler ใน registry แล้วไม่ต้องเปลี่ยน flow หลักใน engine

อ้างอิง:

- `internal/promotion/engine.go:199`
- `internal/promotion/engine.go:228`
- `internal/promotion/registry.go:81`

### จุดที่โค้ดตรวจไว้แล้ว

`PromotionService` จะ validate ก่อนสร้าง/แก้ promotion ว่า action/condition อยู่ในชุดที่ระบบรองรับ

อ้างอิง:

- `internal/service/promotion_service.go:350`
- `internal/service/promotion_service.go:378`

ดังนั้น logic เดิมจะไม่พังเพราะข้อมูล rule ใหม่ที่ไม่ตรงรูปแบบ

---

## 4) ถ้าจะเพิ่ม promotion ใหม่ที่ไม่เคยมีมาก่อน ควร design ยังไง

### แนวคิดที่ใช้ในโค้ด

promotion ถูกแยกเป็น 3 ชั้น:

1. `promotion` = metadata หลัก
2. `promotion_targets` / `promotion_conditions` / `promotion_actions` = rule data
3. `promotion engine` = evaluator

อ้างอิง schema:

- `internal/model/models.go:69`
- `internal/model/models.go:91`
- `internal/model/models.go:101`
- `internal/model/models.go:113`

### ถ้า promotion ใหม่ยังอยู่ใน primitive เดิม

เช่น:

- target ใหม่แต่ยังเป็นสินค้า/หมวดหมู่
- condition ใหม่แต่ยังเป็น promo code / amount / payment method
- action ใหม่แต่ยังเป็น percentage / fixed amount

ก็เพิ่มข้อมูล rule ลง DB ได้เลย

### ถ้า promotion ใหม่ต้อง logic ใหม่จริง

เช่น action แบบพิเศษที่ไม่ใช่ลดเงินตรง ๆ

ให้:

1. เพิ่ม handler ใหม่ใน registry
2. ให้ promotion service ยอมรับ `actionType` ใหม่
3. ให้ engine เรียกผ่าน interface เดิม

แนวนี้ทำให้ core flow ไม่ต้องแก้

อ้างอิง:

- `internal/promotion/registry.go:39`
- `internal/promotion/registry.go:43`
- `internal/promotion/engine.go:199`

### ตัวอย่างที่พิสูจน์ได้จากโค้ด

มี test ที่ register action ใหม่ชื่อ `LOYALTY_BONUS` ได้โดยไม่แตะ engine core

อ้างอิง:

- `internal/promotion/engine_test.go:96`

---

## 5) Design pattern ที่ใช้จริงในโค้ด

### 5.1 Strategy Pattern

ใช้กับ action / condition handler

- action handler ถูก map ตาม `actionType`
- condition handler ถูก map ตาม `conditionType`

อ้างอิง:

- `internal/promotion/registry.go:20`
- `internal/promotion/registry.go:22`

### 5.2 Rule Engine

engine เป็นตัว:

- sort promotion
- ตรวจ target
- ตรวจ condition
- apply action
- รวมผลลัพธ์

อ้างอิง:

- `internal/promotion/engine.go:94`
- `internal/promotion/engine.go:138`

### 5.3 Repository Pattern

ข้อมูล promotion และ log ถูก query ผ่าน repository แทนการเขียน SQL กระจายทั่วระบบ

อ้างอิง:

- `internal/repository/promotion_repository.go:1`
- `internal/repository/calculation_log_repository.go:1`

### 5.4 Service Layer

`PricingService` เป็นตัว orchestrate:

- load product
- load active promotion
- call calculator
- persist calculation log

อ้างอิง:

- `internal/service/pricing_service.go:48`
- `internal/service/pricing_service.go:113`
- `internal/service/pricing_service.go:184`

---

## 6) Table design ที่ยืดหยุ่น

### ตารางหลัก

#### `promotion`
เก็บ metadata ของ promotion:

- code
- name
- scope
- priority
- stackable
- exclusive
- stop_processing
- status
- starts_at / ends_at
- version

อ้างอิง:

- `internal/model/models.go:69`

#### `promotion_targets`
เก็บ target แบบแยก record

อ้างอิง:

- `internal/model/models.go:91`

#### `promotion_conditions`
เก็บเงื่อนไขของ promotion

อ้างอิง:

- `internal/model/models.go:101`

#### `promotion_actions`
เก็บ action strategy ของ promotion

อ้างอิง:

- `internal/model/models.go:113`

#### `promotion_usages`
เก็บ usage ตอน confirm order

อ้างอิง:

- `internal/model/models.go:126`

#### `promotion_calculation_logs`
เก็บ snapshot สำหรับ audit / replay

อ้างอิง:

- `internal/model/models.go:137`

### ทำไม design นี้ยืดหยุ่น

- เพิ่ม promotion ใหม่ได้โดยเพิ่มข้อมูล rule
- ไม่ต้องเพิ่มคอลัมน์ใหม่ทุกครั้ง
- แยก lifecycle ของ promotion ออกจากผลคำนวณ
- ใช้ snapshot เพื่อ replay ได้

---

## 7) คำนวณโปรโมชั่นได้ถูกต้องยังไง

### 7.1 โหลดราคาจาก server-side

pricing service จะ batch load product จาก repository แล้วใช้ `PriceAmount` จาก DB

อ้างอิง:

- `internal/service/pricing_service.go:70`
- `internal/service/pricing_service.go:94`

### 7.2 ห้าม items ว่าง / quantity ต้องมากกว่า 0

มี validation ใน pricing service

อ้างอิง:

- `internal/service/pricing_service.go:61`
- `internal/service/pricing_service.go:210`

### 7.3 คำนวณจากยอดที่เหลือจริง

engine เริ่มจาก:

- set `OriginalAmount`
- set `FinalAmount`
- apply discount ทีละ promotion
- update `FinalAmount`

อ้างอิง:

- `internal/promotion/engine.go:126`
- `internal/promotion/engine.go:159`
- `internal/promotion/engine.go:189`

### 7.4 ป้องกันยอดติดลบ

การ apply ส่วนลดถูก clamp ไม่ให้เกินฐานราคา

อ้างอิง:

- `internal/promotion/engine.go:253`
- `internal/promotion/engine.go:271`
- `internal/promotion/engine.go:309`

### 7.5 มี test ยืนยัน

มี test ครอบ:

- promotion ซ้อนกัน item/cart
- promotion inactive
- custom registered action

อ้างอิง:

- `internal/promotion/engine_test.go:11`
- `internal/promotion/engine_test.go:69`
- `internal/promotion/engine_test.go:96`

---

## 8) สรุปตอบโจทย์โจทย์นี้จากโค้ด

### ถ้าถามเรื่องลำดับคำนวณ

ตอบว่า:

- ใช้ `ITEM -> CART -> COUPON -> SHIPPING`
- ในแต่ละ scope sort ต่อด้วย `priority -> created_at -> id`
- คำนวณจากยอดที่เหลือหลังรอบก่อน

อ้างอิง:

- `internal/promotion/engine.go:101`
- `internal/promotion/engine.go:138`

### ถ้าถามเรื่องเพิ่ม promotion โดยไม่กระทบ logic เดิม

ตอบว่า:

- แยก rule data ออกจาก engine
- ใช้ registry สำหรับ action/condition handler
- เพิ่ม handler ใหม่ได้โดยไม่แก้ calculate flow หลัก

อ้างอิง:

- `internal/promotion/registry.go:29`
- `internal/promotion/registry.go:39`
- `internal/promotion/registry.go:43`

### ถ้าถามเรื่อง design ให้รองรับ promotion ใหม่ที่ไม่เคยมีมาก่อน

ตอบว่า:

- เพิ่ม schema ในระดับ rule data ก่อน
- ถ้ายังใช้ primitive เดิมได้ก็ใส่ data ใหม่
- ถ้าต้อง logic ใหม่จริง ค่อย register handler ใหม่

อ้างอิง:

- `internal/model/models.go:69`
- `internal/promotion/registry.go:81`
- `internal/promotion/engine_test.go:96`

---

## 9) ประโยคสรุปแบบใช้ส่งงานได้

> ระบบนี้ออกแบบด้วย Rule Engine + Strategy Pattern + Repository Pattern โดยแยก promotion เป็น metadata และ rule data ในตาราง `promotion`, `promotion_targets`, `promotion_conditions`, `promotion_actions` แล้วให้ engine จัดลำดับ `ITEM -> CART -> COUPON -> SHIPPING` พร้อม sort ตาม `priority`, `created_at`, `id` เพื่อคำนวณแบบ deterministic; การเพิ่ม promotion ใหม่ทำได้โดยเพิ่ม rule data หรือ register strategy ใหม่ใน registry โดยไม่กระทบ logic เดิม

