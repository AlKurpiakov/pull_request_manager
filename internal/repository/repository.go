package repository

import (
	"context"

	"prmanager/internal/models"
)

type Repository interface {
	CreateTeam(ctx context.Context, name string) (models.Team, error)
	GetTeamByID(ctx context.Context, id int) (models.Team, error)

	CreateUser(ctx context.Context, u models.User) (models.User, error)
	GetUserByID(ctx context.Context, id int) (models.User, error)
	ListActiveUsersInTeam(ctx context.Context, teamID int) ([]models.User, error)
	DeactivateUsersInTeam(ctx context.Context, teamID int, userIDs []int) error

	CreatePR(ctx context.Context, pr models.PR) (models.PR, error)
	GetPRByID(ctx context.Context, id int) (models.PR, error)
	SetPRStatus(ctx context.Context, id int, status string) error

	AssignReviewers(ctx context.Context, prID int, userIDs []int) error
	GetReviewersByPR(ctx context.Context, prID int) ([]models.User, error)
	ReplaceReviewer(ctx context.Context, prID int, oldUserID int, newUserID int) error

	ListPRsAssignedToUser(ctx context.Context, userID int) ([]models.PRWithReviewers, error)
	CountAssignments(ctx context.Context) (int, error)
}
