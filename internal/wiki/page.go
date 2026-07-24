package wiki

import "time"

type Page struct {
	CacheVersion int       `json:"cache_version"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	RevisionID   string    `json:"revision_id,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
	Summary      string    `json:"summary,omitempty"`
	Facts        []Fact    `json:"facts,omitempty"`
	Abilities    []Ability `json:"abilities,omitempty"`
	Catalog      []string  `json:"catalog,omitempty"`
	Sections     []Section `json:"sections,omitempty"`
}

type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Ability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Section struct {
	Title string   `json:"title"`
	Text  []string `json:"text"`
}
