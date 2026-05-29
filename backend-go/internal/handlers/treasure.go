package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TreasureHandler struct {
	db *sql.DB
}

func NewTreasureHandler(db *sql.DB) *TreasureHandler {
	return &TreasureHandler{db: db}
}

// Create a treasure hunt (admin only)
func (h *TreasureHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	var isAdmin bool
	h.db.QueryRow("SELECT is_admin FROM users WHERE id::text=$1", userID).Scan(&isAdmin)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	var req struct {
		Title           string  `json:"title" binding:"required"`
		Description     string  `json:"description"`
		PrizeDescription string `json:"prize_description"`
		PrizeValueSBD   float64 `json:"prize_value_sbd"`
		ShopName        string  `json:"shop_name"`
		ShopContact     string  `json:"shop_contact"`
		Lat             float64 `json:"lat" binding:"required"`
		Lng             float64 `json:"lng" binding:"required"`
		MaxFinalists    int     `json:"max_finalists"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.MaxFinalists == 0 { req.MaxFinalists = 5 }
	id := uuid.New().String()
	_, err := h.db.Exec(`
		INSERT INTO treasure_hunts (id, title, description, prize_description, prize_value_sbd, shop_name, shop_contact, lat, lng, max_finalists, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, req.Title, req.Description, req.PrizeDescription, req.PrizeValueSBD,
		req.ShopName, req.ShopContact, req.Lat, req.Lng, req.MaxFinalists, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create treasure hunt"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Treasure hunt created!"})
}

// List all active treasure hunts
func (h *TreasureHandler) List(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, title, description, prize_description, prize_value_sbd, shop_name, lat, lng, status, max_finalists, created_at
		FROM treasure_hunts WHERE status='active' ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()
	type Hunt struct {
		ID               string    `json:"id"`
		Title            string    `json:"title"`
		Description      string    `json:"description"`
		PrizeDescription string    `json:"prize_description"`
		PrizeValueSBD    float64   `json:"prize_value_sbd"`
		ShopName         string    `json:"shop_name"`
		Lat              float64   `json:"lat"`
		Lng              float64   `json:"lng"`
		Status           string    `json:"status"`
		MaxFinalists     int       `json:"max_finalists"`
		CreatedAt        time.Time `json:"created_at"`
	}
	hunts := []Hunt{}
	for rows.Next() {
		var h Hunt
		rows.Scan(&h.ID, &h.Title, &h.Description, &h.PrizeDescription, &h.PrizeValueSBD,
			&h.ShopName, &h.Lat, &h.Lng, &h.Status, &h.MaxFinalists, &h.CreatedAt)
		hunts = append(hunts, h)
	}
	c.JSON(http.StatusOK, gin.H{"hunts": hunts})
}

