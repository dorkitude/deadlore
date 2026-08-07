// Package deadlockapi reads public Deadlock community-build metadata. It never
// copies a build's item list, annotations, or description into the local cache.
package deadlockapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.deadlock-api.com/v1"

const DefaultBuildViewerURL = "https://deadlocklabs.gg/builds"

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	cache      *cache
	ttl        time.Duration
}

type Sort string

const (
	SortWeekly  Sort = "weekly_favorites"
	SortAllTime Sort = "favorites"
	SortRecent  Sort = "updated_at"
)

type ListOptions struct {
	Limit    int
	Sort     Sort
	Language string
}

type Build struct {
	ID              int        `json:"id"`
	HeroID          int        `json:"hero_id"`
	AuthorAccountID int        `json:"author_account_id"`
	Name            string     `json:"name"`
	Version         int        `json:"version"`
	UpdatedAt       time.Time  `json:"updated_at"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	Favorites       int        `json:"favorites"`
	WeeklyFavorites int        `json:"weekly_favorites"`
	ViewerURL       string     `json:"viewer_url"`
}

type hero struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type apiBuild struct {
	HeroBuild struct {
		HeroID               int    `json:"hero_id"`
		HeroBuildID          int    `json:"hero_build_id"`
		AuthorAccountID      int    `json:"author_account_id"`
		LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
		PublishTimestamp     *int64 `json:"publish_timestamp"`
		Name                 string `json:"name"`
		Version              int    `json:"version"`
	} `json:"hero_build"`
	NumFavorites       *int `json:"num_favorites"`
	NumWeeklyFavorites *int `json:"num_weekly_favorites"`
}

func NewClient(baseURL, cacheDirectory string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse Deadlock API URL: %w", err)
	}
	responseCache, err := newCache(cacheDirectory)
	if err != nil {
		return nil, fmt.Errorf("create Deadlock API cache: %w", err)
	}
	return &Client{baseURL: parsed, httpClient: &http.Client{Timeout: 15 * time.Second}, cache: responseCache, ttl: 6 * time.Hour}, nil
}

func (c *Client) ListBuilds(ctx context.Context, heroName string, options ListOptions, refresh bool) (string, []Build, bool, time.Time, error) {
	heroes, _, err := c.heroes(ctx, refresh)
	if err != nil {
		return "", nil, false, time.Time{}, err
	}
	var selected *hero
	for index := range heroes {
		if normalize(heroes[index].Name) == normalize(heroName) {
			selected = &heroes[index]
			break
		}
	}
	if selected == nil {
		return "", nil, false, time.Time{}, fmt.Errorf("%q was not found in the Deadlock API hero catalog", heroName)
	}

	if options.Limit <= 0 {
		options.Limit = 5
	}
	if options.Limit > 100 {
		return "", nil, false, time.Time{}, fmt.Errorf("build limit must be between 1 and 100")
	}
	if options.Sort == "" {
		options.Sort = SortWeekly
	}
	if options.Sort != SortWeekly && options.Sort != SortAllTime && options.Sort != SortRecent {
		return "", nil, false, time.Time{}, fmt.Errorf("unknown build sort %q", options.Sort)
	}

	query := url.Values{
		"hero_id":        {strconv.Itoa(selected.ID)},
		"limit":          {strconv.Itoa(options.Limit)},
		"only_latest":    {"true"},
		"sort_by":        {string(options.Sort)},
		"sort_direction": {"desc"},
	}
	if strings.TrimSpace(options.Language) != "" {
		query.Set("build_language", options.Language)
	}
	key := "builds?" + query.Encode()
	var response []apiBuild
	fetchedAt, cached, err := c.get(ctx, key, "builds", query, &response, refresh)
	if err != nil {
		return "", nil, false, time.Time{}, err
	}
	builds := make([]Build, 0, len(response))
	for _, candidate := range response {
		build := Build{
			ID:              candidate.HeroBuild.HeroBuildID,
			HeroID:          candidate.HeroBuild.HeroID,
			AuthorAccountID: candidate.HeroBuild.AuthorAccountID,
			Name:            candidate.HeroBuild.Name,
			Version:         candidate.HeroBuild.Version,
			UpdatedAt:       time.Unix(candidate.HeroBuild.LastUpdatedTimestamp, 0).UTC(),
			ViewerURL:       buildViewerURL(candidate.HeroBuild.HeroBuildID),
		}
		if candidate.HeroBuild.PublishTimestamp != nil {
			publishedAt := time.Unix(*candidate.HeroBuild.PublishTimestamp, 0).UTC()
			build.PublishedAt = &publishedAt
		}
		if candidate.NumFavorites != nil {
			build.Favorites = *candidate.NumFavorites
		}
		if candidate.NumWeeklyFavorites != nil {
			build.WeeklyFavorites = *candidate.NumWeeklyFavorites
		}
		builds = append(builds, build)
	}
	return selected.Name, builds, cached, fetchedAt, nil
}

func (c *Client) CacheStatus() (int, time.Time, error) { return c.cache.status() }

func (c *Client) ClearCache() error { return c.cache.clear() }

func (c *Client) heroes(ctx context.Context, refresh bool) ([]hero, bool, error) {
	var heroes []hero
	_, cached, err := c.get(ctx, "heroes", "assets/heroes", nil, &heroes, refresh)
	return heroes, cached, err
}

func (c *Client) get(ctx context.Context, key, endpoint string, query url.Values, target any, refresh bool) (time.Time, bool, error) {
	if fetchedAt, found, err := c.cache.load(key, target); err != nil {
		return time.Time{}, false, err
	} else if found && !refresh && time.Since(fetchedAt) < c.ttl {
		return fetchedAt, true, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(endpoint, query), nil)
	if err != nil {
		return time.Time{}, false, err
	}
	request.Header.Set("User-Agent", "deadlore (+https://github.com/dorkitude/deadlore; public build metadata)")
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("fetch Deadlock API %s: %w", endpoint, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return time.Time{}, false, fmt.Errorf("fetch Deadlock API %s: unexpected HTTP status %s", endpoint, response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return time.Time{}, false, fmt.Errorf("parse Deadlock API %s: %w", endpoint, err)
	}
	fetchedAt := time.Now().UTC()
	if err := c.cache.save(key, target, fetchedAt); err != nil {
		return time.Time{}, false, err
	}
	return fetchedAt, false, nil
}

func (c *Client) url(endpoint string, query url.Values) string {
	copy := *c.baseURL
	copy.Path = strings.TrimRight(copy.Path, "/") + "/" + strings.TrimLeft(endpoint, "/")
	copy.RawQuery = query.Encode()
	return copy.String()
}

func buildViewerURL(buildID int) string { return fmt.Sprintf("%s/%d/", DefaultBuildViewerURL, buildID) }

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("&", "and", "'", "", "’", "", "-", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}
