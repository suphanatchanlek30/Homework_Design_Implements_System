# Future Strategy Gap Analysis

เอกสารนี้อธิบายเฉพาะกล่อง `Future Strategy` ในภาพ architecture ว่าตอนนี้โค้ดปัจจุบันรองรับอะไรแล้ว, อะไรยังไม่ตรง, และช่องว่างที่เหลือคืออะไร

## จุดประสงค์ของเอกสารนี้

ใช้ตอบคำถาม 4 เรื่อง:

1. ภาพ architecture คาดหวังอะไรจาก `Future Strategy`
2. โค้ดปัจจุบันรองรับการขยาย strategy ได้แค่ไหน
3. อะไรคือส่วนที่ “ตรงเชิงโครงสร้าง” แต่ “ยังไม่ตรงเชิงพฤติกรรม”
4. ถ้าจะทำให้ตรงกับภาพมากขึ้น ต้องเติมอะไรต่อ

## สรุปสั้น

สรุปแบบตรงไปตรงมา:

- `Future Strategy` ตรงในเชิง architecture
- `Future Strategy` ยังไม่ตรงเต็มในเชิง runtime behavior

เหตุผลหลักคือ:

- ระบบมี `registry-based design` ที่เปิดให้เสียบ action/condition ใหม่ได้
- แต่ action/condition บางชนิดยังเป็น placeholder หรือยังไม่ถูก register ให้ทำงานจริง

## ภาพคาดหวังของ Future Strategy

จากภาพ `Future Strategy` หมายถึงความสามารถแบบนี้:

- เพิ่ม promotion strategy ใหม่โดยไม่รื้อ core pricing flow
- ให้ strategy ใหม่เข้าร่วม flow เดิมได้ เช่น ผ่าน target, condition, ordering, stacking, calculation
- เมื่อ strategy ใหม่ถูกเพิ่มแล้ว ระบบควร apply, skip, หรือ error ได้อย่าง deterministic
- strategy ใหม่ควรอธิบายได้ในผลลัพธ์เดียวกับ strategy เดิม ไม่ต้องมี flow พิเศษแยก

ถ้าแปลเป็นภาษาวิศวกรรม:

- core engine ต้องไม่ hardcode promo รายตัว
- ต้องมี extension point สำหรับ action/condition
- schema ต้องเก็บกติกาใหม่ได้
- runtime ต้องมี handler ที่ execute ได้จริง

แนวคิดนี้สอดคล้องกับ README ของโปรเจกต์ที่อธิบายว่า:

- promotion ถูกเก็บเป็น data ไม่ hardcode
- ใช้ `registry` map `actionType` และ `conditionType` ไปยัง handler
- ต้องขยาย action/condition ใหม่ได้โดยไม่รื้อ flow หลัก

อ้างอิง:

