package deadlockio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dorkitude/deadlore/internal/wiki"
)

const DefaultBaseURL = "https://deadlock.io/api/v1"

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	cache      *cache
	ttl        time.Duration
}

func NewClient(baseURL, cacheDirectory string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse Deadlock.io URL: %w", err)
	}
	pageCache, err := newCache(cacheDirectory)
	if err != nil {
		return nil, fmt.Errorf("create Deadlock.io cache: %w", err)
	}
	return &Client{baseURL: parsed, httpClient: &http.Client{Timeout: 15 * time.Second}, cache: pageCache, ttl: 6 * time.Hour}, nil
}

func (c *Client) Get(ctx context.Context, title, kind string, refresh bool) (*wiki.Page, bool, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, false, fmt.Errorf("a Deadlock.io title is required")
	}
	if kind == "" {
		for _, candidate := range []string{"hero", "item", "mechanic"} {
			page, cached, err := c.Get(ctx, title, candidate, refresh)
			if page != nil {
				return page, cached, err
			}
		}
		return nil, false, fmt.Errorf("%q was not found on deadlock.io", title)
	}

	key := kind + ":" + title
	if cached, err := c.cache.load(key); err != nil {
		return nil, false, err
	} else if cached != nil && !refresh && time.Since(cached.FetchedAt) < c.ttl {
		return cached, true, nil
	}

	slug, err := c.resolveSlug(ctx, title, kind, refresh)
	if err != nil {
		return nil, false, err
	}
	var page *wiki.Page
	switch kind {
	case "hero":
		page, err = c.fetchHero(ctx, slug)
	case "item":
		page, err = c.fetchItem(ctx, slug)
	case "mechanic":
		page, err = c.fetchMechanic(ctx, slug)
	default:
		return nil, false, fmt.Errorf("unknown Deadlock.io kind %q", kind)
	}
	if err != nil {
		return nil, false, err
	}
	if err := c.cache.save(key, page); err != nil {
		return nil, false, err
	}
	return page, false, nil
}

func (c *Client) Catalog(ctx context.Context, kind string, refresh bool) (*wiki.Page, bool, error) {
	key := "catalog:" + kind
	if cached, err := c.cache.load(key); err != nil {
		return nil, false, err
	} else if cached != nil && !refresh && time.Since(cached.FetchedAt) < c.ttl {
		return cached, true, nil
	}
	var response catalogResponse
	endpoint := map[string]string{"hero": "heroes.json", "item": "items.json", "mechanic": "mechanics.json"}[kind]
	if endpoint == "" {
		return nil, false, fmt.Errorf("unknown Deadlock.io kind %q", kind)
	}
	if err := c.fetchJSON(ctx, endpoint, &response); err != nil {
		return nil, false, err
	}
	entries := response.entries(kind)
	if len(entries) == 0 {
		return nil, false, fmt.Errorf("Deadlock.io did not provide a %s catalog", kind)
	}
	page := &wiki.Page{Title: "Deadlock.io " + kind + " catalog", URL: c.url(endpoint), Catalog: entries, FetchedAt: time.Now().UTC(), RevisionID: response.Source.BuildID, LastModified: "Deadlock.io generated: " + response.Source.GeneratedAt}
	if err := c.cache.save(key, page); err != nil {
		return nil, false, err
	}
	return page, false, nil
}

func (c *Client) GetAbility(ctx context.Context, name string, refresh bool) (*wiki.Page, wiki.Ability, bool, bool, error) {
	catalog, _, err := c.Catalog(ctx, "hero", refresh)
	if err != nil {
		return nil, wiki.Ability{}, false, false, err
	}
	for _, entry := range catalog.Catalog {
		title, _, found := strings.Cut(entry, "\t")
		if !found {
			continue
		}
		page, cached, fetchErr := c.Get(ctx, title, "hero", refresh)
		if page == nil {
			continue
		}
		for _, ability := range page.Abilities {
			if normalize(ability.Name) == normalize(name) {
				return page, ability, true, cached, fetchErr
			}
		}
	}
	return nil, wiki.Ability{}, false, false, fmt.Errorf("%q was not found in Deadlock.io hero abilities", name)
}

func (c *Client) CacheStatus() (int, time.Time, error) { return c.cache.status() }

func (c *Client) ClearCache(title string) error {
	if strings.TrimSpace(title) == "" {
		return c.cache.clear("")
	}
	for _, kind := range []string{"hero", "item", "mechanic"} {
		if err := c.cache.clear(kind + ":" + title); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) resolveSlug(ctx context.Context, title, kind string, refresh bool) (string, error) {
	catalog, _, err := c.Catalog(ctx, kind, refresh)
	if err != nil {
		return "", err
	}
	for _, entry := range catalog.Catalog {
		name, slug, found := strings.Cut(entry, "\t")
		if found && normalize(name) == normalize(title) {
			return slug, nil
		}
	}
	return "", fmt.Errorf("%q was not found in the Deadlock.io %s catalog", title, kind)
}

func (c *Client) fetchJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(endpoint), nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "deadlore (+https://github.com/dorkitude/deadlore; fallback lookup)")
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("fetch Deadlock.io %s: %w", endpoint, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Deadlock.io %s was not found", endpoint)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("fetch Deadlock.io %s: unexpected HTTP status %s", endpoint, response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("parse Deadlock.io %s: %w", endpoint, err)
	}
	return nil
}

func (c *Client) url(endpoint string) string {
	copy := *c.baseURL
	copy.Path = strings.TrimRight(copy.Path, "/") + "/" + endpoint
	return copy.String()
}

func normalize(value string) string { return strings.ToLower(strings.Join(strings.Fields(value), " ")) }
