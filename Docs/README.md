# Docs Index

เอกสารในโปรเจกต์นี้ถูกจัดใหม่ให้เหลือเฉพาะชุดที่ใช้งานจริงและตรงกับโค้ดปัจจุบัน

## เอกสารหลัก

- [ARCHITECTURE.md](./ARCHITECTURE.md) ใช้อ่านโครงสร้างระบบ, design pattern, flow คำนวณ, data model และข้อจำกัดของ implementation ปัจจุบัน
- [API_TESTING_GUIDE.md](./API_TESTING_GUIDE.md) ใช้เทส API และ Postman pipeline สำหรับพิสูจน์โจทย์เรื่อง promotion stacking, extensibility และ correctness
- [PROMOTION_POLICY_REFERENCE.md](./PROMOTION_POLICY_REFERENCE.md) ใช้อ่านคำอธิบายเชิงอ้างอิงของคำในภาพ เช่น stackable, exclusive, non-stackable, deactivate, usage, replay พร้อมลิงก์ไปยังโค้ดจริงระดับบรรทัด
- [LOGIC_AND_RESULT_WALKTHROUGH.md](./LOGIC_AND_RESULT_WALKTHROUGH.md) ใช้อ่าน flow logic จาก request ไปจนถึง result และ calculation log แบบละเอียด
- [FUTURE_STRATEGY_GAP_ANALYSIS.md](./FUTURE_STRATEGY_GAP_ANALYSIS.md) ใช้ดูช่องว่างของ `Future Strategy` เทียบกับภาพ architecture

## เอกสารฐานข้อมูล

- [database/README.md](../database/README.md) อธิบาย schema หลัก, index สำคัญ และ query pattern ที่ระบบใช้จริง

## ไฟล์อ้างอิงอื่นในโฟลเดอร์นี้

- `.pdf`, `.png`, `.sql`, `.txt` เป็นเอกสารอ้างอิงเดิมจากช่วงออกแบบ
- ใช้เป็น supporting material ได้ แต่ไม่ถือเป็น source of truth หลัก
- source of truth ฝั่งเอกสาร markdown ตอนนี้คือ [README.md](../README.md), `Docs/*.md`, และ [database/README.md](../database/README.md)