- `README.md` แนวคิดระบบที่ [README.md](../README.md#L15)

## โค้ดปัจจุบันที่เกี่ยวข้อง

ส่วนของระบบที่เป็นฐานรองรับ `Future Strategy` ตอนนี้มี 4 ชั้น:

1. `Schema / Model`
2. `Promotion validation`
3. `Registry`
4. `Engine runtime`

### 1. Schema / Model

model ของ action และ condition ถูกออกแบบให้รองรับหลายชนิด:

- `PromotionCondition.ConditionType` รองรับ:
  - `PRODUCT_ID`
  - `CATEGORY_ID`
  - `MIN_ORDER_AMOUNT`
  - `MAX_ORDER_AMOUNT`
  - `COUPON_CODE`
  - `USER_SEGMENT`
  - `FIRST_ORDER`
  - `PAYMENT_METHOD`
  - `DATE_RANGE`
- `PromotionAction.ActionType` รองรับ:
  - `PERCENTAGE_DISCOUNT`
  - `FIXED_AMOUNT_DISCOUNT`
  - `CART_PERCENTAGE_DISCOUNT`
  - `CART_FIXED_AMOUNT_DISCOUNT`
  - `FREE_SHIPPING`
  - `BUY_X_GET_Y`
  - `BUNDLE_DISCOUNT`

อ้างอิง:

- action/condition enum ที่ [internal/model/models.go](../internal/model/models.go#L101)

### 2. Promotion validation

ก่อน promotion จะถูกใช้ ระบบ validate ว่า action/condition ที่ส่งเข้ามา “รองรับหรือไม่” ผ่าน `isSupportedAction` และ `isSupportedCondition`

จุดสำคัญคือ:

- create/replace/validate จะไม่ยอม actionType ที่ไม่อยู่ใน supported list
- แปลว่า extension point มี gate อยู่ชัดเจน

อ้างอิง:

- validation config ที่ [internal/service/promotion_service.go](../internal/service/promotion_service.go#L402)

### 3. Registry

`Registry` เป็น extension point หลักของระบบ:

- map `actionType -> ActionHandler`
- map `conditionType -> ConditionHandler`
- runtime เรียกผ่าน lookup ไม่ hardcode promo รายตัว

โครงสร้างหลัก:

- `RegisterAction` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L43)
- `RegisterCondition` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L49)
- supported actions ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L69)
- supported conditions ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L81)

### 4. Engine runtime

runtime ใช้ registry execute strategy จริง:

- engine วน actions ของ promotion
- lookup handler จาก `registry.Action(action.ActionType)`
- ถ้าไม่มี handler จะ error `ACTION_STRATEGY_NOT_SUPPORTED`
- condition ก็ทำเหมือนกันผ่าน `registry.Condition(condition.ConditionType)`

อ้างอิง:

- action execution path ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L238)
- condition execution path ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L269)

## ตารางเทียบ Future Strategy โดยเฉพาะ

| ภาพคาดหวัง | โค้ดปัจจุบัน | สถานะ | ช่องว่างที่เหลือ |
|---|---|---|---|
| เพิ่ม strategy ใหม่ได้โดยไม่แก้ flow หลัก | มี `Registry`, `ActionHandler`, `ConditionHandler` และ engine dispatch ตาม type | ตรง | ยังต้องเพิ่ม registration และ handler ให้ strategy ใหม่แต่ละตัว |
| ระบบเก็บ action/condition ใหม่เป็น data | model มี `ActionType`, `ConditionType`, `ValueJSON` และ enum หลายแบบ | ตรง | บาง enum อยู่ใน schema แล้วแต่ runtime ยังไม่รองรับ |
| strategy ใหม่ควรถูก validate ได้ก่อนใช้งาน | promotion service validate ผ่าน supported lists | ตรงบางส่วน | supported list ตอนนี้ไม่รวม action ใหม่บางตัวที่มีใน model เช่น `BUY_X_GET_Y`, `BUNDLE_DISCOUNT` |
| strategy ใหม่ควรถูกรันจริงใน engine | engine lookup handler จาก registry แล้ว execute จริง | ตรงบางส่วน | ถ้าไม่มี handler จะ fail ทันที และบาง handler ยังเป็น placeholder |
| future action แบบ shipping ควรคำนวณผลจริงได้ | `FREE_SHIPPING` ถูก register แล้ว | ตรงบางส่วน | handler ยังคืน `0` ตลอด จึงยังไม่เกิด business effect จริง |
| future condition แบบ user-based ควรมี logic จริง | `USER_SEGMENT` และ `FIRST_ORDER` ถูก register แล้ว | ตรงบางส่วน | ตอนนี้ทั้งคู่ใช้ passthrough handler ที่คืน `true` เสมอ |
| strategy ใหม่ควรแสดงผลใน flow เดียวกับ applied/skipped promotions | engine รวมผลผ่าน `AppliedPromotions` / `SkippedPromotions` เดียวกัน | ตรง | strategy ใหม่ต้องมี handler ที่ให้ discount/reason ที่ meaningful |
| เพิ่ม strategy ใหม่แล้วไม่ควรต้องสร้าง endpoint ใหม่ | promotion flow ใช้ API เดิมสร้าง/validate/activate ได้ | ตรง | ยังต้องขยาย validation และ runtime ให้รองรับชนิดใหม่ครบ |

