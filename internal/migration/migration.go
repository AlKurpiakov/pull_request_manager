package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	conn := pool

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS teams (
		 id SERIAL PRIMARY KEY,
		 name TEXT UNIQUE NOT NULL,
		 created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS users (
		 id SERIAL PRIMARY KEY,
		 team_id INT REFERENCES teams(id) ON DELETE SET NULL,
		 name TEXT NOT NULL,
		 is_active BOOLEAN NOT NULL DEFAULT true,
		 created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS prs (
		 id SERIAL PRIMARY KEY,
		 title TEXT NOT NULL,
		 author_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		 status TEXT NOT NULL DEFAULT 'OPEN',
		 created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS pr_reviewers (
		 pr_id INT NOT NULL REFERENCES prs(id) ON DELETE CASCADE,
		 user_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		 assigned_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		 PRIMARY KEY(pr_id, user_id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id)`,
		`CREATE INDEX IF NOT EXISTS idx_prs_author_id ON prs(author_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pr_reviewers(user_id)`,
	}

	for i, s := range stmts {
		if _, err := conn.Exec(ctx, s); err != nil {
			return fmt.Errorf("migrations stmt %d failed: %w", i, err)
		}
	}
	return nil
}
