package handlers

import (
	"database/sql"
	"log"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BountyHandler struct {
	db *sql.DB
}

func NewBountyHandler(db *sql.DB) *BountyHandler {
	return &BountyHandler{db: db}
}

func (h *BountyHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		Title       string  `json:"title"       binding:"required"`
		Description string  `json:"description"`
		Lat         float64 `json:"lat"         binding:"required"`
		Lng         float64 `json:"lng"         binding:"required"`
		RewardSBD   float64 `json:"reward_sbd"`
		SubmitType  string  `json:"submit_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.RewardSBD == 0 { req.RewardSBD = 5.00 }
	if req.SubmitType == "" { req.SubmitType = "both" }
	id := uuid.New().String()
	_, err := h.db.Exec(`
		INSERT INTO bounty_jobs
			(id, title, description, lat, lng, geom, reward_sbd, submit_type, created_by)
		VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($5, $4), 4326), $6, $7, $8)
	`, id, req.Title, req.Description, req.Lat, req.Lng, req.RewardSBD, req.SubmitType, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bounty"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Bounty created"})
}

func (h *BountyHandler) List(c *gin.Context) {
	lat    := parseQueryFloat(c, "lat",    -9.4313)
	lng    := parseQueryFloat(c, "lng",    160.0521)
	radius := parseQueryFloat(c, "radius", 10000)
	rows, err := h.db.Query(`
		SELECT id, title, description, lat, lng,
			reward_sbd, submit_type, status, claimed_by, created_at
		FROM bounty_jobs
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3
		)
		AND status IN ('open', 'claimed', 'submitted')
		ORDER BY created_at DESC
	`, lat, lng, radius)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()
	type Bounty struct {
		ID          string    `json:"id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Lat         float64   `json:"lat"`
		Lng         float64   `json:"lng"`
		RewardSBD   float64   `json:"reward_sbd"`
		SubmitType  string    `json:"submit_type"`
		Status      string    `json:"status"`
		ClaimedBy   string    `json:"claimed_by,omitempty"`
		CreatedAt   time.Time `json:"created_at"`
	}
	bounties := []Bounty{}
	for rows.Next() {
		var b Bounty
		var desc, claimedBy sql.NullString
		rows.Scan(&b.ID, &b.Title, &desc, &b.Lat, &b.Lng,
			&b.RewardSBD, &b.SubmitType, &b.Status, &claimedBy, &b.CreatedAt)
		b.Description = desc.String
		b.ClaimedBy   = claimedBy.String
		bounties = append(bounties, b)
	}
	c.JSON(http.StatusOK, gin.H{"bounties": bounties})
}

func (h *BountyHandler) Claim(c *gin.Context) {
	id     := c.Param("id")
	userID := c.GetString("user_id")
	result, err := h.db.Exec(`
		UPDATE bounty_jobs SET status='claimed', claimed_by=$1, claimed_at=NOW()
		WHERE id::text=$2 AND status='open'
	`, userID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Claim failed"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Bounty already claimed or not available"})
		return
	}
	h.db.Exec(`INSERT INTO wallets (id, user_id) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		uuid.New().String(), userID)
	c.JSON(http.StatusOK, gin.H{"message": "Bounty claimed! Go take your photos/videos."})
}

func (h *BountyHandler) Submit(c *gin.Context) {
	id     := c.Param("id")
	userID := c.GetString("user_id")
	log.Printf("Submit called: id=%q userID=%q", id, userID)
	var req struct {
		Files []struct {
			URL      string  `json:"url"`
			FileType string  `json:"file_type"`
			FileSize int64   `json:"file_size"`
			Lat      float64 `json:"lat"`
			Lng      float64 `json:"lng"`
		} `json:"files"`
	}
	c.ShouldBindJSON(&req)
	var claimedBy string
	err := h.db.QueryRow(`SELECT COALESCE(claimed_by::text,'') FROM bounty_jobs WHERE id::text=$1`, id).Scan(&claimedBy)
	if err != nil || claimedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You have not claimed this bounty"})
		return
	}
	for _, f := range req.Files {
		h.db.Exec(`INSERT INTO bounty_submissions (id,job_id,user_id,file_url,file_type,file_size,lat,lng)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			uuid.New().String(), id, userID, f.URL, f.FileType, f.FileSize, f.Lat, f.Lng)
	}
	h.db.Exec(`UPDATE bounty_jobs SET status='submitted', submitted_at=NOW() WHERE id::text=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "Submitted! Waiting for admin review."})
}

func (h *BountyHandler) UploadFile(c *gin.Context) {
	id     := c.Param("id")
	userID := c.GetString("user_id")
	log.Printf("UploadFile called: id=%q userID=%q len=%d bytes=%v", id, userID, len(id), []byte(id))

	// Parse multipart form BEFORE database query
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("ParseMultipartForm error: %v", err)
	}

	// Re-read param after form parsing
	id2 := c.Param("id")
	log.Printf("id after form parse: %q", id2)

	var claimedBy, status string
	queryID := id2
	log.Printf("Using queryID: %q len=%d", queryID, len(queryID))
	err := h.db.QueryRow("SELECT COALESCE(claimed_by::text,''), status FROM bounty_jobs WHERE id::text=$1", queryID).
		Scan(&claimedBy, &status)
	log.Printf("Query result: claimedBy=%q status=%q err=%v", claimedBy, status, err)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bounty not found: " + err.Error()})
		return
	}
	if claimedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your bounty. claimed_by=" + claimedBy + " but you are=" + userID})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	fileType := "photo"
	if strings.Contains(contentType, "video") {
		fileType = "video"
	}

	// Read file bytes and encode as base64
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	b64 := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(fileBytes)

	submissionID := uuid.New().String()
	h.db.Exec(`INSERT INTO bounty_submissions (id,job_id,user_id,file_url,file_type,file_size)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		submissionID, id, userID, b64, fileType, header.Size)

	var fileCount int
	h.db.QueryRow(`SELECT COUNT(*) FROM bounty_submissions WHERE job_id::text=$1`, id).Scan(&fileCount)

	c.JSON(http.StatusOK, gin.H{
		"file_url":   "/submissions/" + submissionID,
		"file_type":  fileType,
		"file_count": fileCount,
		"message":    "File uploaded successfully",
	})
}