## แยกตามชนิด strategy ที่ภาพน่าจะคาดหวัง

### A. Percentage และ Fixed Amount

สองกลุ่มนี้ถือว่า “เป็น baseline strategy ที่ตรงแล้ว”

โค้ดที่มีจริง:

- `percentageActionHandler` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L296)
- `fixedAmountActionHandler` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L316)
- register ใน registry ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L97)

สรุป:

- ตรงทั้งเชิง architecture
- ตรงทั้งเชิง runtime

### B. FREE_SHIPPING

นี่คือกรณีที่ชัดที่สุดของคำว่า “ตรงบางส่วน”

#### ภาพคาดหวัง

- ระบบมี strategy ใหม่ประเภท shipping-based
- เมื่อ action นี้ถูกเรียก ควรเกิดผลกับค่าจัดส่งหรือยอดสุดท้าย

#### โค้ดปัจจุบัน

- `FREE_SHIPPING` อยู่ใน `SupportedActionTypes()` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L69)
- `FREE_SHIPPING` ถูก register ใน default actions ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L97)
- handler จริงคือ `freeShippingActionHandler` ที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L339)

#### สถานะ

ตรงบางส่วน

#### ช่องว่างที่เหลือ

- handler คืน `0` เสมอ
- ระบบยังไม่มี shipping cost base แยกจากสินค้า
- จึงยังไม่สามารถพิสูจน์ business effect ของ free shipping ได้จริง

#### ข้อสรุป

`FREE_SHIPPING` ตอนนี้เป็น “placeholder strategy” มากกว่า “working future strategy”

### C. BUY_X_GET_Y

นี่คือกรณี “มีใน schema แต่ยังไม่เข้า runtime”

#### ภาพคาดหวัง

- ระบบสามารถรองรับ strategy ซับซ้อนขึ้น เช่น ซื้อ X แถม Y

#### โค้ดปัจจุบัน

- `BUY_X_GET_Y` อยู่ใน model enum ที่ [internal/model/models.go](../internal/model/models.go#L113)
- แต่ `SupportedActionTypes()` ไม่มี `BUY_X_GET_Y` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L69)
- และไม่มี registration ใน default actions ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L97)

#### สถานะ

ไม่ตรง

#### ช่องว่างที่เหลือ

- ต้องเพิ่ม `BUY_X_GET_Y` เข้า supported actions
- ต้อง register handler ใหม่
- ต้องออกแบบ `ValueJSON` schema ให้ชัด เช่น:
  - buy product id
  - buy quantity
  - get product id
  - get quantity
  - discount rule ของ Y
- ต้องทำ item-matching logic ให้ละเอียดกว่าปัจจุบัน

#### ข้อสรุป

ภาพอาจคาดหวังว่า future strategy แบบนี้เสียบได้ง่าย แต่ ณ ตอนนี้ยังเป็นเพียง “ชื่อที่ schema รู้จัก” ยังไม่ใช่ “strategy ที่ระบบใช้ได้”

### D. BUNDLE_DISCOUNT

สถานะใกล้เคียงกับ `BUY_X_GET_Y`

#### ภาพคาดหวัง

- ระบบควรเพิ่ม strategy แบบ bundle ได้ในอนาคต

#### โค้ดปัจจุบัน

- `BUNDLE_DISCOUNT` อยู่ใน model enum ที่ [internal/model/models.go](../internal/model/models.go#L113)
- ไม่มีใน `SupportedActionTypes()` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L69)
- ไม่มี registration ใน registry ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L97)

#### สถานะ

ไม่ตรง

#### ช่องว่างที่เหลือ

- ต้องนิยาม bundle payload ใน `ValueJSON`
- ต้องทำ matching หลายสินค้าใน order เดียว
- ต้องออกแบบวิธีเฉลี่ย discount ลง item แต่ละตัว
- ต้องทดสอบ interaction กับ stackable/exclusive/conflict group

