package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"prmanager/internal/models"
	"prmanager/internal/repository"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrBadRequest  = errors.New("bad request")
	ErrPRMerged    = errors.New("cannot reassign on merged PR")
	ErrNoCandidate = errors.New("no active replacement candidate in team")
)

type Service struct {
	repo   repository.Repository
	rand   *rand.Rand
	logger *slog.Logger
}

func NewService(r repository.Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:   r,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		logger: logger,
	}
}

func (s *Service) CreateTeam(ctx context.Context, name string) (models.Team, error) {
	s.logger.Info("creating team", "name", name)

	if name == "" {
		return models.Team{}, fmt.Errorf("%w: team name empty", ErrBadRequest)
	}

	t, err := s.repo.CreateTeam(ctx, name)
	if err != nil {
		s.logger.Error("failed to create team", "error", err, "name", name)
		return models.Team{}, err
	}

	s.logger.Info("team created successfully", "team_id", t.ID, "name", t.Name)
	return t, nil
}

func (s *Service) CreateUser(ctx context.Context, teamID *int, name string, isActive bool) (models.User, error) {
	s.logger.Info("creating user", "name", name, "team_id", teamID, "is_active", isActive)

	if name == "" {
		return models.User{}, fmt.Errorf("%w: user name empty", ErrBadRequest)
	}

	if teamID != nil {
		if _, err := s.repo.GetTeamByID(ctx, *teamID); err != nil {
			s.logger.Warn("team not found for user creation", "team_id", *teamID)
			return models.User{}, fmt.Errorf("%w: team not found", ErrBadRequest)
		}
	}

	u := models.User{TeamID: teamID, Name: name, IsActive: isActive}
	user, err := s.repo.CreateUser(ctx, u)
	if err != nil {
		s.logger.Error("failed to create user", "error", err, "name", name, "team_id", teamID)
		return models.User{}, err
	}

	s.logger.Info("user created successfully", "user_id", user.ID, "name", user.Name)
	return user, nil
}

