package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

// --- Models ---

type Choice struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type Question struct {
	ID      int      `json:"id"`
	Body    string   `json:"body"`
	Choices []Choice `json:"choices"`
}

// SubmitRequest: examinee name + map of question_id -> choice_id
type SubmitRequest struct {
	Examinee string         `json:"examinee" binding:"required"`
	Answers  map[string]int `json:"answers"  binding:"required"`
}

type AnswerDetail struct {
	QuestionID  int    `json:"question_id"`
	ChosenID    *int   `json:"chosen_id"`
	CorrectID   int    `json:"correct_id"`
	CorrectBody string `json:"correct_body"`
	IsCorrect   bool   `json:"is_correct"`
}

type SubmitResponse struct {
	ID       int64          `json:"id"`
	Examinee string         `json:"examinee"`
	Score    int            `json:"score"`
	Total    int            `json:"total"`
	Details  []AnswerDetail `json:"details"`
}

type ExamResult struct {
	ID          int64  `json:"id"`
	Examinee    string `json:"examinee"`
	Score       int    `json:"score"`
	Total       int    `json:"total"`
	SubmittedAt string `json:"submitted_at"`
}

// --- Database ---

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "quiz.db")
	if err != nil {
		log.Fatal("cannot open database:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS questions (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			body TEXT    NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("create table questions failed:", err)
	}

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
		log.Fatal("create table choices failed:", err)
	}

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
		log.Fatal("create table exam_results failed:", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	if count == 0 {
		seedData()
		log.Println("seed data inserted")
	}

	log.Println("database ready (quiz.db)")
}

func seedData() {
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

// --- Handlers ---

// GET /api/questions - returns all questions with choices (is_correct excluded)
func getQuestions(c *gin.Context) {
	rows, err := db.Query("SELECT id, body FROM questions ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
		return
	}
	defer rows.Close()

	questions := []Question{}
	for rows.Next() {
		var q Question
		rows.Scan(&q.ID, &q.Body)

		cRows, err := db.Query(
			"SELECT id, body FROM choices WHERE question_id = ? ORDER BY id", q.ID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch choices"})
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

// POST /api/submit - grade answers, save result, return score + details
func submitExam(c *gin.Context) {
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Examinee == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "examinee name is required"})
		return
	}

	rows, err := db.Query("SELECT id FROM questions ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
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

	for _, qid := range questionIDs {
		var correctID int
		var correctBody string
		err := db.QueryRow(
			"SELECT id, body FROM choices WHERE question_id = ? AND is_correct = 1",
			qid,
		).Scan(&correctID, &correctBody)
		if err != nil {
			continue
		}

		chosenVal, answered := req.Answers[strconv.Itoa(qid)]
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

	submittedAt := time.Now().Format("2006-01-02 15:04:05")
	result, err := db.Exec(
		"INSERT INTO exam_results (examinee, score, total, submitted_at) VALUES (?, ?, ?, ?)",
		req.Examinee, score, total, submittedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save result"})
		return
	}

	resultID, _ := result.LastInsertId()
	c.JSON(http.StatusOK, SubmitResponse{
		ID:       resultID,
		Examinee: req.Examinee,
		Score:    score,
		Total:    total,
		Details:  details,
	})
}

// GET /api/results - returns all exam history, newest first
func getResults(c *gin.Context) {
	rows, err := db.Query(
		"SELECT id, examinee, score, total, submitted_at FROM exam_results ORDER BY submitted_at DESC",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch results"})
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

// GET /api/health - server health check
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"company": "example.com",
	})
}

// --- Main ---

func main() {
	initDB()
	defer db.Close()

	r := gin.Default()

	// Allow requests from Vue dev server and production frontend
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: false,
	}))

	api := r.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/questions", getQuestions)
		api.POST("/submit", submitExam)
		api.GET("/results", getResults)
	}

	log.Println("server running at http://localhost:8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
