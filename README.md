# Promotion Engine Homework

ระบบนี้เป็นโปรเจกต์ Go/Fiber สำหรับ promotion engine ที่คำนวณราคาสุทธิของออเดอร์ มี MySQL เป็นฐานข้อมูลหลัก และรองรับการ bootstrap schema/seed ทั้งผ่าน Docker และคำสั่ง Go แยกต่างหาก

## โครงสร้างหลัก

- `cmd/server/` จุดเริ่มต้นของแอป
- `database/schema.sql` schema หลักของ MySQL 8.0+
- `database/seed.sql` ข้อมูลตัวอย่าง Product 1 / Product 2 และ promotions
- `internal/` โครงสร้าง application layer สำหรับต่อยอด business logic
- `cmd/seed/` คำสั่งสำหรับ seed schema + data เข้า MySQL โดยตรง
- `docker-compose.yml` สตาร์ต MySQL + แอปพร้อม seed
- `Dockerfile` build แอป Go เป็น container

## สิ่งที่ต้องมี

- Docker Desktop หรือ Docker Engine + Docker Compose
- Go 1.24.5 ถ้าจะรันแบบ local โดยไม่ใช้ Docker

## เริ่มใช้งานแบบ Docker

1. สร้างไฟล์ `.env` จาก `.env.example` แล้วปรับค่าตามต้องการ
2. สตาร์ตระบบด้วยคำสั่ง:

```bash
docker compose up --build
```

3. เปิดใช้งานแอปที่ `http://localhost:${APP_PORT}` เช่น `http://localhost:3000`
4. MySQL จะพร้อมใช้งานที่ `localhost:${MYSQL_HOST_PORT}` เช่น `localhost:3307`

Health check ของแอปคือ `http://localhost:${APP_PORT}/health`

## การ seed ฐานข้อมูล

Docker Compose จะ mount โฟลเดอร์ `database/` เข้าไปที่ `/docker-entrypoint-initdb.d` ของ MySQL
และ MySQL official image จะรันไฟล์ `.sql` ตามลำดับอัตโนมัติเมื่อ volume ของฐานข้อมูลยังว่างอยู่

ลำดับการ init คือ:

1. `database/schema.sql`
2. `database/seed.sql`

ถ้าต้องการล้างฐานข้อมูลแล้ว seed ใหม่ ให้รัน:

```bash
docker compose down -v
docker compose up --build
```

ถ้าต้องการ seed ข้อมูลผ่าน Go command โดยตรง ให้รัน:

```bash
go run ./cmd/seed
```

คำสั่งนี้จะรันเฉพาะ `database/seed.sql` ลงในฐานข้อมูลที่มี schema อยู่แล้ว

ถ้าต้องการสร้าง schema ใหม่และ seed ไปพร้อมกัน ให้รัน:

```bash
go run ./cmd/seed --schema database/schema.sql --seed database/seed.sql
```

คำสั่งนี้จะเชื่อมต่อ MySQL จากค่าที่อยู่ใน `.env` แล้วรัน schema + seed ให้ทันที

## ตรวจสอบข้อมูลใน MySQL

```bash
docker compose exec mysql mysql -u${MYSQL_USER} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE}
```

ตัวอย่าง query:

```sql
SELECT * FROM products;
SELECT * FROM promotions;
SELECT * FROM promotion_actions;
```

## รันแบบ local

หากต้องการรันเฉพาะแอป Go แบบไม่ใช้ Docker:

```bash
go run ./cmd/server
```

ตัวแปรสำคัญจะถูกอ่านจาก `.env` โดยตรง ถ้าไฟล์ `.env` อยู่ที่ root ของโปรเจกต์:

- `APP_PORT` สำหรับพอร์ตของ Go service
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` สำหรับ MySQL connection
- `MYSQL_ROOT_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_HOST_PORT` สำหรับ Docker Compose
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` สำหรับคำสั่ง Go ที่รันบนเครื่อง local

หมายเหตุ: Docker Compose จะ override `DB_HOST` และ `DB_PORT` ภายใน container ของแอปให้ชี้ไปที่ MySQL service (`mysql:3306`) โดยตรง

คำสั่ง Go `cmd/server` และ `cmd/seed` จะโหลด `.env` อัตโนมัติใน environment local

ถ้าจะรัน local ให้แน่ใจว่า MySQL พร้อมใช้งานและค่าฐานข้อมูลใน `.env` ตรงกับเครื่องของคุณ

## API ที่มีตอนนี้

- `GET /` ตรวจสอบว่าแอปตอบสนอง
- `GET /health` ตรวจสอบการเชื่อมต่อ MySQL จริง

## ลำดับการใช้งานที่แนะนำ

1. ตั้งค่า `.env`
2. รัน `docker compose up --build`
3. ตรวจ `GET /health`
4. ถ้าต้องการ seed ใหม่ แยกใช้ `go run ./cmd/seed`

## หมายเหตุ

- Schema หลักและ seed data อ้างอิงจากไฟล์ใน `database/`
- หากปรับ schema หรือ seed ให้ล้าง volume ก่อนเพื่อให้ MySQL init ใหม่
- เอกสารออกแบบเพิ่มเติมอยู่ใน `Docs/`