package api

import (
	"context"

	"prmanager/internal/models"
)

type ServiceInterface interface {
	CreateTeam(ctx context.Context, name string) (models.Team, error)
	CreateUser(ctx context.Context, teamID *int, name string, isActive bool) (models.User, error)
	CreatePR(ctx context.Context, title string, authorID int) (models.PRWithReviewers, error)
	ReassignReviewer(ctx context.Context, prID int, oldUserID int) (models.PRWithReviewers, error)
	MergePR(ctx context.Context, prID int) (models.PRWithReviewers, error)
	ListPRsAssignedToUser(ctx context.Context, userID int) ([]models.PRWithReviewers, error)
	StatsAssignments(ctx context.Context) (int, error)
}
