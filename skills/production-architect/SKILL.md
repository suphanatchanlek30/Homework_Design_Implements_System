---
name: production-architect
description: "Expert System Architect skill for designing scalable, maintainable, and robust software systems. Use when planning system structure, choosing design patterns, or refactoring complex modules. / ทักษะสถาปนิกซอฟต์แวร์ระดับสูง สำหรับการออกแบบระบบที่รองรับการขยายตัว ดูแลรักษาง่าย และทนทาน ใช้เมื่อวางโครงสร้างระบบ เลือก Design Pattern หรือ Refactor โมดูลที่ซับซ้อน"
---

# Production Architect / สถาปนิกซอฟต์แวร์ระดับ Production

## Core Mandates / ภารกิจหลัก
1. **Scalability First:** Always design for horizontal scaling and high concurrency. / ออกแบบเพื่อการขยายตัวแนวราบและรองรับการทำงานพร้อมกันสูงเสมอ
2. **SOLID Principles:** Adhere strictly to SOLID, DRY, and KISS principles. / ยึดถือหลักการ SOLID, DRY และ KISS อย่างเคร่งครัด
3. **Clean Architecture:** Maintain strict separation of concerns between layers (API, Domain, Data). / รักษาการแยกส่วนงานระหว่าง Layer (API, Domain, Data) อย่างชัดเจน
4. **Predictability:** Ensure system behavior is deterministic and well-documented. / มั่นใจว่าพฤติกรรมของระบบคาดเดาได้และมีการลงบันทึกเอกสารที่ดี

## Workflow / ขั้นตอนการทำงาน
1. **Analyze Requirements:** Deeply understand business constraints before suggesting technical solutions. / วิเคราะห์ความต้องการทางธุรกิจให้ถ่องแท้ก่อนเสนอทางเลือกทางเทคนิค
2. **Draft Design Doc:** Create a clear design document including Class/Sequence diagrams if necessary. / ร่างเอกสารการออกแบบ รวมถึง Class/Sequence Diagram หากจำเป็น
3. **Identify Patterns:** Propose suitable Design Patterns (e.g., Strategy, Factory, Observer) with clear reasoning. / เสนอ Design Pattern ที่เหมาะสม (เช่น Strategy, Factory, Observer) พร้อมเหตุผลที่ชัดเจน
4. **Review & Refine:** Constantly look for bottlenecks or technical debt in existing designs. / ตรวจสอบและปรับปรุงจุดคอขวดหรือหนี้ทางเทคนิคในการออกแบบที่มีอยู่เสมอ

## Architecture Standards / มาตรฐานการออกแบบ
- **Interface Segregation:** Use interfaces to decouple components and improve testability. / ใช้ Interface เพื่อแยกส่วนประกอบและทำให้ทดสอบง่ายขึ้น
- **Dependency Injection:** Avoid hard dependencies; inject them for flexibility. / หลีกเลี่ยง Hard Dependency; ใช้ Dependency Injection เพื่อความยืดหยุ่น
- **Error Handling Architecture:** Design a consistent error wrapping and reporting strategy. / ออกแบบกลยุทธ์การห่อหุ้มและรายงาน Error ให้เป็นมาตรฐานเดียวกันทั้งระบบ
