package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"prmanager/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateTeam(ctx context.Context, name string) (models.Team, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(models.Team), args.Error(1)
}

func (m *MockRepository) GetTeamByID(ctx context.Context, id int) (models.Team, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.Team), args.Error(1)
}

func (m *MockRepository) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockRepository) GetUserByID(ctx context.Context, id int) (models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockRepository) ListActiveUsersInTeam(ctx context.Context, teamID int) ([]models.User, error) {
	args := m.Called(ctx, teamID)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockRepository) DeactivateUsersInTeam(ctx context.Context, teamID int, userIDs []int) error {
	args := m.Called(ctx, teamID, userIDs)
	return args.Error(0)
}

func (m *MockRepository) CreatePR(ctx context.Context, pr models.PR) (models.PR, error) {
	args := m.Called(ctx, pr)
	return args.Get(0).(models.PR), args.Error(1)
}

func (m *MockRepository) GetPRByID(ctx context.Context, id int) (models.PR, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.PR), args.Error(1)
}

func (m *MockRepository) SetPRStatus(ctx context.Context, id int, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRepository) AssignReviewers(ctx context.Context, prID int, userIDs []int) error {
	args := m.Called(ctx, prID, userIDs)
	return args.Error(0)
}

func (m *MockRepository) GetReviewersByPR(ctx context.Context, prID int) ([]models.User, error) {
	args := m.Called(ctx, prID)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockRepository) ReplaceReviewer(ctx context.Context, prID int, oldUserID int, newUserID int) error {
	args := m.Called(ctx, prID, oldUserID, newUserID)
	return args.Error(0)
}

func (m *MockRepository) ListPRsAssignedToUser(ctx context.Context, userID int) ([]models.PRWithReviewers, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.PRWithReviewers), args.Error(1)
}

