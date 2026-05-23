# interview-question-010

ระบบสอบออนไลน์ | example.com

## Stack

| Layer    | Technology              |
|----------|-------------------------|
| Backend  | Go 1.24 + Gin + SQLite  |
| Frontend | Vue 3 + Vite            |
| Deploy   | Docker Compose          |

## หน้าที่มีในระบบ

| หน้า    | คำอธิบาย                                                   |
|---------|------------------------------------------------------------|
| IT 10-1 | แบบทดสอบ — กรอกชื่อ, เลือกคำตอบ (single choice), ส่งข้อสอบ |
| IT 10-2 | ผลคะแนน, เฉลย, ตารางประวัติการสอบ, ปุ่ม "สอบอีกครั้ง"      |

## วิธีรัน (Development)

### Backend
```bash
cd backend
go mod tidy
go run main.go
# → http://localhost:8000
```

### Frontend
```bash
cd frontend
npm install
npm run dev
# → http://localhost:5173
```

## วิธีรัน (Docker)

```bash
docker compose up --build
# Frontend → http://localhost:5173
# Backend  → http://localhost:8000
```

## API Endpoints

| Method | Path           | Description                 |
|--------|----------------|-----------------------------|
| GET    | /api/questions | ดึงคำถามทั้งหมดพร้อม choices |
| POST   | /api/submit    | ส่งคำตอบ + คำนวณคะแนน       |
| GET    | /api/results   | ประวัติการสอบทั้งหมด         |
| GET    | /api/health    | Health check                |

### POST /api/submit — Request body
```json
{
  "examinee": "สมชาย ใจดี",
  "answers": {
    "1": 2,
    "2": 5,
    "3": 9
  }
}
```

## Database Schema

```sql
questions    (id, body)
choices      (id, question_id, body, is_correct)
exam_results (id, examinee, score, total, submitted_at)
```

> Mock data (5 ข้อ) จะถูก seed อัตโนมัติเมื่อ DB ว่างเปล่า