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
	Artist    string   `json:"artist"`
	Title     string   `json:"title"`
	Provider  string   `json:"provider"`
	SourceID  string   `json:"source_id,omitempty"`
	UpdatedAt int64    `json:"updated_at"`
	Files     []string `json:"files"`
}
