// Package main เป็น entry point ของ Backend API
// บริษัท: example.com
// โปรเจกต์: interview-question-010
// ภาษา: Go 1.24 + Gin Framework + SQLite (modernc.org/sqlite)
// วัตถุประสงค์: ระบบสอบออนไลน์ที่รองรับการสอบ, บันทึกคะแนน และแสดงประวัติ
package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite" // SQLite driver แบบ pure Go (ไม่ต้องใช้ CGo)
)

// ─────────────────────────────────────────────────────────────────────────────
// Struct definitions — โครงสร้างข้อมูลที่ใช้ในระบบ
// ─────────────────────────────────────────────────────────────────────────────

// Choice คือตัวเลือกคำตอบของแต่ละข้อ
type Choice struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

// Question คือข้อสอบหนึ่งข้อ พร้อม choices
type Question struct {
	ID      int      `json:"id"`
	Body    string   `json:"body"`
	Choices []Choice `json:"choices"`
}

// SubmitRequest คือ body ที่ frontend ส่งมาเมื่อกดส่งข้อสอบ
// answers เป็น map ของ question_id -> choice_id
type SubmitRequest struct {
	Examinee string         `json:"examinee" binding:"required"`
	Answers  map[string]int `json:"answers"  binding:"required"`
}

// AnswerDetail คือผลเฉลยรายข้อสำหรับแสดงในหน้า IT 10-2
type AnswerDetail struct {
	QuestionID  int    `json:"question_id"`
	ChosenID    *int   `json:"chosen_id"`
	CorrectID   int    `json:"correct_id"`
	CorrectBody string `json:"correct_body"`
	IsCorrect   bool   `json:"is_correct"`
}

// SubmitResponse คือ response ที่ส่งกลับหลังจากบันทึกผลสอบ
type SubmitResponse struct {
	ID       int64          `json:"id"`
	Examinee string         `json:"examinee"`
	Score    int            `json:"score"`
	Total    int            `json:"total"`
	Details  []AnswerDetail `json:"details"`
}

// ExamResult คือแถวข้อมูลในตาราง exam_results สำหรับแสดงประวัติ
type ExamResult struct {
	ID          int64  `json:"id"`
	Examinee    string `json:"examinee"`
	Score       int    `json:"score"`
	Total       int    `json:"total"`
	SubmittedAt string `json:"submitted_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Database — การเชื่อมต่อและ migration
// ─────────────────────────────────────────────────────────────────────────────

// db เป็น global database connection ที่ใช้ร่วมกันทั้ง application
var db *sql.DB

// initDB เปิด connection ไปยัง SQLite และสร้างตาราง + seed ข้อมูลตัวอย่าง
func initDB() {
	var err error

	// เปิดไฟล์ quiz.db (สร้างใหม่อัตโนมัติถ้ายังไม่มี)
	db, err = sql.Open("sqlite", "quiz.db")
	if err != nil {
		log.Fatal("ไม่สามารถเปิด database ได้:", err)
	}

	// สร้างตาราง questions — เก็บข้อสอบ
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS questions (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			body TEXT    NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("สร้างตาราง questions ไม่สำเร็จ:", err)
	}

	// สร้างตาราง choices — เก็บตัวเลือกของแต่ละข้อ
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS choices (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			question_id INTEGER NOT NULL,
			body        TEXT    NOT NULL,
			is_correct  INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (question_id) REFERENCES questions(id)
		)
	`)
	if err != nil {
		log.Fatal("สร้างตาราง choices ไม่สำเร็จ:", err)
	}

	// สร้างตาราง exam_results — เก็บผลการสอบของผู้เข้าสอบแต่ละคน
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS exam_results (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			examinee     TEXT    NOT NULL,
			score        INTEGER NOT NULL,
			total        INTEGER NOT NULL,
			submitted_at TEXT    NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("สร้างตาราง exam_results ไม่สำเร็จ:", err)
	}

	// Seed ข้อมูลตัวอย่างถ้า questions ยังว่างอยู่
	var count int
	db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	if count == 0 {
		seedData()
		log.Println("✅ Seed mock data สำเร็จ")
	}

	log.Println("✅ Database พร้อมใช้งาน (quiz.db)")
}

