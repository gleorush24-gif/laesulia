package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Question struct {
	ID            string   `json:"id"`
	Subject       string   `json:"subject"`
	GradeLevel    string   `json:"grade_level"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	Difficulty    string   `json:"difficulty"`
}

// GetSIQuestions returns questions from SI database
func GetSIQuestions(c *gin.Context) {
	subject := c.Query("subject")
	gradeLevel := c.Query("grade_level")
	
	query := `SELECT id, subject, grade_level, question, options, correct_answer, difficulty 
	         FROM si_questions WHERE 1=1`
	
	if subject != "" {
		query += ` AND subject = '` + subject + `'`
	}
	if gradeLevel != "" {
		query += ` AND grade_level = '` + gradeLevel + `'`
	}
	query += ` ORDER BY RANDOM() LIMIT 5`
	
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON string
		if err := rows.Scan(&q.ID, &q.Subject, &q.GradeLevel, &q.Question, &optionsJSON, &q.CorrectAnswer, &q.Difficulty); err != nil {
			continue
		}
		json.Unmarshal([]byte(optionsJSON), &q.Options)
		questions = append(questions, q)
	}
	
	c.JSON(http.StatusOK, gin.H{"questions": questions})
}

// GetOpenTriviaQuestions fetches from Open Trivia API
func GetOpenTriviaQuestions(c *gin.Context) {
	subject := c.Query("subject")
	difficulty := c.DefaultQuery("difficulty", "easy")
	
	categoryMap := map[string]string{
		"Science":      "17",
		"History":      "23",
		"Geography":    "22",
		"Math":         "19",
		"Computers":    "18",
	}
	
	categoryID := categoryMap[subject]
	if categoryID == "" {
		categoryID = "9"
	}
	
	url := "https://opentdb.com/api.php?amount=5&category=" + categoryID + "&difficulty=" + difficulty + "&type=multiple"
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	
	var result struct {
		Results []struct {
			Question         string   `json:"question"`
			CorrectAnswer    string   `json:"correct_answer"`
			IncorrectAnswers []string `json:"incorrect_answers"`
		} `json:"results"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var questions []Question
	for _, q := range result.Results {
		options := q.IncorrectAnswers
		options = append(options, q.CorrectAnswer)
		questions = append(questions, Question{
			Question:      q.Question,
			Options:       options,
			CorrectAnswer: q.CorrectAnswer,
			Difficulty:    difficulty,
			Subject:       subject,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{"questions": questions})
}
