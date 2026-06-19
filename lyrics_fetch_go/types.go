package main

type Track struct {
	Artist     string
	Title      string
	Album      string
	DurationMs int
	TrackID    string
}

type Candidate struct {
	Text              string
	Provider          string
	SourceID          string
	Artist            string
	Title             string
	Album             string
	DurationMs        int
	Score             int
	MetadataAvailable bool
	ProvenanceStatus  string
	ValidationVersion string
}

type IndexEntry struct {
	Artist                     string   `json:"artist"`
	Title                      string   `json:"title"`
	Provider                   string   `json:"provider,omitempty"`
	SourceID                   string   `json:"source_id,omitempty"`
	CreatedAt                  int64    `json:"created_at,omitempty"`
	UpdatedAt                  int64    `json:"updated_at,omitempty"`
	DurationMs                 int      `json:"duration_ms,omitempty"`
	Status                     string   `json:"status,omitempty"`
	RejectionReason            string   `json:"rejection_reason,omitempty"`
	Files                      []string `json:"files,omitempty"`
	CandidateArtist            string   `json:"candidate_artist,omitempty"`
	CandidateTitle             string   `json:"candidate_title,omitempty"`
	CandidateAlbum             string   `json:"candidate_album,omitempty"`
	CandidateDurationMs        int      `json:"candidate_duration_ms,omitempty"`
	Score                      int      `json:"score,omitempty"`
	CandidateMetadataAvailable bool     `json:"candidate_metadata_available,omitempty"`
	ProvenanceStatus           string   `json:"provenance_status,omitempty"`
	ValidationVersion          string   `json:"validation_version,omitempty"`
	AcceptedAt                 int64    `json:"accepted_at,omitempty"`
}

type CandidateEvaluationEvent struct {
	Event                      string   `json:"event"`
	Provider                   string   `json:"provider"`
	SourceID                   string   `json:"source_id,omitempty"`
	TargetTrackID              string   `json:"target_track_id,omitempty"`
	TargetArtist               string   `json:"target_artist"`
	TargetTitle                string   `json:"target_title"`
	TargetAlbum                string   `json:"target_album,omitempty"`
	TargetDurationMs           *int     `json:"target_duration_ms,omitempty"`
	EvaluationStage            string   `json:"evaluation_stage,omitempty"`
	Decision                   string   `json:"decision,omitempty"`
	CacheReused                bool     `json:"cache_reused,omitempty"`
	CandidateMetadataAvailable *bool    `json:"candidate_metadata_available,omitempty"`
	ProvenanceStatus           string   `json:"provenance_status,omitempty"`
	CandidateArtist            string   `json:"candidate_artist,omitempty"`
	CandidateTitle             string   `json:"candidate_title,omitempty"`
	CandidateAlbum             string   `json:"candidate_album,omitempty"`
	CandidateDurationMs        *int     `json:"candidate_duration_ms,omitempty"`
	TitleMatchType             string   `json:"title_match_type,omitempty"`
	ArtistMatchType            string   `json:"artist_match_type,omitempty"`
	DurationDeltaMs            *int     `json:"duration_delta_ms,omitempty"`
	Score                      *int     `json:"score,omitempty"`
	Accepted                   *bool    `json:"accepted,omitempty"`
	RejectionReasons           []string `json:"rejection_reasons,omitempty"`
	ValidationVersion          string   `json:"validation_version,omitempty"`
	AcceptedAt                 int64    `json:"accepted_at,omitempty"`
	CreatedAt                  int64    `json:"created_at,omitempty"`
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
