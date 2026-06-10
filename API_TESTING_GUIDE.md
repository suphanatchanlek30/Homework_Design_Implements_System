# Full API Testing Guide (Postman)

คู่มือฉบับสมบูรณ์สำหรับทดสอบทุก Endpoint ในระบบ (Health, Category, และ Product)

## Base Configuration
- **Base URL:** `http://localhost:3000/api/v1`
- **Headers:** 
  - `Content-Type: application/json`
  - `Idempotency-Key: <unique-uuid>` (Optional - สำหรับ POST)

---

## 1. System Health APIs
ใช้ตรวจสอบว่า Server และ Database พร้อมใช้งานหรือไม่

### 🟢 Check Health (Live Check)
- **Method:** `GET`
- **Path:** `/healthz`
- **Expected:** `200 OK` + `{"status": "UP", ...}`

### 🟢 Check Readiness (Dependency Check)
- **Method:** `GET`
- **Path:** `/readyz`
- **Expected:** `200 OK` + `{"status": "READY", "database": "CONNECTED"}`

---

## 2. Category APIs (จัดการหมวดหมู่)

### 🟢 Create Category (Parent)
- **Method:** `POST`
- **Path:** `/categories`
- **Body:** `{ "name": "Electronics", "status": "ACTIVE", "parentId": null }`
- **Expected:** `201 Created`

### 🟢 Create Category (Child)
- **Method:** `POST`
- **Path:** `/categories`
- **Body:** `{ "name": "Mobile Phones", "status": "ACTIVE", "parentId": 1 }`
- **Expected:** `201 Created`

### 🔍 List Categories (Pagination & Filters)
- **Method:** `GET`
- **Path:** `/categories?status=ACTIVE&page=1&limit=10&keyword=Mobile`
- **Expected:** `200 OK` + List of categories

### 🔍 Get Category by ID
- **Method:** `GET`
- **Path:** `/categories/1`
- **Expected:** `200 OK` + Category details

### 🟡 Update Category (Partial Update)
- **Method:** `PATCH`
- **Path:** `/categories/1`
- **Body:** `{ "name": "Gadgets & Electronics" }`
- **Expected:** `200 OK`

### 🔴 Edge Case: Circular Hierarchy (Update Parent to Child)
- **Method:** `PATCH`
- **Path:** `/categories/1`
- **Body:** `{ "parentId": 2 }` (สมมติ 2 เป็นลูกของ 1)
- **Expected:** `422 Unprocessable Entity`

---

## 3. Product APIs (จัดการสินค้า)

### 🟢 Create Product
- **Method:** `POST`
- **Path:** `/products`
- **Body:**
  ```json
  {
    "sku": "IPHONE-15-PRO",
    "name": "iPhone 15 Pro",
    "categoryId": 2,
    "priceAmount": 41900,
    "currency": "THB",
    "status": "ACTIVE"
  }
  ```
- **Expected:** `201 Created`

### 🔍 List Products (Full Filtering)
- **Method:** `GET`
- **Path:** `/products?categoryId=2&status=ACTIVE&sku=IPHONE&page=1&limit=20&sort=price_amount desc`
- **Expected:** `200 OK`

### 🔍 Get Product by ID
- **Method:** `GET`
- **Path:** `/products/1`
- **Expected:** `200 OK`

### 🟡 Update Product Price/Status
- **Method:** `PATCH`
- **Path:** `/products/1`
- **Body:** `{ "priceAmount": 39900, "status": "INACTIVE" }`
- **Expected:** `200 OK`

### 🔴 Error Case: Duplicate SKU
- **Method:** `POST`
- **Path:** `/products`
- **Body:** ใช้ SKU เดิมที่เคยสร้างไปแล้ว (`IPHONE-15-PRO`)
- **Expected:** `409 Conflict`

### 🔴 Error Case: Invalid Category
- **Method:** `POST`
- **Path:** `/products`
- **Body:** `{ "sku": "NEW", "name": "Item", "categoryId": 999, ... }`
- **Expected:** `404 Not Found`

---

## สรุป Status Codes ที่ควรได้รับ
- `200 OK`: สำเร็จ (GET/PATCH)
- `201 Created`: สร้างสำเร็จ (POST)
- `400 Bad Request`: ข้อมูลที่ส่งมาไม่ถูกต้อง (เช่น ชื่อว่าง)
- `404 Not Found`: ไม่พบ Resource หรือ Parent/Category
- `409 Conflict`: ข้อมูลซ้ำ (SKU)
- `422 Unprocessable Entity`: ข้อมูลถูกต้องตามโครงสร้าง แต่ขัดต่อ Business Logic (Circular Loop / ราคาติดลบ)
