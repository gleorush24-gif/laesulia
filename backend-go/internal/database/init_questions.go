package database

func InitSIQuestionsTable(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS si_questions (
	  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	  subject VARCHAR(50),
	  grade_level VARCHAR(20),
	  question TEXT NOT NULL,
	  options JSONB NOT NULL,
	  correct_answer TEXT NOT NULL,
	  difficulty VARCHAR(20),
	  created_at TIMESTAMP DEFAULT NOW()
	);
	`
	
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}
	
	// Insert sample questions
	insertSQL := `
	INSERT INTO si_questions (subject, grade_level, question, options, correct_answer, difficulty)
	VALUES 
	($1, $2, $3, $4, $5, $6)
	ON CONFLICT DO NOTHING;
	`
	
	questions := []struct {
		subject, grade, question, options, correct, difficulty string
	}{
		("History", "Year 6", "What year did Solomon Islands gain independence?", `["1975", "1978", "1980", "1985"]`, "1978", "easy"),
		("History", "Year 7", "Who was the first PM?", `["Peter Kenilorea", "Manasseh Sogavare", "Bartholomew Ulufa'u", "Francis Billy Hilly"]`, "Peter Kenilorea", "medium"),
		("Science", "Year 6", "What is the capital?", `["Honiara", "Gizo", "Auki", "Buala"]`, "Honiara", "easy"),
		("Math", "Year 6", "What is 4 × 4?", `["12", "16", "20", "24"]`, "16", "easy"),
		("Math", "Year 7", "What is 5² + 3²?", `["8", "25", "34", "64"]`, "34", "medium"),
	}
	
	for _, q := range questions {
		db.Exec(insertSQL, q.subject, q.grade, q.question, q.options, q.correct, q.difficulty)
	}
	
	return nil
}
