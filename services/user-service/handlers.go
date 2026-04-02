package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/httpx"
	"job-finder/shared/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "user-service"})
}

func (a *app) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid signup payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		httpx.WriteError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if len(req.Password) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to secure password")
		return
	}

	var user models.UserResponse
	err = a.pool.QueryRow(
		r.Context(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, created_at`,
		req.Email,
		string(hash),
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteError(w, http.StatusConflict, "email already exists")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	go func(emailAddress string) {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := a.emailClient.Send(ctx, emailAddress, "Welcome to Job Finder", "Your account is ready. Upload resumes to start receiving job matches."); err != nil {
			log.Printf("welcome email failed: %v", err)
		}
	}(user.Email)

	httpx.WriteJSON(w, http.StatusCreated, user)
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid login payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.UserResponse
	var passwordHash string
	err := a.pool.QueryRow(
		r.Context(),
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &passwordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

func (a *app) handleProfile(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "X-User-ID header is required")
		return
	}

	var user models.UserResponse
	err := a.pool.QueryRow(
		r.Context(),
		`SELECT id, email, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to fetch profile")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

func (a *app) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "X-User-ID header is required")
		return
	}

	var req models.UpdateProfileRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid update profile payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		httpx.WriteError(w, http.StatusBadRequest, "valid email is required")
		return
	}

	var user models.UserResponse
	err := a.pool.QueryRow(
		r.Context(),
		`UPDATE users
		 SET email = $1
		 WHERE id = $2
		 RETURNING id, email, created_at`,
		req.Email,
		userID,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteError(w, http.StatusConflict, "email already exists")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}