func (s *Service) CreatePR(ctx context.Context, title string, authorID int) (models.PRWithReviewers, error) {
	s.logger.Info("creating PR", "title", title, "author_id", authorID)

	author, err := s.repo.GetUserByID(ctx, authorID)
	if err != nil {
		s.logger.Warn("author not found", "author_id", authorID, "error", err)
		return models.PRWithReviewers{}, fmt.Errorf("%w: author not found", ErrBadRequest)
	}

	if !author.IsActive {
		s.logger.Warn("author is not active", "author_id", authorID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: author is not active", ErrBadRequest)
	}

	pr, err := s.repo.CreatePR(ctx, models.PR{
		Title:    title,
		AuthorID: authorID,
		Status:   models.PRStatusOpen,
	})
	if err != nil {
		s.logger.Error("failed to create PR", "error", err, "title", title, "author_id", authorID)
		return models.PRWithReviewers{}, err
	}

	if author.TeamID == nil {
		s.logger.Info("PR created without reviewers (author has no team)", "pr_id", pr.ID)
		return models.PRWithReviewers{PR: pr, Reviewers: []models.User{}}, nil
	}

	candidates, err := s.repo.ListActiveUsersInTeam(ctx, *author.TeamID)
	if err != nil {
		s.logger.Error("failed to get team members", "error", err, "team_id", *author.TeamID)
		return models.PRWithReviewers{}, err
	}

	filtered := make([]models.User, 0, len(candidates))
	for _, u := range candidates {
		if u.ID != authorID {
			filtered = append(filtered, u)
		}
	}

	count := 2
	if len(filtered) < count {
		count = len(filtered)
	}

	chosenIDs := []int{}
	if count > 0 {
		idxs := s.randomSample(len(filtered), count)
		for _, i := range idxs {
			chosenIDs = append(chosenIDs, filtered[i].ID)
		}

		if err := s.repo.AssignReviewers(ctx, pr.ID, chosenIDs); err != nil {
			s.logger.Error("failed to assign reviewers", "error", err, "pr_id", pr.ID, "reviewer_ids", chosenIDs)
			return models.PRWithReviewers{}, err
		}
	}

	revs, err := s.repo.GetReviewersByPR(ctx, pr.ID)
	if err != nil {
		s.logger.Error("failed to get assigned reviewers", "error", err, "pr_id", pr.ID)
		return models.PRWithReviewers{}, err
	}

	s.logger.Info("PR created successfully",
		"pr_id", pr.ID,
		"reviewers_count", len(revs),
		"reviewer_ids", chosenIDs)

	return models.PRWithReviewers{PR: pr, Reviewers: revs}, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID int, oldUserID int) (models.PRWithReviewers, error) {
	s.logger.Info("reassigning reviewer", "pr_id", prID, "old_user_id", oldUserID)

	pr, err := s.repo.GetPRByID(ctx, prID)
	if err != nil {
		s.logger.Warn("PR not found for reassignment", "pr_id", prID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: pr not found", ErrBadRequest)
	}

	if pr.Status == models.PRStatusMerged {
		s.logger.Warn("attempt to reassign reviewer on merged PR", "pr_id", prID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: cannot reassign merged pr", ErrPRMerged)
	}

	oldUser, err := s.repo.GetUserByID(ctx, oldUserID)
	if err != nil {
		s.logger.Warn("old reviewer not found", "user_id", oldUserID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: user not found", ErrBadRequest)
	}

	if oldUser.TeamID == nil {
		s.logger.Warn("reviewer has no team", "user_id", oldUserID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: reviewer has no team", ErrBadRequest)
	}

	currentReviewers, err := s.repo.GetReviewersByPR(ctx, prID)
	if err != nil {
		s.logger.Error("failed to get current reviewers", "error", err, "pr_id", prID)
		return models.PRWithReviewers{}, err
	}

	found := false
	for _, reviewer := range currentReviewers {
		if reviewer.ID == oldUserID {
			found = true
			break
		}
	}
	if !found {
		s.logger.Warn("old reviewer not assigned to PR", "pr_id", prID, "user_id", oldUserID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: reviewer is not assigned to this PR", ErrBadRequest)
	}

	candidates, err := s.repo.ListActiveUsersInTeam(ctx, *oldUser.TeamID)
	if err != nil {
		s.logger.Error("failed to get team candidates", "error", err, "team_id", *oldUser.TeamID)
		return models.PRWithReviewers{}, err
	}

	filtered := make([]models.User, 0)
	for _, u := range candidates {
		if u.ID != pr.AuthorID && u.ID != oldUserID {
			alreadyAssigned := false
			for _, reviewer := range currentReviewers {
				if reviewer.ID == u.ID {
					alreadyAssigned = true
					break
				}
			}
			if !alreadyAssigned {
				filtered = append(filtered, u)
			}
		}
	}

	if len(filtered) == 0 {
		s.logger.Warn("no available candidates for reassignment",
			"pr_id", prID, "old_user_id", oldUserID, "team_id", *oldUser.TeamID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: no active candidates to reassign", ErrNoCandidate)
	}

	newIdx := s.rand.Intn(len(filtered))
	newUser := filtered[newIdx]

	if err := s.repo.ReplaceReviewer(ctx, prID, oldUserID, newUser.ID); err != nil {
		s.logger.Error("failed to replace reviewer", "error", err, "pr_id", prID, "old_user", oldUserID, "new_user", newUser.ID)
		return models.PRWithReviewers{}, err
	}

	revs, err := s.repo.GetReviewersByPR(ctx, prID)
	if err != nil {
		s.logger.Error("failed to get updated reviewers", "error", err, "pr_id", prID)
		return models.PRWithReviewers{}, err
	}

	s.logger.Info("reviewer reassigned successfully",
		"pr_id", prID,
		"old_user_id", oldUserID,
		"new_user_id", newUser.ID,
		"new_user_name", newUser.Name)

	return models.PRWithReviewers{PR: pr, Reviewers: revs}, nil
}

func (s *Service) MergePR(ctx context.Context, prID int) (models.PRWithReviewers, error) {
	s.logger.Info("merging PR", "pr_id", prID)

	pr, err := s.repo.GetPRByID(ctx, prID)
	if err != nil {
		s.logger.Warn("PR not found for merge", "pr_id", prID)
		return models.PRWithReviewers{}, fmt.Errorf("%w: pr not found", ErrBadRequest)
	}

	if pr.Status == models.PRStatusMerged {
		s.logger.Info("PR already merged", "pr_id", prID)
		revs, _ := s.repo.GetReviewersByPR(ctx, prID)
		return models.PRWithReviewers{PR: pr, Reviewers: revs}, nil
	}

	if pr.Status != models.PRStatusOpen {
		return models.PRWithReviewers{}, fmt.Errorf("%w: can only merge OPEN pull requests", ErrBadRequest)
	}

	if err := s.repo.SetPRStatus(ctx, prID, string(models.PRStatusMerged)); err != nil {
		s.logger.Error("failed to set PR status to merged", "error", err, "pr_id", prID)
		return models.PRWithReviewers{}, err
	}

	pr.Status = models.PRStatusMerged
	revs, err := s.repo.GetReviewersByPR(ctx, prID)
	if err != nil {
		s.logger.Error("failed to get reviewers after merge", "error", err, "pr_id", prID)
		return models.PRWithReviewers{}, err
	}

	s.logger.Info("PR merged successfully", "pr_id", prID)
	return models.PRWithReviewers{PR: pr, Reviewers: revs}, nil
}

func (s *Service) ListPRsAssignedToUser(ctx context.Context, userID int) ([]models.PRWithReviewers, error) {
	s.logger.Debug("listing PRs assigned to user", "user_id", userID)

	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		s.logger.Warn("user not found for PRs query", "user_id", userID)
		return nil, fmt.Errorf("%w: user not found", ErrBadRequest)
	}

	prs, err := s.repo.ListPRsAssignedToUser(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list PRs for user", "error", err, "user_id", userID)
		return nil, err
	}

	s.logger.Debug("retrieved PRs for user", "user_id", userID, "prs_count", len(prs))
	return prs, nil
}

func (s *Service) StatsAssignments(ctx context.Context) (int, error) {
	count, err := s.repo.CountAssignments(ctx)
	if err != nil {
		s.logger.Error("failed to count assignments", "error", err)
		return 0, err
	}
	return count, nil
}

func (s *Service) randomSample(n, k int) []int {
	if n <= k {
		// Return all indices if n <= k
		res := make([]int, n)
		for i := 0; i < n; i++ {
			res[i] = i
		}
		return res
	}

	res := make([]int, n)
	for i := 0; i < n; i++ {
		res[i] = i
	}
	for i := 0; i < k; i++ {
		r := i + s.rand.Intn(n-i)
		res[i], res[r] = res[r], res[i]
	}
	return res[:k]
}
