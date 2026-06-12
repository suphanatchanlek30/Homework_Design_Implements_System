# Promotion Policy Reference

เอกสารนี้ทำไว้สำหรับตอบคำถามเวลาโดนถามถึงคำในภาพ เช่น `Stackable`, `Non-Stackable`, `Exclusive`, `Non-Exclusive`, `Create Exclusive Promotion`, `Promotion Usage`, `Replay Calculation Log` ว่าคืออะไร ใช้ต่างกันยังไง และ logic อยู่ตรงไหนของโค้ด

เป้าหมายของเอกสารนี้มี 3 อย่าง:

1. อธิบายคำศัพท์ในภาพให้ตอบได้เร็ว
2. พาไปยัง route, service, engine, test ที่เกี่ยวข้องได้ทันที
3. บอกข้อจำกัดของ implementation ปัจจุบันแบบตรงไปตรงมา

## เส้นทางหลักของ flow

ภาพรวมของ flow ที่เกี่ยวกับ request ในภาพ:

1. route ถูกประกาศที่ [cmd/server/main.go](../cmd/server/main.go#L60-L105)
2. handler รับ HTTP request และ parse body
3. service ทำ orchestration
4. promotion engine คำนวณ policy จริง
5. order / usage / calculation log ใช้ผลคำนวณต่อ

จุดสำคัญ:

- pricing routes: [cmd/server/main.go](../cmd/server/main.go#L90-L93)
- promotion routes: [cmd/server/main.go](../cmd/server/main.go#L78-L88)
- order routes: [cmd/server/main.go](../cmd/server/main.go#L95-L99)
- calculation log routes: [cmd/server/main.go](../cmd/server/main.go#L101-L105)

## แผนที่จาก request ในภาพไปยังโค้ด

| ในภาพ | route | handler | service/logic หลัก |
|---|---|---|---|
| Request 13: Explain Pricing With Stacked Promotions | `POST /pricing/explain` | [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L37) | [internal/service/pricing_service.go](../internal/service/pricing_service.go#L58), [internal/service/pricing_service.go](../internal/service/pricing_service.go#L70), [internal/promotion/engine.go](../internal/promotion/engine.go#L100) |
| Request 14: Calculate Pricing Final Check | `POST /pricing/calculate` | [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L21) | [internal/service/pricing_service.go](../internal/service/pricing_service.go#L52), [internal/service/pricing_service.go](../internal/service/pricing_service.go#L70) |
| Request 15: Confirm Order | `POST /orders/confirm` | [internal/handler/order_handler.go](../internal/handler/order_handler.go#L26) | [internal/service/order_service.go](../internal/service/order_service.go#L55) |
| Request 16: Check Order Detail | `GET /orders/:orderId` | [internal/handler/order_handler.go](../internal/handler/order_handler.go#L84) | [internal/service/order_service.go](../internal/service/order_service.go#L208), [internal/service/order_service.go](../internal/service/order_service.go#L291) |
| Request 17: Check Promotion Usage | `GET /promotions/:promotionId/usages` | [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L179) | [internal/service/promotion_service.go](../internal/service/promotion_service.go#L346), [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L172-L195) |
| Request 18: Check Calculation Log List | `GET /calculation-logs` | [internal/handler/calculation_log_handler.go](../internal/handler/calculation_log_handler.go#L26) | [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L49) |
| Request 19: Check Calculation Log Detail | `GET /calculation-logs/:calculationId` | [internal/handler/calculation_log_handler.go](../internal/handler/calculation_log_handler.go#L42) | [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L87) |
| Request 20: Replay Calculation Log | `POST /calculation-logs/:calculationId/replay` | [internal/handler/calculation_log_handler.go](../internal/handler/calculation_log_handler.go#L58) | [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L112) |
| Request 21: Create Non-Stackable Promotion | `POST /promotions` | [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L26) | [internal/service/promotion_service.go](../internal/service/promotion_service.go#L56) |
| Request 23: Explain Pricing For Non-Stackable Scenario | `POST /pricing/explain` | [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L37) | engine non-stackable policy ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L163) |
| Request 24: Deactivate Non-Stackable Promotion | `POST /promotions/:promotionId/deactivate` | [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L158) | [internal/service/promotion_service.go](../internal/service/promotion_service.go#L325) |
| Request 25: Create Exclusive Promotion | `POST /promotions` | [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L26) | [internal/service/promotion_service.go](../internal/service/promotion_service.go#L56) |
| Request 26: Validate and Activate Exclusive Promotion | `POST /promotions/:promotionId/validate` แล้ว `POST /promotions/:promotionId/activate` | [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L116), [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L137) | [internal/service/promotion_service.go](../internal/service/promotion_service.go#L284), [internal/service/promotion_service.go](../internal/service/promotion_service.go#L301) |
| Request 27: Explain Pricing For Exclusive Scenario | `POST /pricing/explain` | [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L37) | engine exclusive policy ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L155), [internal/promotion/engine.go](../internal/promotion/engine.go#L202) |
| Request 28: Deactivate Exclusive Promotion | `POST /promotions/:promotionId/deactivate` | [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L158) | [internal/service/promotion_service.go](../internal/service/promotion_service.go#L325) |
| Request 29: Confirm Order With Same Idempotency Key | `POST /orders/confirm` | [internal/handler/order_handler.go](../internal/handler/order_handler.go#L26) | idempotency logic ที่ [internal/service/order_service.go](../internal/service/order_service.go#L71) |

เอกสาร request ตัวเต็มที่อยู่ในภาพอ่านต่อได้ที่ [Docs/API_TESTING_GUIDE.md](./API_TESTING_GUIDE.md#L168-L188)

## Promotion fields ที่ต้องตอบให้ได้

field สำคัญของ promotion model อยู่ที่ [internal/model/models.go](../internal/model/models.go#L69-L89)

จุดที่ควรจำ:

- `scope`: [internal/model/models.go](../internal/model/models.go#L74)
- `priority`: [internal/model/models.go](../internal/model/models.go#L75)
- `stackable`: [internal/model/models.go](../internal/model/models.go#L76)
- `exclusive`: [internal/model/models.go](../internal/model/models.go#L77)
- `stop_processing`: [internal/model/models.go](../internal/model/models.go#L78)
- `conflict_group`: [internal/model/models.go](../internal/model/models.go#L79)
- `status`: [internal/model/models.go](../internal/model/models.go#L80)
- `starts_at`, `ends_at`: [internal/model/models.go](../internal/model/models.go#L81-L82)
- `max_usage`, `max_usage_per_user`: [internal/model/models.go](../internal/model/models.go#L83-L84)
- `version`: [internal/model/models.go](../internal/model/models.go#L85)

## Stackable คืออะไร

`stackable=true` แปลว่า promotion ตัวนี้ “ยอมให้คิดร่วมกับ promo อื่นได้” ถ้าไม่มี policy ตัวอื่นมาขวาง

พูดง่าย ๆ:

- ถ้า promo นี้ยังไม่ชน `exclusive`
- ไม่ชน `conflictGroup`
- และ promo ก่อนหน้าไม่ได้ทำให้ loop หยุด

ก็มีสิทธิ์ถูก apply ต่อจาก promo อื่นได้

logic อยู่ตรงนี้:

- เช็กว่าห้าม stack หรือไม่: [internal/promotion/engine.go](../internal/promotion/engine.go#L163-L169)
- ถ้า apply สำเร็จแล้ว mark state: [internal/promotion/engine.go](../internal/promotion/engine.go#L201-L206)

ใช้ต่างจาก non-stackable ยังไง:

- `stackable=true` เป็นค่า default เชิง policy ว่า “ยอมอยู่ร่วมกับ promo อื่น”
- ไม่ได้แปลว่าจะต้องถูก apply เสมอ เพราะยังอาจโดน skip ด้วย `TARGET_MISMATCH`, `CONDITION_MISMATCH`, `CONFLICT_GROUP_BLOCKED`, หรือโดน promo แบบ `exclusive` ขวางได้

ตัวอย่าง behavior:

- stacked promotions demo: [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L14-L72)

## Non-Stackable คืออะไร

`stackable=false` แปลว่า promo นี้ “ไม่ยอมคิดร่วมกับ promo อื่น”

behavior มี 2 แบบ และเป็นจุดที่พี่มักถาม:

1. ถ้ามี promo อื่น apply ไปก่อนแล้ว แล้วค่อยมาเจอ promo ที่ `stackable=false`
   ระบบจะ skip promo ตัวนี้ด้วยเหตุผล `NON_STACKABLE_CANNOT_STACK`
2. ถ้า promo ที่ `stackable=false` apply สำเร็จก่อน
   promo ถัดไปจะถูก skip ด้วยเหตุผล `NON_STACKABLE_ALREADY_APPLIED`

logic อยู่ตรงนี้:

- กรณี non-stackable มา “ทีหลัง”: [internal/promotion/engine.go](../internal/promotion/engine.go#L167-L169)
- กรณี non-stackable apply ไปแล้ว block ตัวถัดไป: [internal/promotion/engine.go](../internal/promotion/engine.go#L163-L165)
- ตอน apply สำเร็จแล้ว mark state: [internal/promotion/engine.go](../internal/promotion/engine.go#L205-L206)

test ที่พิสูจน์ behavior:

- block promo ถัดไป: [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L141-L183)
- apply ไม่ได้เมื่อมี applied promo มาก่อน: [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L188-L230)

คำตอบสั้นเวลาโดนถาม:

- `stackable=false` ไม่ได้แปลว่า exclusive
- มันแค่ “ห้ามคิดร่วม”
- แต่ไม่ได้บังคับให้ break loop ทันทีแบบ `exclusive`

## Exclusive คืออะไร

`exclusive=true` แปลว่า promo นี้แรงกว่า non-stackable อีกขั้น:

1. ถ้ามี applied promo มาก่อนแล้ว promo นี้จะถูก skip
2. ถ้า promo นี้ apply สำเร็จ จะหยุดการประมวลผลทันที

logic อยู่ตรงนี้:

- ถ้ามี exclusive ถูก apply ไปแล้ว promo หลังจากนั้นจะถูก skip: [internal/promotion/engine.go](../internal/promotion/engine.go#L155-L157)
- ถ้า promo ปัจจุบันเป็น exclusive แต่มี applied promo มาก่อน จะ skip: [internal/promotion/engine.go](../internal/promotion/engine.go#L159-L161)
- ถ้า apply สำเร็จ ให้ mark state: [internal/promotion/engine.go](../internal/promotion/engine.go#L202-L203)
- หลัง apply สำเร็จให้ break loop: [internal/promotion/engine.go](../internal/promotion/engine.go#L215-L217)

ผลลัพธ์ที่ควรจำ:

- `exclusive=true` คือ “apply คนเดียว”
- ถ้าได้ apply แล้ว promo หลังจากนั้นจะไม่ถูกพิจารณาต่อ
- ใน implementation ตอนนี้ promo ที่อยู่หลัง exclusive จะไม่ถูกบันทึกเป็น skipped หาก `break` เกิดขึ้นก่อนถึงมัน

test ที่พิสูจน์ behavior:

- [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L235-L278)

## Non-Exclusive คืออะไร

ไม่มี field ชื่อ `non-exclusive` โดยตรงใน model

ความหมายจริงคือ:

- `exclusive=false`

ผลเชิง behavior:

- promo นี้ไม่มีสิทธิ์บังคับจบ loop ด้วยตัวเอง
- ยังอาจถูก apply ร่วมกับ promo อื่นได้ ถ้า policy อื่นอนุญาต

field อยู่ที่:

- [internal/model/models.go](../internal/model/models.go#L77)

และตอน create / replace promotion ค่านี้จะถูกส่งผ่านตรงจาก request ไป model:

- ตอน create: [internal/service/promotion_service.go](../internal/service/promotion_service.go#L68-L85)
- ตอน replace: [internal/service/promotion_service.go](../internal/service/promotion_service.go#L196-L210)

## Stop Processing คืออะไร

`stopProcessing=true` คล้าย exclusive ตรงที่ “หยุด loop”
แต่ต่างกันที่ไม่ใช่ policy ว่า “คิดร่วมไม่ได้”

สรุปต่างกัน:

- `exclusive=true` = ถ้า apply สำเร็จ ให้จบ และ promo นี้ก็ apply ไม่ได้ถ้ามี applied promo มาก่อน
- `stopProcessing=true` = ถ้า apply สำเร็จ ให้จบ แต่ไม่ได้แปลว่า promo นี้เป็น exclusive

logic อยู่ตรงนี้:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L220-L222)

test ที่พิสูจน์ behavior:

- [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L283-L326)

## Conflict Group คืออะไร

`conflictGroup` คือชื่อกลุ่มเอาไว้กัน promo ชนกันแบบเป็นกลุ่ม

ความหมาย:

- ถ้า promo ตัวหนึ่งในกลุ่มเดียวกัน apply สำเร็จแล้ว
- promo ตัวต่อไปที่มี `conflictGroup` เดียวกันจะถูก skip ด้วย `CONFLICT_GROUP_BLOCKED`

logic อยู่ตรงนี้:

- เช็กว่ากลุ่มถูกใช้ไปแล้วหรือยัง: [internal/promotion/engine.go](../internal/promotion/engine.go#L171-L173)
- mark ว่ากลุ่มนี้ถูกใช้แล้วหลัง apply สำเร็จ: [internal/promotion/engine.go](../internal/promotion/engine.go#L209-L210)

ตัวอย่างใน test stacked promotions:

- มีการตั้ง `conflictGroup`: [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L18), [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L53-L63)

## Explain Pricing กับ Calculate Pricing ต่างกันยังไง

สอง route นี้ใช้ calculator ตัวเดียวกัน แต่ intent ต่างกัน:

- `POST /pricing/explain` เอาไว้ขอดูผลพร้อมเหตุผลเชิงอธิบาย
- `POST /pricing/calculate` เอาไว้ขอราคาสุดท้ายที่จะเอาไปใช้งานต่อ

route และ handler:

- calculate: [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L21-L32)
- explain: [internal/handler/pricing_handler.go](../internal/handler/pricing_handler.go#L37-L48)

service:

- calculate entrypoint: [internal/service/pricing_service.go](../internal/service/pricing_service.go#L52-L54)
- explain entrypoint: [internal/service/pricing_service.go](../internal/service/pricing_service.go#L58-L60)
- shared logic: [internal/service/pricing_service.go](../internal/service/pricing_service.go#L70-L191)

จุดที่ต่างจริงใน code ตอนนี้:

- `Explain()` เรียก `calculate(..., explain=true, persistLog=true)`
- `Calculate()` เรียก `calculate(..., explain=false, persistLog=true)`
- ทั้งคู่ยัง persist calculation log เหมือนกัน

ค่าที่ถูกเก็บเพิ่มใน snapshot:

- `explain`
- `decisionTrace`
- `scopeOrder`

ดูได้ที่ [internal/service/pricing_service.go](../internal/service/pricing_service.go#L196-L219)

## Calculate pricing แบบ stacked promotions ทำงานยังไง

flow จริงของ stacked promotions:

1. service โหลดสินค้าและ promotions active จาก DB
2. engine sort promotions ตาม scope, priority, createdAt, id
3. engine check target
4. engine check conditions
5. engine enforce stackable / exclusive / conflictGroup
6. engine คำนวณ action
7. engine apply discount ลง item หรือ cart

code หลัก:

- load active promotions: [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L74-L84)
- call calculator: [internal/service/pricing_service.go](../internal/service/pricing_service.go#L128-L137)
- sort promotion order: [internal/promotion/engine.go](../internal/promotion/engine.go#L107-L120)
- main engine loop: [internal/promotion/engine.go](../internal/promotion/engine.go#L147-L222)
- apply action: [internal/promotion/engine.go](../internal/promotion/engine.go#L238-L263)

## Create Non-Stackable Promotion คืออะไรในเชิง request

ในระบบนี้ “Create Non-Stackable Promotion” ไม่ใช่ endpoint พิเศษ

มันคือการ `POST /promotions` แล้วส่ง field:

```json
{
  "stackable": false
}
```

จุดรับ request:

- [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L26-L37)

logic create:

- [internal/service/promotion_service.go](../internal/service/promotion_service.go#L56-L85)

จุดที่ field ถูกเขียนลง model:

- [internal/service/promotion_service.go](../internal/service/promotion_service.go#L68-L77)

field นี้ถูกเลือกให้บันทึกลง DB ชัดเจนใน create columns:

- [internal/service/promotion_service.go](../internal/service/promotion_service.go#L88-L113)

## Create Exclusive Promotion คืออะไรในเชิง request

เหมือนกันกับ non-stackable คือไม่ใช่ endpoint พิเศษ

มันคือการ `POST /promotions` แล้วส่ง:

```json
{
  "exclusive": true
}
```

route / handler / service เหมือนหัวข้อบน:

- route: [cmd/server/main.go](../cmd/server/main.go#L79-L87)
- handler create: [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L26-L37)
- service create: [internal/service/promotion_service.go](../internal/service/promotion_service.go#L56-L85)

## Validate Promotion คืออะไร

`validate` คือการเช็กว่า promotion ที่สร้างไว้ “พร้อมใช้งาน” หรือยัง

มันยังไม่ activate

สิ่งที่ validate ตอนนี้ดูหลัก ๆ:

- code ต้องมี
- name ต้องมี
- scope ต้องมี
- date range ต้องถูก
- ต้องมี action
- action / condition ต้องเป็นชนิดที่ระบบรองรับ

handler:

- [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L116-L133)

service:

- [internal/service/promotion_service.go](../internal/service/promotion_service.go#L284-L296)
- validation rules: [internal/service/promotion_service.go](../internal/service/promotion_service.go#L432-L445)

## Activate Promotion คืออะไร

`activate` คือเปลี่ยน status เป็น `ACTIVE`

ก่อน activate ระบบจะ:

1. โหลด promotion ตาม id
2. เช็ก version
3. validate model อีกรอบ
4. เช็กว่ายังไม่หมดอายุ
5. set `status = ACTIVE`
6. เพิ่ม version

handler:

- [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L137-L154)

service:

- [internal/service/promotion_service.go](../internal/service/promotion_service.go#L301-L320)

## Deactivate Promotion คืออะไร

`deactivate` ไม่ได้ลบ promotion

มันคือเปลี่ยน `status` เป็น `INACTIVE`

ผลที่ตามมา:

- promotion จะไม่ถูก `FindActivePromotions()` โหลดเข้ามาคำนวณ

handler:

- [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L158-L175)

service:

- [internal/service/promotion_service.go](../internal/service/promotion_service.go#L325-L341)

repository active filter:

- [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L74-L84)

## Promotion Usage คืออะไร

promotion usage คือ record การใช้ promo หลัง order confirm สำเร็จ

ระบบจะสร้าง usage ต่อ 1 applied promotion

ตอน confirm order:

- loop applied promotions แล้ว insert usage: [internal/service/order_service.go](../internal/service/order_service.go#L152-L160)

โครงสร้าง model:

- [internal/model/models.go](../internal/model/models.go#L126-L135)

query usages:

- handler: [internal/handler/promotion_handler.go](../internal/handler/promotion_handler.go#L179-L196)
- service: [internal/service/promotion_service.go](../internal/service/promotion_service.go#L346-L379)
- repository: [internal/repository/promotion_repository.go](../internal/repository/promotion_repository.go#L172-L195)

ข้อสำคัญ:

- usage จะเกิดตอน `orders/confirm`
- `pricing/calculate` อย่างเดียวจะยังไม่สร้าง usage

## Confirm Order คืออะไร

confirm order คือการเอาผลราคาที่ user ยอมรับแล้วมาสร้าง order จริง

flow:

1. อ่าน `Idempotency-Key`
2. hash request
3. ถ้า key เดิมเคยใช้แล้วและ request hash ตรงกัน ให้คืน order เดิม
4. เช็กว่า calculation log เดิมมีอยู่
5. recalculate ใหม่ผ่าน pricing service
6. เช็กว่า `acceptedFinalTotal` ยังตรง
7. lock promotion usage limit
8. insert order
9. insert order items
10. insert promotion usages

handler:

- [internal/handler/order_handler.go](../internal/handler/order_handler.go#L26-L65)

service:

- main confirm flow: [internal/service/order_service.go](../internal/service/order_service.go#L55-L176)
- usage limit locking: [internal/service/order_service.go](../internal/service/order_service.go#L231-L258)

## Confirm Order With Same Idempotency Key คืออะไร

นี่คือ behavior ของ request 29 ในภาพ

ความหมาย:

- ถ้า client ส่ง `Idempotency-Key` เดิมซ้ำ
- และ request payload เดิมจริง
- ระบบจะไม่สร้าง order ใหม่
- แต่จะคืน order เดิมกลับไป

logic อยู่ตรงนี้:

- เช็ก order จาก idempotency key: [internal/service/order_service.go](../internal/service/order_service.go#L71-L75)
- repository lookup: [internal/repository/order_repository.go](../internal/repository/order_repository.go#L62-L71)

ถ้า key เดิมแต่ request hash ไม่ตรง:

- ระบบจะ fail ด้วย `ErrOrderConfirmationFailed`
- จุดเช็กอยู่ที่ [internal/service/order_service.go](../internal/service/order_service.go#L71-L72)

## Check Order Detail คืออะไร

คือการดึง order ที่ confirm ไปแล้วกลับมาดูพร้อม:

- order summary
- items
- applied promotions
- skipped promotions
- calculation snapshot

handler:

- [internal/handler/order_handler.go](../internal/handler/order_handler.go#L84-L113)

service:

- load order + access check: [internal/service/order_service.go](../internal/service/order_service.go#L208-L216)
- map to detail response: [internal/service/order_service.go](../internal/service/order_service.go#L291-L327)

repository:

- [internal/repository/order_repository.go](../internal/repository/order_repository.go#L30-L39)

## Calculation Log คืออะไร

calculation log คือ audit log ของการคำนวณราคา

มันเก็บ:

- calculation id
- original total
- discount total
- final total
- applied promotions
- skipped promotions
- snapshot ของ request/response

model:

- [internal/model/models.go](../internal/model/models.go#L137-L150)

ตอน persist log:

- [internal/service/pricing_service.go](../internal/service/pricing_service.go#L196-L219)

ข้อควรจำ:

- ทั้ง `pricing/calculate` และ `pricing/explain` บันทึก log
- `pricing/preview` ไม่บันทึก log

## Check Calculation Log List คืออะไร

ดูรายการ calculation logs ทั้งหมดหรือกรองตามเงื่อนไข

handler:

- [internal/handler/calculation_log_handler.go](../internal/handler/calculation_log_handler.go#L26-L37)

service:

- [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L49-L82)

## Check Calculation Log Detail คืออะไร

ดู log รายตัวตาม `calculationId`

สิ่งที่ได้กลับ:

- summary
- applied promotions
- skipped promotions
- calculation snapshot

handler:

- [internal/handler/calculation_log_handler.go](../internal/handler/calculation_log_handler.go#L42-L53)

service:

- [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L87-L107)

## Replay Calculation Log คืออะไร

คือการเอา request เดิมใน snapshot กลับมารันใหม่ แล้วเทียบผลกับของเดิม

implementation ปัจจุบันรองรับ mode เดียว:

- `SNAPSHOT_CONFIG`

flow:

1. โหลด log ตาม `calculationId`
2. decode snapshot
3. เอา request เดิมไปเรียก `pricing.Preview()`
4. เทียบ `originalTotal`, `discountTotal`, `finalTotal`, items, applied promotions, skipped promotions

handler:

- [internal/handler/calculation_log_handler.go](../internal/handler/calculation_log_handler.go#L58-L74)

service:

- replay main flow: [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L112-L146)
- snapshot decode: [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L174-L183)
- compare logic: [internal/service/calculation_log_service.go](../internal/service/calculation_log_service.go#L188-L211)

## เรื่องลำดับการคำนวณที่ควรตอบให้ได้

engine sort promotion ตามลำดับนี้:

1. `scope`
2. `priority`
3. `created_at`
4. `id`

code:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L107-L120)

scope order จริง:

- `ITEM`
- `CART`
- `COUPON`
- `SHIPPING`

นิยามอยู่ที่:

- [internal/promotion/engine.go](../internal/promotion/engine.go#L16-L20)
- [internal/promotion/engine.go](../internal/promotion/engine.go#L226-L231)

## ถ้าพี่ถามว่า policy พวกนี้ไปอยู่ตรงไหนของ code จริง

จุดตอบสั้นที่สุดคือ:

- field อยู่ใน model: [internal/model/models.go](../internal/model/models.go#L69-L89)
- ตอน create/replace ใช้ค่าพวกนี้ใน service: [internal/service/promotion_service.go](../internal/service/promotion_service.go#L56-L85), [internal/service/promotion_service.go](../internal/service/promotion_service.go#L196-L210)
- ตอนคำนวณจริง enforce ใน engine loop: [internal/promotion/engine.go](../internal/promotion/engine.go#L147-L222)
- test behavior อยู่ใน engine tests: [test/unit/promotion/engine_test.go](../test/unit/promotion/engine_test.go#L141-L326)

## ข้อจำกัดที่ควรตอบอย่างตรงไปตรงมา

มีบางอย่างที่ schema หรือชื่อ route ดูเหมือนรองรับเยอะ แต่ runtime ปัจจุบันยังไม่เต็ม:

1. `FREE_SHIPPING` ถูก register แล้ว แต่ handler คืน discount `0`
   อ้างอิง: [internal/promotion/engine.go](../internal/promotion/engine.go#L339-L340)
2. `USER_SEGMENT` และ `FIRST_ORDER` ถูก register แล้ว แต่ปัจจุบันผ่านเสมอ
   อ้างอิงจุด evaluate: [internal/promotion/engine.go](../internal/promotion/engine.go#L269-L289)
3. `logical_operator` และ `operator` ใน condition ยังไม่ได้ถูก evaluate แบบเต็มรูปแบบ
   field อยู่ที่ [internal/model/models.go](../internal/model/models.go#L101-L110)
4. `BUY_X_GET_Y` และ `BUNDLE_DISCOUNT` มีใน model แต่ยังไม่มี runtime handler จริง
   action types อยู่ที่ [internal/model/models.go](../internal/model/models.go#L113-L124)

## เอกสารที่ควรเปิดคู่กัน

- flow การทดสอบ request ตามภาพ: [Docs/API_TESTING_GUIDE.md](./API_TESTING_GUIDE.md)
- ภาพรวม architecture: [Docs/ARCHITECTURE.md](./ARCHITECTURE.md)
- logic walkthrough แบบ end-to-end: [Docs/LOGIC_AND_RESULT_WALKTHROUGH.md](./LOGIC_AND_RESULT_WALKTHROUGH.md)
- gap ของ future strategy: [Docs/FUTURE_STRATEGY_GAP_ANALYSIS.md](./FUTURE_STRATEGY_GAP_ANALYSIS.md)