func (h *BountyHandler) Approve(c *gin.Context) {
	id      := c.Param("id")
	adminID := c.GetString("user_id")
	var claimedBy string
	var rewardSBD float64
	err := h.db.QueryRow(`SELECT claimed_by, reward_sbd FROM bounty_jobs WHERE id::text=$1 AND status='submitted'`, id).
		Scan(&claimedBy, &rewardSBD)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or not submitted yet"})
		return
	}
	h.db.Exec(`UPDATE bounty_jobs SET status='approved', approved_at=NOW(), approved_by=$1 WHERE id::text=$2`, adminID, id)
	var walletID string
	h.db.QueryRow(`SELECT id FROM wallets WHERE user_id=$1`, claimedBy).Scan(&walletID)
	h.db.Exec(`UPDATE wallets SET balance_sbd=balance_sbd+$1, total_earned=total_earned+$1, updated_at=NOW() WHERE user_id=$2`,
		rewardSBD, claimedBy)
	h.db.Exec(`INSERT INTO wallet_transactions (id,wallet_id,job_id,amount_sbd,type,note) VALUES ($1,$2,$3,$4,'credit','Bounty approved')`,
		uuid.New().String(), walletID, id, rewardSBD)
	c.JSON(http.StatusOK, gin.H{"message": "Approved! Payment sent.", "reward_sbd": rewardSBD, "paid_to": claimedBy})
}

func (h *BountyHandler) GetWallet(c *gin.Context) {
	userID := c.GetString("user_id")
	var balance, totalEarned, totalWithdrawn float64
	err := h.db.QueryRow(`SELECT balance_sbd, total_earned, total_withdrawn FROM wallets WHERE user_id=$1`, userID).
		Scan(&balance, &totalEarned, &totalWithdrawn)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"balance_sbd": 0, "total_earned": 0, "total_withdrawn": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance_sbd": balance, "total_earned": totalEarned, "total_withdrawn": totalWithdrawn})
}

func (h *BountyHandler) GetSubmitted(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT j.id, j.title, j.lat, j.lng, j.reward_sbd, j.submit_type,
			j.submitted_at, u.username, COUNT(s.id) AS file_count
		FROM bounty_jobs j
		JOIN users u ON u.id = j.claimed_by
		LEFT JOIN bounty_submissions s ON s.job_id = j.id
		WHERE j.status = 'submitted'
		GROUP BY j.id, u.username
		ORDER BY j.submitted_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()
	type Job struct {
		ID          string    `json:"id"`
		Title       string    `json:"title"`
		Lat         float64   `json:"lat"`
		Lng         float64   `json:"lng"`
		RewardSBD   float64   `json:"reward_sbd"`
		SubmitType  string    `json:"submit_type"`
		SubmittedAt time.Time `json:"submitted_at"`
		Username    string    `json:"username"`
		FileCount   int       `json:"file_count"`
	}
	jobs := []Job{}
	for rows.Next() {
		var j Job
		var submittedAt sql.NullTime
		rows.Scan(&j.ID, &j.Title, &j.Lat, &j.Lng, &j.RewardSBD, &j.SubmitType,
			&submittedAt, &j.Username, &j.FileCount)
		if submittedAt.Valid { j.SubmittedAt = submittedAt.Time }
		jobs = append(jobs, j)
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (h *BountyHandler) GetFiles(c *gin.Context) {
	id := c.Param("id")
	rows, err := h.db.Query(`
		SELECT id, file_url, file_type, file_size, lat, lng, created_at
		FROM bounty_submissions WHERE job_id=$1 ORDER BY created_at ASC
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()
	type File struct {
		ID        string    `json:"id"`
		URL       string    `json:"url"`
		FileType  string    `json:"file_type"`
		FileSize  int64     `json:"file_size"`
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		CreatedAt time.Time `json:"created_at"`
	}
	files := []File{}
	for rows.Next() {
		var f File
		var lat, lng sql.NullFloat64
		var fileSize sql.NullInt64
		rows.Scan(&f.ID, &f.URL, &f.FileType, &fileSize, &lat, &lng, &f.CreatedAt)
		f.FileSize = fileSize.Int64
		f.Lat = lat.Float64
		f.Lng = lng.Float64
		files = append(files, f)
	}
	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *BountyHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	h.db.Exec(`DELETE FROM bounty_jobs WHERE id = $1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
// force redeploy Wed May 27 15:32:52 AEST 2026
