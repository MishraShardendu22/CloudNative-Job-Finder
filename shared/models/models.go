package models

import "time"

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type ResumeResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	FileURL       string    `json:"file_url"`
	ParsedKeyword []string  `json:"parsed_keywords"`
	CreatedAt     time.Time `json:"created_at"`
}

type JobRecommendation struct {
	JobID    string   `json:"job_id"`
	Title    string   `json:"title"`
	Company  string   `json:"company"`
	Location string   `json:"location"`
	URL      string   `json:"url"`
	Keywords []string `json:"keywords"`
	Score    float64  `json:"score"`
}

type RecommendationPage struct {
	ResumeID string              `json:"resume_id"`
	Total    int                 `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
	Items    []JobRecommendation `json:"items"`
}
