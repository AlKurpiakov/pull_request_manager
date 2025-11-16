package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type Handler struct {
	svc    ServiceInterface
	r      *chi.Mux
	logger *slog.Logger
}

func NewHandler(s ServiceInterface, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &Handler{svc: s, r: chi.NewRouter(), logger: logger}
	h.routes()
	return h
}

func (h *Handler) Router() http.Handler { return h.r }

func (h *Handler) routes() {
	h.r.Post("/teams", h.createTeam)
	h.r.Post("/teams/{team_id}/users", h.createUser)
	h.r.Post("/prs", h.createPR)
	h.r.Post("/prs/{pr_id}/reassign", h.reassign)
	h.r.Post("/prs/{pr_id}/merge", h.merge)
	h.r.Get("/users/{user_id}/prs", h.listPRsForUser)
	h.r.Get("/stats", h.stats)
}

func (h *Handler) writeJSON(w http.ResponseWriter, v interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResp := ErrorResponse{}
	errorResp.Error.Code = code
	errorResp.Error.Message = message
	json.NewEncoder(w).Encode(errorResp)
}

func (h *Handler) createTeam(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("createTeam request")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn("invalid JSON in createTeam request", "error", err)
		h.writeError(w, "BAD_REQUEST", "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Name == "" {
		h.writeError(w, "BAD_REQUEST", "name is required", http.StatusBadRequest)
		return
	}

	t, err := h.svc.CreateTeam(r.Context(), body.Name)
	if err != nil {
		h.logger.Error("failed to create team", "error", err, "name", body.Name)
		h.writeError(w, "INTERNAL_ERROR", "failed to create team", http.StatusInternalServerError)
		return
	}

	h.logger.Info("team created successfully", "team_id", t.ID, "name", t.Name)
	h.writeJSON(w, t, http.StatusCreated)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("createUser request")

	teamIDStr := chi.URLParam(r, "team_id")
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil || teamID <= 0 {
		h.writeError(w, "BAD_REQUEST", "valid team_id is required", http.StatusBadRequest)
		return
	}

	var body struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn("invalid JSON in createUser request", "error", err)
		h.writeError(w, "BAD_REQUEST", "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Name == "" {
		h.writeError(w, "BAD_REQUEST", "name is required", http.StatusBadRequest)
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	u, err := h.svc.CreateUser(r.Context(), &teamID, body.Name, isActive)
	if err != nil {
		h.logger.Error("failed to create user", "error", err, "team_id", teamID, "name", body.Name)
		h.writeError(w, "BAD_REQUEST", err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Info("user created successfully", "user_id", u.ID, "name", u.Name, "team_id", teamID)
	h.writeJSON(w, u, http.StatusCreated)
}

func (h *Handler) createPR(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("createPR request")

	var body struct {
		Title    string `json:"title"`
		AuthorID int    `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn("invalid JSON in createPR request", "error", err)
		h.writeError(w, "BAD_REQUEST", "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Title == "" {
		h.writeError(w, "BAD_REQUEST", "title is required", http.StatusBadRequest)
		return
	}

	if body.AuthorID <= 0 {
		h.writeError(w, "BAD_REQUEST", "valid author_id is required", http.StatusBadRequest)
		return
	}

	pr, err := h.svc.CreatePR(r.Context(), body.Title, body.AuthorID)
	if err != nil {
		h.logger.Error("failed to create PR", "error", err, "title", body.Title, "author_id", body.AuthorID)
		switch err.Error() {
		case "bad request: author not found", "bad request: author is not active":
			h.writeError(w, "NOT_FOUND", err.Error(), http.StatusNotFound)
		default:
			h.writeError(w, "BAD_REQUEST", err.Error(), http.StatusBadRequest)
		}
		return
	}

	h.logger.Info("PR created successfully", "pr_id", pr.ID, "reviewers_count", len(pr.Reviewers))
	h.writeJSON(w, pr, http.StatusCreated)
}

func (h *Handler) reassign(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("reassign reviewer request")

	prIDStr := chi.URLParam(r, "pr_id")
	prID, err := strconv.Atoi(prIDStr)
	if err != nil || prID <= 0 {
		h.writeError(w, "BAD_REQUEST", "valid pr_id is required", http.StatusBadRequest)
		return
	}

	var body struct {
		OldUserID int `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn("invalid JSON in reassign request", "error", err)
		h.writeError(w, "BAD_REQUEST", "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.OldUserID <= 0 {
		h.writeError(w, "BAD_REQUEST", "valid old_user_id is required", http.StatusBadRequest)
		return
	}

	res, err := h.svc.ReassignReviewer(r.Context(), prID, body.OldUserID)
	if err != nil {
		h.logger.Error("failed to reassign reviewer", "error", err, "pr_id", prID, "old_user_id", body.OldUserID)
		switch err.Error() {
		case "bad request: cannot reassign merged pr":
			h.writeError(w, "PR_MERGED", err.Error(), http.StatusConflict)
		case "bad request: no active candidates to reassign":
			h.writeError(w, "NO_CANDIDATE", err.Error(), http.StatusConflict)
		case "bad request: reviewer is not assigned to this PR":
			h.writeError(w, "NOT_ASSIGNED", err.Error(), http.StatusConflict)
		case "bad request: pr not found", "bad request: user not found":
			h.writeError(w, "NOT_FOUND", err.Error(), http.StatusNotFound)
		default:
			h.writeError(w, "BAD_REQUEST", err.Error(), http.StatusBadRequest)
		}
		return
	}

	h.logger.Info("reviewer reassigned successfully", "pr_id", prID, "old_user_id", body.OldUserID)
	h.writeJSON(w, res, http.StatusOK)
}

func (h *Handler) merge(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("merge PR request")

	prIDStr := chi.URLParam(r, "pr_id")
	prID, err := strconv.Atoi(prIDStr)
	if err != nil || prID <= 0 {
		h.writeError(w, "BAD_REQUEST", "valid pr_id is required", http.StatusBadRequest)
		return
	}

	res, err := h.svc.MergePR(r.Context(), prID)
	if err != nil {
		h.logger.Error("failed to merge PR", "error", err, "pr_id", prID)
		if err.Error() == "bad request: pr not found" {
			h.writeError(w, "NOT_FOUND", err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "BAD_REQUEST", err.Error(), http.StatusBadRequest)
		}
		return
	}

	h.logger.Info("PR merged successfully", "pr_id", prID)
	h.writeJSON(w, res, http.StatusOK)
}

func (h *Handler) listPRsForUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		h.writeError(w, "BAD_REQUEST", "valid user_id is required", http.StatusBadRequest)
		return
	}

	res, err := h.svc.ListPRsAssignedToUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list PRs for user", "error", err, "user_id", userID)
		h.writeError(w, "BAD_REQUEST", err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Debug("retrieved PRs for user", "user_id", userID, "prs_count", len(res))
	h.writeJSON(w, res, http.StatusOK)
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.StatsAssignments(r.Context())
	if err != nil {
		h.logger.Error("failed to get stats", "error", err)
		h.writeError(w, "INTERNAL_ERROR", "failed to get statistics", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]int{"total_assignments": c}, http.StatusOK)
}