// seedData ใส่ข้อสอบตัวอย่าง 5 ข้อลงในฐานข้อมูล
func seedData() {
	// โครงสร้างข้อมูล seed แบบ anonymous struct เพื่อให้อ่านง่าย
	type seedChoice struct {
		body      string
		isCorrect bool
	}
	type seedQuestion struct {
		body    string
		choices []seedChoice
	}

	questions := []seedQuestion{
		{
			body: "ภาษา Go ถูกพัฒนาโดยบริษัทใด?",
			choices: []seedChoice{
				{"Apple", false},
				{"Google", true},
				{"Microsoft", false},
				{"Amazon", false},
			},
		},
		{
			body: "HTTP Status Code 404 หมายความว่าอะไร?",
			choices: []seedChoice{
				{"Internal Server Error", false},
				{"Unauthorized", false},
				{"Not Found", true},
				{"Bad Request", false},
			},
		},
		{
			body: "ข้อใดคือ Primary Key ในฐานข้อมูลเชิงสัมพันธ์?",
			choices: []seedChoice{
				{"คอลัมน์ที่มีค่าซ้ำกันได้", false},
				{"คอลัมน์ที่ระบุแถวข้อมูลได้อย่างไม่ซ้ำกัน", true},
				{"คอลัมน์ที่เชื่อมตารางอื่น", false},
				{"คอลัมน์ที่เก็บค่า NULL ได้", false},
			},
		},
		{
			body: "Vue 3 ใช้ Composition API ผ่าน function ใด?",
			choices: []seedChoice{
				{"createApp()", false},
				{"setup()", true},
				{"mounted()", false},
				{"defineProps()", false},
			},
		},
		{
			body: "RESTful API ใช้ HTTP Method ใดในการสร้างข้อมูลใหม่?",
			choices: []seedChoice{
				{"GET", false},
				{"PUT", false},
				{"POST", true},
				{"DELETE", false},
			},
		},
	}

	// วน insert ทีละข้อพร้อม choices
	for _, q := range questions {
		result, err := db.Exec("INSERT INTO questions (body) VALUES (?)", q.body)
		if err != nil {
			log.Println("seed question error:", err)
			continue
		}
		qid, _ := result.LastInsertId()

		for _, c := range q.choices {
			correct := 0
			if c.isCorrect {
				correct = 1
			}
			_, err = db.Exec(
				"INSERT INTO choices (question_id, body, is_correct) VALUES (?, ?, ?)",
				qid, c.body, correct,
			)
			if err != nil {
				log.Println("seed choice error:", err)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Handlers — ฟังก์ชันจัดการแต่ละ API endpoint
// ─────────────────────────────────────────────────────────────────────────────

// getQuestions คืนรายการข้อสอบทั้งหมดพร้อม choices
// GET /api/questions
// หมายเหตุ: ไม่ส่ง is_correct กลับไปเพื่อป้องกันการโกง
func getQuestions(c *gin.Context) {
	// ดึงข้อสอบทั้งหมด
	rows, err := db.Query("SELECT id, body FROM questions ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึงข้อสอบไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	questions := []Question{}
	for rows.Next() {
		var q Question
		rows.Scan(&q.ID, &q.Body)

		// ดึง choices ของแต่ละข้อ (ไม่รวม is_correct)
		cRows, err := db.Query(
			"SELECT id, body FROM choices WHERE question_id = ? ORDER BY id", q.ID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึง choices ไม่สำเร็จ"})
			return
		}
		defer cRows.Close()

		q.Choices = []Choice{}
		for cRows.Next() {
			var ch Choice
			cRows.Scan(&ch.ID, &ch.Body)
			q.Choices = append(q.Choices, ch)
		}

		questions = append(questions, q)
	}

	c.JSON(http.StatusOK, questions)
}

// submitExam รับคำตอบจาก frontend, ตรวจคะแนน และบันทึกลงฐานข้อมูล
// POST /api/submit
// Body: { "examinee": "ชื่อ", "answers": { "1": 2, "2": 5, ... } }
func submitExam(c *gin.Context) {
	var req SubmitRequest

	// Validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง"})
		return
	}
	if req.Examinee == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณากรอกชื่อผู้สอบ"})
		return
	}

	// ดึงข้อสอบทั้งหมดเพื่อใช้ตรวจคะแนน
	rows, err := db.Query("SELECT id FROM questions ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึงข้อสอบไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	var questionIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		questionIDs = append(questionIDs, id)
	}

	total := len(questionIDs)
	score := 0
	details := []AnswerDetail{}

	// ตรวจคะแนนทีละข้อ โดยเปรียบเทียบกับ correct choice
	for _, qid := range questionIDs {
		// ดึง choice ที่ถูกต้องของข้อนี้
		var correctID int
		var correctBody string
		err := db.QueryRow(
			"SELECT id, body FROM choices WHERE question_id = ? AND is_correct = 1",
			qid,
		).Scan(&correctID, &correctBody)
		if err != nil {
			continue
		}

		// หา choice ที่ผู้สอบเลือก (อาจไม่ได้ตอบ)
		chosenIDStr := strconv.Itoa(qid)
		chosenVal, answered := req.Answers[chosenIDStr]

		var chosenPtr *int
		isCorrect := false

		if answered {
			chosenPtr = &chosenVal
			isCorrect = chosenVal == correctID
		}

		if isCorrect {
			score++
		}

		details = append(details, AnswerDetail{
			QuestionID:  qid,
			ChosenID:    chosenPtr,
			CorrectID:   correctID,
			CorrectBody: correctBody,
			IsCorrect:   isCorrect,
		})
	}

	// บันทึกผลสอบลง exam_results
	submittedAt := time.Now().Format("2006-01-02 15:04:05")
	result, err := db.Exec(
		"INSERT INTO exam_results (examinee, score, total, submitted_at) VALUES (?, ?, ?, ?)",
		req.Examinee, score, total, submittedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกผลสอบไม่สำเร็จ"})
		return
	}

	resultID, _ := result.LastInsertId()

	// ส่งผลลัพธ์กลับไปให้ frontend แสดงในหน้า IT 10-2
	c.JSON(http.StatusOK, SubmitResponse{
		ID:       resultID,
		Examinee: req.Examinee,
		Score:    score,
		Total:    total,
		Details:  details,
	})
}

// getResults คืนประวัติการสอบทั้งหมดเรียงจากใหม่ไปเก่า
// GET /api/results
func getResults(c *gin.Context) {
	rows, err := db.Query(
		"SELECT id, examinee, score, total, submitted_at FROM exam_results ORDER BY submitted_at DESC",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึงประวัติไม่สำเร็จ"})
		return
	}
	defer rows.Close()

	results := []ExamResult{}
	for rows.Next() {
		var r ExamResult
		rows.Scan(&r.ID, &r.Examinee, &r.Score, &r.Total, &r.SubmittedAt)
		results = append(results, r)
	}

	c.JSON(http.StatusOK, results)
}

// healthCheck ใช้ตรวจสอบว่า server ทำงานอยู่
// GET /api/health
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"company": "example.com",
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// main — เริ่มต้น server
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// เริ่มต้น database
	initDB()
	defer db.Close()

	// สร้าง Gin router (ใช้ Release mode ใน production)
	r := gin.Default()

	// ตั้งค่า CORS เพื่อให้ Vue frontend (port 5173) เรียก API ได้
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: false,
	}))

	// ─── Routes ───────────────────────────────────────────────
	api := r.Group("/api")
	{
		// GET  /api/health    — ตรวจสอบสถานะ server
		api.GET("/health", healthCheck)

		// GET  /api/questions — ดึงข้อสอบทั้งหมด (IT 10-1)
		api.GET("/questions", getQuestions)

		// POST /api/submit    — ส่งคำตอบ + บันทึกคะแนน (IT 10-1 → IT 10-2)
		api.POST("/submit", submitExam)

		// GET  /api/results   — ดึงประวัติการสอบทั้งหมด (IT 10-2)
		api.GET("/results", getResults)
	}

	// รัน server บน port 8000
	log.Println("🚀 Server เริ่มทำงานที่ http://localhost:8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatal("ไม่สามารถเริ่ม server ได้:", err)
	}
}
