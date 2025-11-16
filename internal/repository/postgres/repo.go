package postgres

import (
	"context"
	"fmt"

	"prmanager/internal/models"
	"prmanager/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) repository.Repository {
	return &repo{pool: pool}
}

func (r *repo) CreateTeam(ctx context.Context, name string) (models.Team, error) {
	var t models.Team
	row := r.pool.QueryRow(ctx, `INSERT INTO teams(name) VALUES($1) RETURNING id, name, created_at`, name)
	if err := row.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
		return t, fmt.Errorf("create team: %w", err)
	}
	return t, nil
}

func (r *repo) GetTeamByID(ctx context.Context, id int) (models.Team, error) {
	var t models.Team
	row := r.pool.QueryRow(ctx, `SELECT id, name, created_at FROM teams WHERE id=$1`, id)
	if err := row.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
		return t, fmt.Errorf("get team: %w", err)
	}
	return t, nil
}

func (r *repo) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	var res models.User
	row := r.pool.QueryRow(ctx, `INSERT INTO users(team_id, name, is_active) VALUES($1,$2,$3) RETURNING id, team_id, name, is_active, created_at`, u.TeamID, u.Name, u.IsActive)
	if err := row.Scan(&res.ID, &res.TeamID, &res.Name, &res.IsActive, &res.CreatedAt); err != nil {
		return res, fmt.Errorf("create user: %w", err)
	}
	return res, nil
}

func (r *repo) GetUserByID(ctx context.Context, id int) (models.User, error) {
	var u models.User
	row := r.pool.QueryRow(ctx, `SELECT id, team_id, name, is_active, created_at FROM users WHERE id=$1`, id)
	if err := row.Scan(&u.ID, &u.TeamID, &u.Name, &u.IsActive, &u.CreatedAt); err != nil {
		return u, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *repo) ListActiveUsersInTeam(ctx context.Context, teamID int) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, team_id, name, is_active, created_at FROM users WHERE team_id=$1 AND is_active=true`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list active users: %w", err)
	}
	defer rows.Close()

	res := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.TeamID, &u.Name, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		res = append(res, u)
	}
	return res, nil
}

func (r *repo) DeactivateUsersInTeam(ctx context.Context, teamID int, userIDs []int) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET is_active = false WHERE team_id = $1 AND id = ANY($2)`, teamID, userIDs)
	if err != nil {
		return fmt.Errorf("deactivate users: %w", err)
	}
	return nil
}

func (r *repo) CreatePR(ctx context.Context, pr models.PR) (models.PR, error) {
	var res models.PR
	row := r.pool.QueryRow(ctx, `INSERT INTO prs(title, author_id, status) VALUES($1,$2,$3) RETURNING id, title, author_id, status, created_at`, pr.Title, pr.AuthorID, pr.Status)
	if err := row.Scan(&res.ID, &res.Title, &res.AuthorID, &res.Status, &res.CreatedAt); err != nil {
		return res, fmt.Errorf("create PR: %w", err)
	}
	return res, nil
}

func (r *repo) GetPRByID(ctx context.Context, id int) (models.PR, error) {
	var p models.PR
	row := r.pool.QueryRow(ctx, `SELECT id, title, author_id, status, created_at FROM prs WHERE id=$1`, id)
	if err := row.Scan(&p.ID, &p.Title, &p.AuthorID, &p.Status, &p.CreatedAt); err != nil {
		return p, fmt.Errorf("get PR: %w", err)
	}
	return p, nil
}

func (r *repo) SetPRStatus(ctx context.Context, id int, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE prs SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return fmt.Errorf("set PR status: %w", err)
	}
	return nil
}

func (r *repo) AssignReviewers(ctx context.Context, prID int, userIDs []int) error {
	for _, uid := range userIDs {
		if _, err := r.pool.Exec(ctx, `INSERT INTO pr_reviewers(pr_id, user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, prID, uid); err != nil {
			return fmt.Errorf("assign reviewer %d: %w", uid, err)
		}
	}
	return nil
}

func (r *repo) GetReviewersByPR(ctx context.Context, prID int) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT u.id, u.team_id, u.name, u.is_active, u.created_at FROM users u JOIN pr_reviewers r ON r.user_id = u.id WHERE r.pr_id=$1`, prID)
	if err != nil {
		return nil, fmt.Errorf("get reviewers by PR: %w", err)
	}
	defer rows.Close()

	res := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.TeamID, &u.Name, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan reviewer: %w", err)
		}
		res = append(res, u)
	}
	return res, nil
}

func (r *repo) ReplaceReviewer(ctx context.Context, prID int, oldUserID int, newUserID int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM pr_reviewers WHERE pr_id=$1 AND user_id=$2`, prID, oldUserID); err != nil {
		return fmt.Errorf("delete old reviewer: %w", err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO pr_reviewers(pr_id, user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, prID, newUserID); err != nil {
		return fmt.Errorf("insert new reviewer: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *repo) ListPRsAssignedToUser(ctx context.Context, userID int) ([]models.PRWithReviewers, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.id, p.title, p.author_id, p.status, p.created_at FROM prs p JOIN pr_reviewers r ON r.pr_id = p.id WHERE r.user_id=$1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list PRs assigned to user: %w", err)
	}
	defer rows.Close()

	out := make([]models.PRWithReviewers, 0)
	for rows.Next() {
		var p models.PR
		if err := rows.Scan(&p.ID, &p.Title, &p.AuthorID, &p.Status, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan PR: %w", err)
		}
		revs, err := r.GetReviewersByPR(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("get reviewers for PR %d: %w", p.ID, err)
		}
		out = append(out, models.PRWithReviewers{PR: p, Reviewers: revs})
	}
	return out, nil
}

func (r *repo) CountAssignments(ctx context.Context) (int, error) {
	row := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pr_reviewers`)
	var c int
	if err := row.Scan(&c); err != nil {
		return 0, fmt.Errorf("count assignments: %w", err)
	}
	return c, nil
}
