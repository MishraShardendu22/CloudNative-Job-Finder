package events

import "time"

const (
	EventResumeUploaded      = "resume_uploaded"
	EventResumeParsed        = "resume_parsed"
	EventJobScraped          = "job_scraped"
	EventJobIndexed          = "job_indexed"
	EventJobMatchesGenerated = "job_matches_generated"
	EventUserInteraction     = "user_interaction"
	EventUserFeaturesUpdated = "user_features_updated"
	EventOutboxFailed        = "outbox_failed"
)

const (
	TopicJobsScrapedV1       = "jobs.scraped.v1"
	TopicJobsProcessedV1     = "jobs.processed.v1"
	TopicResumeParsedV1      = "resume.parsed.v1"
	TopicMatchesGeneratedV1  = "matches.generated.v1"
	TopicUserInteractionV1   = "user.interaction.v1"
	TopicUserFeaturesV1      = "user.features.updated.v1"
	TopicDLQPrefixDefault    = "dlq."
	OutboxStatusPending      = "pending"
	OutboxStatusPublished    = "published"
	OutboxStatusFailed       = "failed"
	OutboxStatusDeadLettered = "dead_letter"
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

type UserInteractionEvent struct {
	EventID         string         `json:"event_id,omitempty"`
	UserID          string         `json:"user_id"`
	ResumeID        string         `json:"resume_id"`
	JobID           string         `json:"job_id"`
	InteractionType string         `json:"interaction_type"`
	Source          string         `json:"source"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	OccurredAt      time.Time      `json:"occurred_at"`
}

type UserFeaturesUpdatedEvent struct {
	EventID        string    `json:"event_id,omitempty"`
	UserID         string    `json:"user_id"`
	ResumeID       string    `json:"resume_id"`
	JobID          string    `json:"job_id"`
	CTR            float64   `json:"ctr"`
	AffinityScore  float64   `json:"affinity_score"`
	Impressions    int       `json:"impressions"`
	Clicks         int       `json:"clicks"`
	Applies        int       `json:"applies"`
	LastOccurredAt time.Time `json:"last_occurred_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func TopicForEventType(eventType string) string {
	switch eventType {
	case EventJobScraped:
		return TopicJobsScrapedV1
	case EventJobIndexed:
		return TopicJobsProcessedV1
	case EventResumeParsed:
		return TopicResumeParsedV1
	case EventJobMatchesGenerated:
		return TopicMatchesGeneratedV1
	case EventUserInteraction:
		return TopicUserInteractionV1
	case EventUserFeaturesUpdated:
		return TopicUserFeaturesV1
	default:
		return ""
	}
}

func DLQTopicFor(topic, prefix string) string {
	if prefix == "" {
		prefix = TopicDLQPrefixDefault
	}
	return prefix + topic
}
