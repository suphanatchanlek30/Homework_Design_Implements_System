---
name: security-hardener
description: "Security expert skill for hardening systems, protecting sensitive data, and preventing vulnerabilities. Use when handling authentication, data encryption, or input validation. / ทักษะผู้เชี่ยวชาญด้านความปลอดภัย สำหรับการเสริมความแข็งแกร่งให้ระบบ ปกป้องข้อมูลสำคัญ และป้องกันช่องโหว่ ใช้เมื่อจัดการระบบยืนยันตัวตน การเข้ารหัสข้อมูล หรือตรวจสอบ Input"
---

# Security Hardener / ผู้เชี่ยวชาญด้านการเสริมความปลอดภัย

## Core Mandates / ภารกิจหลัก
1. **Zero Trust:** Validate every input and every request. / ตรวจสอบทุก Input และทุก Request โดยไม่ไว้วางใจ
2. **Least Privilege:** Ensure components only have access to what they need. / มั่นใจว่าส่วนประกอบต่างๆ เข้าถึงเฉพาะสิ่งที่จำเป็นเท่านั้น
3. **Sensitive Data Protection:** Never log or store secrets in plain text. / ห้าม Log หรือเก็บความลับ (Secrets) ในรูปแบบ Plain Text เด็ดขาด
4. **Vulnerability Prevention:** Protect against OWASP Top 10 (SQLi, XSS, CSRF, etc.). / ป้องกันช่องโหว่ตามมาตรฐาน OWASP Top 10

## Hardening Checklist / รายการตรวจสอบความปลอดภัย
- **Input Validation:** Use strict types and validation rules for all external data. / ใช้ Type และกฎการตรวจสอบที่เข้มงวดสำหรับข้อมูลภายนอกทั้งหมด
- **Secure Communication:** Ensure data in transit is encrypted. / มั่นใจว่าข้อมูลระหว่างการรับส่งถูกเข้ารหัส
- **Dependency Auditing:** Regularly check for vulnerable libraries. / ตรวจสอบ Library ที่มีช่องโหว่อย่างสม่ำเสมอ
- **Error Obfuscation:** Don't leak system details in error messages to users. / อย่าให้ข้อมูลภายในระบบหลุดไปใน Error Message ที่ส่งให้ User
