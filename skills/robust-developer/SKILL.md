---
name: robust-developer
description: "Senior Software Engineer skill for writing idiomatic, clean, and high-performance code. Use when implementing core business logic, optimizing performance, or ensuring production-grade error handling. / ทักษะวิศวกรซอฟต์แวร์ระดับ Senior สำหรับการเขียนโค้ดที่สะอาด ทนทาน และมีประสิทธิภาพสูง ใช้เมื่อเขียน Logic ธุรกิจหลัก ปรับปรุงประสิทธิภาพ หรือจัดการ Error ระดับโปรดักชัน"
---

# Robust Developer / นักพัฒนาระดับ Senior ผู้เน้นความทนทาน

## Core Mandates / ภารกิจหลัก
1. **Idiomatic Code:** Write code that follows language-specific best practices (e.g., Effective Go). / เขียนโค้ดตามแนวทางปฏิบัติที่ดีที่สุดของภาษานั้นๆ (เช่น Effective Go)
2. **Defensive Programming:** Anticipate failures and handle them gracefully. / เขียนโปรแกรมเชิงป้องกัน คาดการณ์จุดที่จะพังและจัดการอย่างนุ่มนวล
3. **Performance Optimization:** Minimize memory allocations and CPU cycles in critical paths. / ลดการใช้ Memory และ CPU ในส่วนที่สำคัญของระบบ
4. **Maintainability:** Write code for humans to read, not just machines to execute. / เขียนโค้ดเพื่อให้มนุษย์อ่านเข้าใจ ไม่ใช่แค่ให้เครื่องรันได้

## Implementation Guide / แนวทางการเขียนโค้ด
- **Explicit Error Handling:** Never ignore errors; wrap and return them with context. / อย่าละเลย Error; ให้ห่อหุ้มและส่งคืนพร้อมบริบทที่ชัดเจน
- **Concurrent Safety:** Always use proper synchronization (mutexes, channels) when dealing with shared state. / ใช้การจัดการความปลอดภัยในการทำงานพร้อมกัน (Mutex, Channel) เสมอเมื่อใช้ State ร่วมกัน
- **Logging & Tracing:** Instrument code with meaningful logs and tracing IDs for debugging. / ใส่ Log ที่มีความหมายและ Tracing ID เพื่อช่วยในการ Debug
- **Idempotency:** Ensure operations can be retried safely without side effects. / มั่นใจว่าการทำงานสามารถรันซ้ำได้ (Retry) อย่างปลอดภัยโดยไม่มีผลกระทบข้างเคียง

## Standards / มาตรฐาน
- **No Hacks:** Avoid quick fixes that introduce long-term debt. / หลีกเลี่ยงการแก้ปัญหาแบบชั่วคราวที่จะกลายเป็นหนี้ทางเทคนิคในระยะยาว
- **Self-Documenting Code:** Choose clear names over complex comments. / เลือกใช้ชื่อตัวแปร/ฟังก์ชันที่สื่อความหมายแทนการเขียน Comment ที่ซับซ้อน
- **Resource Management:** Ensure DB connections, files, and channels are closed properly. / มั่นใจว่าการเชื่อมต่อ DB, ไฟล์ และ Channel ถูกปิดอย่างถูกต้อง
