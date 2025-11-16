package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"prmanager/internal/api"
	"prmanager/internal/config"
	"prmanager/internal/migration"
	"prmanager/internal/repository/postgres"
	"prmanager/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	cfg := config.LoadFromEnv()

	testDBConn := cfg.DBConn + "_test"
	pool, err := pgxpool.New(context.Background(), testDBConn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := migration.Run(context.Background(), pool); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	_, err = pool.Exec(context.Background(), "DELETE FROM pr_reviewers")
	if err != nil {
		t.Fatalf("Failed to clean pr_reviewers: %v", err)
	}
	_, err = pool.Exec(context.Background(), "DELETE FROM prs")
	if err != nil {
		t.Fatalf("Failed to clean prs: %v", err)
	}
	_, err = pool.Exec(context.Background(), "DELETE FROM users")
	if err != nil {
		t.Fatalf("Failed to clean users: %v", err)
	}
	_, err = pool.Exec(context.Background(), "DELETE FROM teams")
	if err != nil {
		t.Fatalf("Failed to clean teams: %v", err)
	}

	return pool
}

func TestIntegrationCreateTeamAndUser(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := postgres.NewRepo(pool)
	svc := service.NewService(repo, nil)
	handler := api.NewHandler(svc, nil)

	teamBody := map[string]interface{}{
		"name": "Integration Test Team",
	}
	teamJSON, _ := json.Marshal(teamBody)

	req := httptest.NewRequest("POST", "/teams", bytes.NewReader(teamJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Router().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var teamResp struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(rr.Body).Decode(&teamResp)

	userBody := map[string]interface{}{
		"name": "Integration Test User",
	}
	userJSON, _ := json.Marshal(userBody)

	userReq := httptest.NewRequest("POST", fmt.Sprintf("/teams/%d/users", teamResp.ID), bytes.NewReader(userJSON))
	userReq.Header.Set("Content-Type", "application/json")
	userRr := httptest.NewRecorder()

	handler.Router().ServeHTTP(userRr, userReq)
	assert.Equal(t, http.StatusCreated, userRr.Code)

	var userResp struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		TeamID *int   `json:"team_id"`
	}
	json.NewDecoder(userRr.Body).Decode(&userResp)

	assert.Equal(t, "Integration Test User", userResp.Name)
	assert.Equal(t, teamResp.ID, *userResp.TeamID)
}

func TestIntegrationCreatePR(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := postgres.NewRepo(pool)
	svc := service.NewService(repo, nil)
	handler := api.NewHandler(svc, nil)

	teamBody := map[string]interface{}{"name": "PR Test Team"}
	teamJSON, _ := json.Marshal(teamBody)
	teamReq := httptest.NewRequest("POST", "/teams", bytes.NewReader(teamJSON))
	teamReq.Header.Set("Content-Type", "application/json")
	teamRr := httptest.NewRecorder()
	handler.Router().ServeHTTP(teamRr, teamReq)

	var teamResp struct{ ID int }
	json.NewDecoder(teamRr.Body).Decode(&teamResp)

	authorBody := map[string]interface{}{"name": "PR Author"}
	authorJSON, _ := json.Marshal(authorBody)
	authorReq := httptest.NewRequest("POST", fmt.Sprintf("/teams/%d/users", teamResp.ID), bytes.NewReader(authorJSON))
	authorReq.Header.Set("Content-Type", "application/json")
	authorRr := httptest.NewRecorder()
	handler.Router().ServeHTTP(authorRr, authorReq)

	var authorResp struct{ ID int }
	json.NewDecoder(authorRr.Body).Decode(&authorResp)

	for i := 1; i <= 3; i++ {
		reviewerBody := map[string]interface{}{"name": fmt.Sprintf("Reviewer %d", i)}
		reviewerJSON, _ := json.Marshal(reviewerBody)
		reviewerReq := httptest.NewRequest("POST", fmt.Sprintf("/teams/%d/users", teamResp.ID), bytes.NewReader(reviewerJSON))
		reviewerReq.Header.Set("Content-Type", "application/json")
		reviewerRr := httptest.NewRecorder()
		handler.Router().ServeHTTP(reviewerRr, reviewerReq)
	}

	prBody := map[string]interface{}{
		"title":     "Integration Test PR",
		"author_id": authorResp.ID,
	}
	prJSON, _ := json.Marshal(prBody)
	prReq := httptest.NewRequest("POST", "/prs", bytes.NewReader(prJSON))
	prReq.Header.Set("Content-Type", "application/json")
	prRr := httptest.NewRecorder()

	handler.Router().ServeHTTP(prRr, prReq)
	assert.Equal(t, http.StatusCreated, prRr.Code)

	var prResp struct {
		ID        int                `json:"id"`
		Title     string             `json:"title"`
		Reviewers []struct{ ID int } `json:"reviewers"`
	}
	json.NewDecoder(prRr.Body).Decode(&prResp)

	assert.Equal(t, "Integration Test PR", prResp.Title)
	assert.Len(t, prResp.Reviewers, 2)
}
