package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	// Render provides DATABASE_URL directly
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fall back to individual env vars for local Docker
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_USER", "laesulia"),
			getEnv("DB_PASSWORD", "laesulia"),
			getEnv("DB_NAME", "laesulia"),
			getEnv("DB_SSLMODE", "disable"),
		)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot reach database: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS postgis;`,
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,

		`CREATE TABLE IF NOT EXISTS users (
			id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			username   TEXT UNIQUE NOT NULL,
			email      TEXT UNIQUE NOT NULL,
			password   TEXT NOT NULL,
			reputation INT DEFAULT 0,
			is_admin   BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS locations (
			id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name         TEXT NOT NULL,
			local_name   TEXT,
			description  TEXT,
			category     TEXT NOT NULL DEFAULT 'place',
			geom         GEOMETRY(POINT, 4326) NOT NULL,
			address_code TEXT UNIQUE,
			created_by   UUID REFERENCES users(id),
			upvotes      INT DEFAULT 0,
			verified     BOOLEAN DEFAULT FALSE,
			created_at   TIMESTAMPTZ DEFAULT NOW(),
			updated_at   TIMESTAMPTZ DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS locations_geom_idx ON locations USING GIST(geom);`,

		`CREATE TABLE IF NOT EXISTS location_upvotes (
			user_id     UUID REFERENCES users(id),
			location_id UUID REFERENCES locations(id),
			PRIMARY KEY (user_id, location_id)
		);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func MigrateBounty(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS bounty_jobs (
			id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			title        TEXT NOT NULL,
			description  TEXT,
			lat          DOUBLE PRECISION NOT NULL,
			lng          DOUBLE PRECISION NOT NULL,
			geom         GEOMETRY(POINT, 4326),
			reward_sbd   NUMERIC(10,2) NOT NULL DEFAULT 5.00,
			submit_type  TEXT NOT NULL DEFAULT 'photos',
			status       TEXT NOT NULL DEFAULT 'open',
			created_by   UUID REFERENCES users(id),
			claimed_by   UUID REFERENCES users(id),
			claimed_at   TIMESTAMPTZ,
			submitted_at TIMESTAMPTZ,
			approved_at  TIMESTAMPTZ,
			approved_by  UUID REFERENCES users(id),
			created_at   TIMESTAMPTZ DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS bounty_jobs_geom_idx ON bounty_jobs USING GIST(geom);`,
		`CREATE INDEX IF NOT EXISTS bounty_jobs_status_idx ON bounty_jobs(status);`,

		`CREATE TABLE IF NOT EXISTS bounty_submissions (
			id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			job_id     UUID NOT NULL REFERENCES bounty_jobs(id) ON DELETE CASCADE,
			user_id    UUID NOT NULL REFERENCES users(id),
			file_url   TEXT NOT NULL,
			file_type  TEXT NOT NULL DEFAULT 'photo',
			file_size  BIGINT,
			lat        DOUBLE PRECISION,
			lng        DOUBLE PRECISION,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS wallets (
			id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id         UUID UNIQUE NOT NULL REFERENCES users(id),
			balance_sbd     NUMERIC(10,2) DEFAULT 0.00,
			total_earned    NUMERIC(10,2) DEFAULT 0.00,
			total_withdrawn NUMERIC(10,2) DEFAULT 0.00,
			updated_at      TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS wallet_transactions (
			id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			wallet_id   UUID NOT NULL REFERENCES wallets(id),
			job_id      UUID REFERENCES bounty_jobs(id),
			amount_sbd  NUMERIC(10,2) NOT NULL,
			type        TEXT NOT NULL,
			note        TEXT,
			created_at  TIMESTAMPTZ DEFAULT NOW()
		);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("bounty migration failed: %w", err)
		}
	}
	return nil
}

func MigrateBase64(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE bounty_submissions ALTER COLUMN file_url TYPE TEXT`)
	if err != nil {
		return nil // ignore if already text
	}
	return nil
}
func MigrateAdmin(db *sql.DB) error {
	// Add is_admin column if not exists
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false`)
	_, err := db.Exec(`UPDATE users SET is_admin=true WHERE email='gordon@laesulia.app'`)
	return err
}

func MigratePhone(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone TEXT DEFAULT ''`)
	if err != nil {
		log.Printf("MigratePhone error: %v", err)
	}
	log.Printf("MigratePhone completed")
	return nil
}

