package main

type Track struct {
	Artist     string
	Title      string
	Album      string
	DurationMs int
	TrackID    string
}

type Candidate struct {
	Text     string
	Provider string
	SourceID string
}

type IndexEntry struct {
	Artist          string   `json:"artist"`
	Title           string   `json:"title"`
	Provider        string   `json:"provider,omitempty"`
	SourceID        string   `json:"source_id,omitempty"`
	CreatedAt       int64    `json:"created_at,omitempty"`
	UpdatedAt       int64    `json:"updated_at,omitempty"`
	DurationMs      int      `json:"duration_ms,omitempty"`
	Status          string   `json:"status,omitempty"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
	Files           []string `json:"files,omitempty"`
}

type FailureEvent struct {
	Artist     string `json:"artist"`
	Title      string `json:"title"`
	Provider   string `json:"provider"`
	Category   string `json:"category"`
	Reason     string `json:"reason,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Status     string `json:"status,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	TrackID    string `json:"track_id,omitempty"`
	DurationMs int    `json:"duration_ms,omitempty"`
	Path       string `json:"path,omitempty"`
	Source     string `json:"source,omitempty"`
	CreatedAt  int64  `json:"created_at,omitempty"`
}