// Add question to hunt (admin only)
func (h *TreasureHandler) AddQuestion(c *gin.Context) {
	userID := c.GetString("user_id")
	var isAdmin bool
	h.db.QueryRow("SELECT is_admin FROM users WHERE id::text=$1", userID).Scan(&isAdmin)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	huntID := c.Param("id")
	var req struct {
		Question      string   `json:"question" binding:"required"`
		Options       []string `json:"options" binding:"required"`
		CorrectAnswer string   `json:"correct_answer" binding:"required"`
		ClueAfter     string   `json:"clue_after"`
		ClueLat       float64  `json:"clue_lat"`
		ClueLng       float64  `json:"clue_lng"`
		OrderNum      int      `json:"order_num"`
		Difficulty    string   `json:"difficulty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Difficulty == "" { req.Difficulty = "medium" }
	optionsJSON := "["
	for i, o := range req.Options {
		if i > 0 { optionsJSON += "," }
		optionsJSON += `"` + o + `"`
	}
	optionsJSON += "]"
	id := uuid.New().String()
	_, err := h.db.Exec(`
		INSERT INTO treasure_questions (id, hunt_id, question, options, correct_answer, clue_after, clue_lat, clue_lng, order_num, difficulty)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9,$10)`,
		id, huntID, req.Question, optionsJSON, req.CorrectAnswer,
		req.ClueAfter, req.ClueLat, req.ClueLng, req.OrderNum, req.Difficulty)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add question: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Question added!"})
}

// Get questions for a hunt (without correct answers)
func (h *TreasureHandler) GetQuestions(c *gin.Context) {
	huntID := c.Param("id")
	rows, err := h.db.Query(`
		SELECT id, question, options, clue_after, clue_lat, clue_lng, order_num, difficulty
		FROM treasure_questions WHERE hunt_id::text=$1 ORDER BY order_num`, huntID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()
	type Question struct {
		ID         string   `json:"id"`
		Question   string   `json:"question"`
		Options    []string `json:"options"`
		ClueAfter  string   `json:"clue_after"`
		ClueLat    float64  `json:"clue_lat"`
		ClueLng    float64  `json:"clue_lng"`
		OrderNum   int      `json:"order_num"`
		Difficulty string   `json:"difficulty"`
	}
	questions := []Question{}
	for rows.Next() {
		var q Question
		var optionsJSON string
		var clueAfter sql.NullString
		var clueLat, clueLng sql.NullFloat64
		rows.Scan(&q.ID, &q.Question, &optionsJSON, &clueAfter, &clueLat, &clueLng, &q.OrderNum, &q.Difficulty)
		q.ClueAfter = clueAfter.String
		q.ClueLat = clueLat.Float64
		q.ClueLng = clueLng.Float64
		// Parse options JSON manually
		optionsJSON = optionsJSON[1:len(optionsJSON)-1]
		for _, o := range splitJSON(optionsJSON) {
			if len(o) > 2 { q.Options = append(q.Options, o[1:len(o)-1]) }
		}
		questions = append(questions, q)
	}
	c.JSON(http.StatusOK, gin.H{"questions": questions})
}

// Start or get attempt
func (h *TreasureHandler) StartAttempt(c *gin.Context) {
	huntID := c.Param("id")
	userID := c.GetString("user_id")
	var attemptID string
	var currentQ int
	var status string
	err := h.db.QueryRow(`SELECT id, current_question, status FROM treasure_attempts WHERE hunt_id::text=$1 AND user_id::text=$2`, huntID, userID).
		Scan(&attemptID, &currentQ, &status)
	if err == sql.ErrNoRows {
		attemptID = uuid.New().String()
		h.db.Exec(`INSERT INTO treasure_attempts (id, hunt_id, user_id) VALUES ($1,$2,$3)`, attemptID, huntID, userID)
		currentQ = 0
		status = "playing"
	}
	c.JSON(http.StatusOK, gin.H{"attempt_id": attemptID, "current_question": currentQ, "status": status})
}

// Submit answer
func (h *TreasureHandler) SubmitAnswer(c *gin.Context) {
	huntID := c.Param("id")
	userID := c.GetString("user_id")
	var req struct {
		QuestionID string `json:"question_id"`
		Answer     string `json:"answer"`
	}
	c.ShouldBindJSON(&req)

	// Get correct answer
	var correctAnswer string
	var clueAfter string
	var clueLat, clueLng sql.NullFloat64
	var orderNum int
	err := h.db.QueryRow(`SELECT correct_answer, COALESCE(clue_after,''), clue_lat, clue_lng, order_num FROM treasure_questions WHERE id::text=$1`, req.QuestionID).
		Scan(&correctAnswer, &clueAfter, &clueLat, &clueLng, &orderNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	isCorrect := req.Answer == correctAnswer

	if isCorrect {
		// Advance to next question
		h.db.Exec(`UPDATE treasure_attempts SET current_question=current_question+1 WHERE hunt_id::text=$1 AND user_id::text=$2`, huntID, userID)
		// Check if all questions done
		var totalQ int
		h.db.QueryRow(`SELECT COUNT(*) FROM treasure_questions WHERE hunt_id::text=$1`, huntID).Scan(&totalQ)
		var currentQ int
		h.db.QueryRow(`SELECT current_question FROM treasure_attempts WHERE hunt_id::text=$1 AND user_id::text=$2`, huntID, userID).Scan(&currentQ)
		if currentQ >= totalQ {
			// Check if finalist spots available
			var finalistCount int
			var maxFinalists int
			h.db.QueryRow(`SELECT COUNT(*) FROM treasure_attempts WHERE hunt_id::text=$1 AND status='finalist'`, huntID).Scan(&finalistCount)
			h.db.QueryRow(`SELECT max_finalists FROM treasure_hunts WHERE id::text=$1`, huntID).Scan(&maxFinalists)
			h.db.Exec(`UPDATE treasure_attempts SET status='winner', completed_at=NOW() WHERE hunt_id::text=$1 AND user_id::text=$2`, huntID, userID)
			h.db.Exec(`UPDATE treasure_hunts SET status='completed', winner_id=$1 WHERE id::text=$2`, userID, huntID)
			var winnerName string
			h.db.QueryRow(`SELECT username FROM users WHERE id::text=$1`, userID).Scan(&winnerName)
			c.JSON(http.StatusOK, gin.H{"correct": true, "winner": true, "message": "🏆 You answered all questions correctly! You are the winner!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"correct":    true,
			"clue":       clueAfter,
			"clue_lat":   clueLat.Float64,
			"clue_lng":   clueLng.Float64,
			"message":    "✅ Correct! Here is your next clue.",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{"correct": false, "message": "❌ Wrong answer! Try again."})
	}
}

// Submit bet (final challenge)
func (h *TreasureHandler) SubmitBet(c *gin.Context) {
	huntID := c.Param("id")
	userID := c.GetString("user_id")
	var req struct {
		BetOption string `json:"bet_option"`
	}
	c.ShouldBindJSON(&req)
	id := uuid.New().String()
	h.db.Exec(`INSERT INTO treasure_bets (id, hunt_id, user_id, bet_option) VALUES ($1,$2,$3,$4) ON CONFLICT (hunt_id, user_id) DO UPDATE SET bet_option=$4`,
		id, huntID, userID, req.BetOption)
	c.JSON(http.StatusOK, gin.H{"message": "Bet placed! Waiting for result."})
}

// Resolve bet (admin sets winner)
func (h *TreasureHandler) ResolveBet(c *gin.Context) {
	huntID := c.Param("id")
	userID := c.GetString("user_id")
	var isAdmin bool
	h.db.QueryRow("SELECT is_admin FROM users WHERE id::text=$1", userID).Scan(&isAdmin)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	var req struct {
		WinningOption string `json:"winning_option"`
	}
	c.ShouldBindJSON(&req)
	h.db.Exec(`UPDATE treasure_bets SET is_correct=(bet_option=$1) WHERE hunt_id::text=$2`, req.WinningOption, huntID)
	c.JSON(http.StatusOK, gin.H{"message": "Bet resolved!"})
}

// Get finalists
func (h *TreasureHandler) GetFinalists(c *gin.Context) {
	huntID := c.Param("id")
	rows, err := h.db.Query(`
		SELECT u.id, u.username, u.phone, ta.status, ta.completed_at
		FROM treasure_attempts ta JOIN users u ON u.id=ta.user_id
		WHERE ta.hunt_id::text=$1 AND ta.status IN ('finalist','winner')
		ORDER BY ta.completed_at ASC`, huntID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()
	type Finalist struct {
		ID          string    `json:"id"`
		Username    string    `json:"username"`
		Phone       string    `json:"phone"`
		Status      string    `json:"status"`
		CompletedAt time.Time `json:"completed_at"`
	}
	finalists := []Finalist{}
	for rows.Next() {
		var f Finalist
		var phone sql.NullString
		var completedAt sql.NullTime
		rows.Scan(&f.ID, &f.Username, &phone, &f.Status, &completedAt)
		f.Phone = phone.String
		if completedAt.Valid { f.CompletedAt = completedAt.Time }
		finalists = append(finalists, f)
	}
	c.JSON(http.StatusOK, gin.H{"finalists": finalists})
}

// Declare winner (admin)
func (h *TreasureHandler) DeclareWinner(c *gin.Context) {
	huntID := c.Param("id")
	userID := c.GetString("user_id")
	var isAdmin bool
	h.db.QueryRow("SELECT is_admin FROM users WHERE id::text=$1", userID).Scan(&isAdmin)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	var req struct {
		WinnerID string `json:"winner_id"`
	}
	c.ShouldBindJSON(&req)
	h.db.Exec(`UPDATE treasure_hunts SET status='completed', winner_id=$1 WHERE id::text=$2`, req.WinnerID, huntID)
	h.db.Exec(`UPDATE treasure_attempts SET status='winner' WHERE hunt_id::text=$1 AND user_id=$2`, huntID, req.WinnerID)
	var winnerName, winnerPhone string
	h.db.QueryRow(`SELECT username, COALESCE(phone,'') FROM users WHERE id=$1`, req.WinnerID).Scan(&winnerName, &winnerPhone)
	c.JSON(http.StatusOK, gin.H{"message": "Winner declared!", "winner": winnerName, "phone": winnerPhone})
}

func splitJSON(s string) []string {
	var result []string
	var current string
	inQuote := false
	for _, c := range s {
		if c == '"' { inQuote = !inQuote }
		if c == ',' && !inQuote {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" { result = append(result, current) }
	return result
}

// Reset attempt (for testing)
func (h *TreasureHandler) ResetAttempt(c *gin.Context) {
	huntID := c.Param("id")
	userID := c.GetString("user_id")
	h.db.Exec(`DELETE FROM treasure_attempts WHERE hunt_id::text=$1 AND user_id::text=$2`, huntID, userID)
	h.db.Exec(`UPDATE treasure_hunts SET status='active', winner_id=NULL WHERE id::text=$1`, huntID)
	c.JSON(http.StatusOK, gin.H{"message": "Attempt reset!"})
}


// Delete hunt (admin only)
func (h *TreasureHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	var isAdmin bool
	h.db.QueryRow("SELECT is_admin FROM users WHERE id::text=$1", userID).Scan(&isAdmin)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins only"})
		return
	}
	huntID := c.Param("id")
	h.db.Exec(`DELETE FROM treasure_bets WHERE hunt_id::text=$1`, huntID)
	h.db.Exec(`DELETE FROM treasure_attempts WHERE hunt_id::text=$1`, huntID)
	h.db.Exec(`DELETE FROM treasure_questions WHERE hunt_id::text=$1`, huntID)
	h.db.Exec(`DELETE FROM treasure_hunts WHERE id::text=$1`, huntID)
	c.JSON(http.StatusOK, gin.H{"message": "Hunt deleted!"})
}