### E. USER_SEGMENT

นี่คือกรณี “register แล้ว แต่ logic ยังไม่จริง”

#### ภาพคาดหวัง

- condition แบบ user-based ต้องบังคับใช้ตามข้อมูล user จริง

#### โค้ดปัจจุบัน

- `USER_SEGMENT` อยู่ใน `SupportedConditionTypes()` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L81)
- ถูก register ใน default conditions ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L107)
- แต่ผูกกับ `passthroughConditionHandler` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L115)
- handler จริงคืน `true` ตลอดที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L427)

#### สถานะ

ตรงบางส่วน

#### ช่องว่างที่เหลือ

- ไม่มี user profile/segment source
- ไม่มี lookup user segment จาก request หรือ service ภายนอก
- ไม่มีการ parse `ValueJSON` เพื่อตรวจ segment จริง

### F. FIRST_ORDER

ใกล้เคียง `USER_SEGMENT`

#### ภาพคาดหวัง

- ระบบควรเช็กว่าผู้ใช้เคยมี order มาก่อนหรือไม่

#### โค้ดปัจจุบัน

- `FIRST_ORDER` อยู่ใน `SupportedConditionTypes()` ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L81)
- ถูก register เป็น passthrough ที่ [internal/promotion/registry.go](../internal/promotion/registry.go#L116)
- runtime handler คืน `true` ตลอดที่ [internal/promotion/engine.go](../internal/promotion/engine.go#L427)

#### สถานะ

ตรงบางส่วน

#### ช่องว่างที่เหลือ

- ไม่มี query order history ใน engine
- ไม่มี dependency ให้ condition ไปถาม order repository
- ไม่มี policy ชัดว่าดู first order จาก user ไหนถ้า `userId` เป็น `nil`

## แยกความต่างระหว่าง “รองรับได้” กับ “ทำงานจริง”

จุดที่มักทำให้เข้าใจคลาดเคลื่อนคือระบบนี้มี 3 ระดับของคำว่า “รองรับ”

### ระดับ 1: รองรับใน schema

หมายถึง:

- model ยอมให้เก็บค่าชนิดนั้นใน DB ได้

ตัวอย่าง:

- `BUY_X_GET_Y`
- `BUNDLE_DISCOUNT`

แต่ยังไม่แปลว่าจะ validate ผ่านหรือ execute ได้

### ระดับ 2: รองรับใน validation / registry

หมายถึง:

- create/validate ผ่าน
- registry รู้จักชนิดนั้น

ตัวอย่าง:

- `FREE_SHIPPING`
- `USER_SEGMENT`
- `FIRST_ORDER`

แต่ยังไม่แปลว่าจะมี business effect ถูกต้อง

### ระดับ 3: รองรับใน runtime behavior จริง

หมายถึง:

- strategy มี handler ที่คำนวณหรือเช็กเงื่อนไขได้จริง
- ส่งผลต่อ applied/skipped promotions อย่างมีความหมาย

ตัวอย่างที่เข้าใกล้ระดับนี้จริงตอนนี้:

- `PERCENTAGE_DISCOUNT`
- `FIXED_AMOUNT_DISCOUNT`
- `CART_PERCENTAGE_DISCOUNT`
- `CART_FIXED_AMOUNT_DISCOUNT`

## สรุปสถานะรายชนิด

| ชนิด | อยู่ใน model | อยู่ใน supported list | register ใน registry | มี logic runtime จริง | สถานะรวม |
|---|---|---|---|---|---|
| `PERCENTAGE_DISCOUNT` | มี | มี | มี | มี | ตรง |
| `FIXED_AMOUNT_DISCOUNT` | มี | มี | มี | มี | ตรง |
| `CART_PERCENTAGE_DISCOUNT` | มี | มี | มี | มี | ตรง |
| `CART_FIXED_AMOUNT_DISCOUNT` | มี | มี | มี | มี | ตรง |
| `FREE_SHIPPING` | มี | มี | มี | ไม่มีผลจริง | ตรงบางส่วน |
| `BUY_X_GET_Y` | มี | ไม่มี | ไม่มี | ไม่มี | ไม่ตรง |
| `BUNDLE_DISCOUNT` | มี | ไม่มี | ไม่มี | ไม่มี | ไม่ตรง |
| `USER_SEGMENT` | มี | มี | มี | เป็น passthrough | ตรงบางส่วน |
| `FIRST_ORDER` | มี | มี | มี | เป็น passthrough | ตรงบางส่วน |

## ทำไมถึงบอกว่า Future Strategy “ยังไม่ตรงเต็ม”

เพราะภาพไม่ได้สื่อแค่ว่า “มีช่องไว้เพิ่ม” แต่สื่อว่า:

- ระบบพร้อมรองรับ strategy ใหม่ใน flow เดิม
- strategy ใหม่เหล่านั้นควรมีพฤติกรรมจริง
- การขยายควรเป็น first-class behavior ไม่ใช่แค่ placeholder

โค้ดปัจจุบันตอบโจทย์ข้อแรกดีมาก แต่ยังตอบข้อสองและสามได้ไม่ครบ

## ถ้าจะทำให้ตรงกับภาพมากขึ้น ต้องเติมอะไร

### ระยะสั้น

- ทำ `FREE_SHIPPING` ให้คำนวณ shipping base จริง
- แทน `passthroughConditionHandler` ของ `USER_SEGMENT` / `FIRST_ORDER` ด้วย logic จริง
- เพิ่ม tests สำหรับ supported future strategies ที่ทำงานแล้ว

### ระยะกลาง

- เพิ่ม `BUY_X_GET_Y` เข้า supported list
- register handler
- นิยาม `ValueJSON` schema สำหรับ rule ซับซ้อน
- เพิ่ม item grouping/matching logic

### ระยะยาว

- แยก action/condition handlers ออกจาก `engine.go` เป็นแพ็กเกจ strategy ที่ใช้จริง
- รองรับ composable strategies ที่ซับซ้อนมากขึ้น
- เพิ่ม trace ที่อธิบายว่า future strategy ตัดสินใจอย่างไรใน explain/log

## ประโยคสรุปที่ใช้ในเอกสารหรือพรีเซนต์ได้เลย

ถ้าต้องการอธิบายแบบกระชับ:

> ระบบรองรับ Future Strategy ในระดับ architecture แล้วผ่าน schema + registry-based extension points แต่ runtime implementation ของบาง strategy ยังเป็น placeholder หรือยังไม่ถูก register ให้ทำงานจริง

ถ้าต้องการอธิบายแบบตรงขึ้น:

> ปัจจุบัน Future Strategy ตรงในเชิงการออกแบบ แต่ยังไม่ตรงเต็มในเชิงพฤติกรรม เพราะ action/condition บางชนิดยังไม่ส่งผลต่อการคำนวณจริง

## อ้างอิงหลัก

- แนวคิด data-driven และ registry ใน [README.md](../README.md#L15)
- model action/condition enums ใน [internal/model/models.go](../internal/model/models.go#L101)
- registry definition และ supported lists ใน [internal/promotion/registry.go](../internal/promotion/registry.go#L24)
- default registry mappings ใน [internal/promotion/registry.go](../internal/promotion/registry.go#L97)
- engine action execution path ใน [internal/promotion/engine.go](../internal/promotion/engine.go#L238)
- engine condition execution path ใน [internal/promotion/engine.go](../internal/promotion/engine.go#L269)
- placeholder free shipping handler ใน [internal/promotion/engine.go](../internal/promotion/engine.go#L339)
- passthrough condition handler ใน [internal/promotion/engine.go](../internal/promotion/engine.go#L427)
- promotion validation gate ใน [internal/service/promotion_service.go](../internal/service/promotion_service.go#L402)
