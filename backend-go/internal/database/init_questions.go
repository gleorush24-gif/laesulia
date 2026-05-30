package database

import (
	"database/sql"
)

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
	
	insertSQL := `
	INSERT INTO si_questions (subject, grade_level, question, options, correct_answer, difficulty)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT DO NOTHING;
	`
	

	questions := [][]interface{}{
		{"History", "Year 6", "What year did Solomon Islands gain independence?", `["1975", "1978", "1980", "1985"]`, "1978", "easy"},
		{"History", "Year 7", "Who was the first Prime Minister of Solomon Islands?", `["Peter Kenilorea", "Manasseh Sogavare", "Bartholomew Ulufa'u", "Francis Billy Hilly"]`, "Peter Kenilorea", "medium"},
		{"Science", "Year 6", "What is the capital of Solomon Islands?", `["Honiara", "Gizo", "Auki", "Buala"]`, "Honiara", "easy"},
		{"Math", "Year 6", "What is 4 × 4?", `["12", "16", "20", "24"]`, "16", "easy"},
		{"Math", "Year 7", "What is 5² + 3²?", `["8", "25", "34", "64"]`, "34", "medium"},
		{"Science", "Year 7", "Which gas do plants absorb from atmosphere?", `["Oxygen", "Nitrogen", "Carbon Dioxide", "Hydrogen"]`, "Carbon Dioxide", "easy"},
		{"Social Studies", "Year 8", "Main language spoken in Solomon Islands?", `["English", "Pijin", "Spanish", "French"]`, "Pijin", "easy"},
		{"History", "Year 6", "Solomon Islands is located in which ocean?", `["Atlantic", "Indian", "Pacific", "Arctic"]`, "Pacific", "easy"},
		{"Geography", "Year 7", "How many provinces does Solomon Islands have?", `["8", "10", "12", "15"]`, "10", "medium"},
		{"Science", "Year 6", "What type of climate does Solomon Islands have?", `["Tropical", "Temperate", "Desert", "Arctic"]`, "Tropical", "easy"},
		{"Math", "Year 8", "What is 12 ÷ 3 + 5?", `["4", "6", "9", "12"]`, "9", "easy"},
		{"History", "Year 7", "In what year did WWII end?", `["1943", "1944", "1945", "1946"]`, "1945", "medium"},
		{"Science", "Year 8", "What is the process plants use to make food?", `["Respiration", "Photosynthesis", "Digestion", "Fermentation"]`, "Photosynthesis", "medium"},
		{"Social Studies", "Year 6", "What currency is used in Solomon Islands?", `["Dollar", "Pound", "Peso", "Rupee"]`, "Dollar", "easy"},
		{"Geography", "Year 8", "Which ocean surrounds Solomon Islands?", `["Atlantic", "Pacific", "Indian", "Arctic"]`, "Pacific", "easy"},
		{"Math", "Year 6", "What is 10 + 5 × 2?", `["30", "20", "15", "25"]`, "20", "medium"},
		{"History", "Year 8", "What was a major event in SI during WWII?", `["Pearl Harbor", "Guadalcanal Campaign", "Berlin Wall", "D-Day"]`, "Guadalcanal Campaign", "hard"},
		{"Science", "Year 7", "Which of these is a renewable resource?", `["Coal", "Oil", "Solar Energy", "Natural Gas"]`, "Solar Energy", "medium"},
		{"Geography", "Year 6", "Honiara is the capital on which island?", `["Guadalcanal", "Malaita", "New Georgia", "Santa Isabel"]`, "Guadalcanal", "easy"},
		{"Computers", "Year 7", "What does CPU stand for?", `["Central Process Unit", "Central Processing Unit", "Computer Personal Unit", "Central Processor Union"]`, "Central Processing Unit", "medium"},
	}

	
	for _, q := range questions {
		db.Exec(insertSQL, q...)
	}
	
	return nil
}