func (m *MockRepository) CountAssignments(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateTeam(t *testing.T) {
	mockRepo := new(MockRepository)
	testLogger := createTestLogger()
	service := NewService(mockRepo, testLogger)

	tests := []struct {
		name        string
		teamName    string
		mockTeam    models.Team
		mockError   error
		expectError bool
	}{
		{
			name:     "success",
			teamName: "Test Team",
			mockTeam: models.Team{
				ID:   1,
				Name: "Test Team",
			},
			expectError: false,
		},
		{
			name:        "empty name",
			teamName:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expectError {
				mockRepo.On("CreateTeam", mock.Anything, tt.teamName).
					Return(tt.mockTeam, tt.mockError)
			}

			result, err := service.CreateTeam(context.Background(), tt.teamName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockTeam, result)
			}

			if !tt.expectError {
				mockRepo.AssertCalled(t, "CreateTeam", mock.Anything, tt.teamName)
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	mockRepo := new(MockRepository)
	testLogger := createTestLogger()
	service := NewService(mockRepo, testLogger)

	teamID := 1
	userName := "Test User"

	t.Run("success with team", func(t *testing.T) {
		mockTeam := models.Team{ID: teamID, Name: "Test Team"}
		mockUser := models.User{
			ID:       1,
			TeamID:   &teamID,
			Name:     userName,
			IsActive: true,
		}

		mockRepo.On("GetTeamByID", mock.Anything, teamID).Return(mockTeam, nil)
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("models.User")).
			Return(mockUser, nil)

		result, err := service.CreateUser(context.Background(), &teamID, userName, true)

		assert.NoError(t, err)
		assert.Equal(t, mockUser, result)
		mockRepo.AssertCalled(t, "GetTeamByID", mock.Anything, teamID)
		mockRepo.AssertCalled(t, "CreateUser", mock.Anything, mock.AnythingOfType("models.User"))
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := service.CreateUser(context.Background(), &teamID, "", true)
		assert.Error(t, err)
	})
}

func TestCreatePRSuccess(t *testing.T) {
	mockRepo := new(MockRepository)
	testLogger := createTestLogger()
	service := NewService(mockRepo, testLogger)

	authorID := 1
	teamID := 1
	title := "Test PR"

	t.Run("success with reviewers", func(t *testing.T) {
		author := models.User{
			ID:       authorID,
			Name:     "Author",
			TeamID:   &teamID,
			IsActive: true,
		}

		candidates := []models.User{
			{ID: 2, Name: "Reviewer 1", TeamID: &teamID, IsActive: true},
			{ID: 3, Name: "Reviewer 2", TeamID: &teamID, IsActive: true},
			{ID: 4, Name: "Reviewer 3", TeamID: &teamID, IsActive: true},
		}

		pr := models.PR{
			ID:       1,
			Title:    title,
			AuthorID: authorID,
			Status:   models.PRStatusOpen,
		}

		mockRepo.On("GetUserByID", mock.Anything, authorID).Return(author, nil)
		mockRepo.On("CreatePR", mock.Anything, mock.AnythingOfType("models.PR")).Return(pr, nil)
		mockRepo.On("ListActiveUsersInTeam", mock.Anything, teamID).Return(candidates, nil)
		mockRepo.On("AssignReviewers", mock.Anything, pr.ID, mock.MatchedBy(func(ids []int) bool {
			if len(ids) != 2 {
				return false
			}
			for _, id := range ids {
				if id == authorID {
					return false
				}
				if id < 2 || id > 4 {
					return false
				}
			}
			if ids[0] == ids[1] {
				return false
			}
			return true
		})).Return(nil)
		mockRepo.On("GetReviewersByPR", mock.Anything, pr.ID).Return(candidates[:2], nil)

		result, err := service.CreatePR(context.Background(), title, authorID)

		assert.NoError(t, err)
		assert.Equal(t, pr.ID, result.ID)
		assert.Len(t, result.Reviewers, 2)

		mockRepo.AssertCalled(t, "GetUserByID", mock.Anything, authorID)
		mockRepo.AssertCalled(t, "CreatePR", mock.Anything, mock.AnythingOfType("models.PR"))
		mockRepo.AssertCalled(t, "ListActiveUsersInTeam", mock.Anything, teamID)
		mockRepo.AssertCalled(t, "AssignReviewers", mock.Anything, pr.ID, mock.MatchedBy(func(ids []int) bool {
			return len(ids) == 2
		}))
		mockRepo.AssertCalled(t, "GetReviewersByPR", mock.Anything, pr.ID)
	})
}

func TestCreatePRNoTeam(t *testing.T) {
	mockRepo := new(MockRepository)
	testLogger := createTestLogger()
	service := NewService(mockRepo, testLogger)

	authorID := 1
	title := "Test PR"

	t.Run("author without team", func(t *testing.T) {
		author := models.User{
			ID:       authorID,
			Name:     "Author",
			TeamID:   nil,
			IsActive: true,
		}

		pr := models.PR{
			ID:       1,
			Title:    title,
			AuthorID: authorID,
			Status:   models.PRStatusOpen,
		}

		mockRepo.On("GetUserByID", mock.Anything, authorID).Return(author, nil)
		mockRepo.On("CreatePR", mock.Anything, mock.AnythingOfType("models.PR")).Return(pr, nil)

		result, err := service.CreatePR(context.Background(), title, authorID)

		assert.NoError(t, err)
		assert.Equal(t, pr.ID, result.ID)
		assert.Empty(t, result.Reviewers)

		mockRepo.AssertCalled(t, "GetUserByID", mock.Anything, authorID)
		mockRepo.AssertCalled(t, "CreatePR", mock.Anything, mock.AnythingOfType("models.PR"))
		mockRepo.AssertNotCalled(t, "ListActiveUsersInTeam")
		mockRepo.AssertNotCalled(t, "AssignReviewers")
		mockRepo.AssertNotCalled(t, "GetReviewersByPR")
	})
}

func TestCreatePRAuthorNotActive(t *testing.T) {
	mockRepo := new(MockRepository)
	testLogger := createTestLogger()
	service := NewService(mockRepo, testLogger)

	authorID := 1
	title := "Test PR"

	t.Run("author is not active", func(t *testing.T) {
		author := models.User{
			ID:       authorID,
			Name:     "Author",
			TeamID:   nil,
			IsActive: false,
		}

		mockRepo.On("GetUserByID", mock.Anything, authorID).Return(author, nil)

		_, err := service.CreatePR(context.Background(), title, authorID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "author is not active")

		mockRepo.AssertCalled(t, "GetUserByID", mock.Anything, authorID)
		mockRepo.AssertNotCalled(t, "CreatePR")
		mockRepo.AssertNotCalled(t, "ListActiveUsersInTeam")
		mockRepo.AssertNotCalled(t, "AssignReviewers")
		mockRepo.AssertNotCalled(t, "GetReviewersByPR")
	})
}

func TestCreatePRAuthorNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	testLogger := createTestLogger()
	service := NewService(mockRepo, testLogger)

	authorID := 1
	title := "Test PR"

	t.Run("author not found", func(t *testing.T) {
		mockRepo.On("GetUserByID", mock.Anything, authorID).Return(models.User{}, assert.AnError)

		_, err := service.CreatePR(context.Background(), title, authorID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "author not found")

		mockRepo.AssertCalled(t, "GetUserByID", mock.Anything, authorID)
		mockRepo.AssertNotCalled(t, "CreatePR")
		mockRepo.AssertNotCalled(t, "ListActiveUsersInTeam")
		mockRepo.AssertNotCalled(t, "AssignReviewers")
		mockRepo.AssertNotCalled(t, "GetReviewersByPR")
	})
}
