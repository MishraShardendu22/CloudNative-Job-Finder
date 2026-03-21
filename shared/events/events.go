package events

import "time"

const (
	EventResumeUploaded      = "resume_uploaded"
	EventResumeParsed        = "resume_parsed"
	EventJobScraped          = "job_scraped"
	EventJobIndexed          = "job_indexed"
	EventJobMatchesGenerated = "job_matches_generated"
)

type ResumeUploadedEvent struct {
	ResumeID   string    `json:"resume_id"`
	UserID     string    `json:"user_id"`
	Bucket     string    `json:"bucket"`
	ObjectKey  string    `json:"object_key"`
	FileURL    string    `json:"file_url"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type ResumeParsedEvent struct {
	ResumeID string    `json:"resume_id"`
	UserID   string    `json:"user_id"`
	Keywords []string  `json:"keywords"`
	ParsedAt time.Time `json:"parsed_at"`
}

type JobScrapedEvent struct {
	JobID     string    `json:"job_id"`
	Source    string    `json:"source"`
	ScrapedAt time.Time `json:"scraped_at"`
}

type JobIndexedEvent struct {
	JobID     string    `json:"job_id"`
	IndexedAt time.Time `json:"indexed_at"`
}

type JobMatchesGeneratedEvent struct {
	ResumeID    string    `json:"resume_id"`
	UserID      string    `json:"user_id"`
	MatchCount  int       `json:"match_count"`
	GeneratedAt time.Time `json:"generated_at"`
}
