package wiki

import "time"

type Page struct {
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	RevisionID   string    `json:"revision_id,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
	Summary      string    `json:"summary,omitempty"`
	Facts        []Fact    `json:"facts,omitempty"`
	Sections     []Section `json:"sections,omitempty"`
}

type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Section struct {
	Title string   `json:"title"`
	Text  []string `json:"text"`
}
