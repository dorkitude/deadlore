package wiki

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://deadlock.wiki"
	DefaultTTL     = 6 * time.Hour
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	cache      *Cache
	ttl        time.Duration
}

func NewClient(baseURL, cacheDirectory string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsedURL, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse wiki URL: %w", err)
	}
	cache, err := NewCache(cacheDirectory)
	if err != nil {
		return nil, fmt.Errorf("create cache: %w", err)
	}

	return &Client{
		baseURL:    parsedURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cache:      cache,
		ttl:        DefaultTTL,
	}, nil
}

func (c *Client) Get(ctx context.Context, title string, refresh bool) (*Page, bool, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, false, fmt.Errorf("a wiki title is required")
	}

	cached, err := c.cache.Load(title)
	if err != nil {
		return nil, false, fmt.Errorf("read cache: %w", err)
	}
	if cached != nil && cached.CacheVersion == CurrentCacheVersion && !refresh && time.Since(cached.FetchedAt) < c.ttl {
		return cached, true, nil
	}

	page, err := c.fetch(ctx, title)
	if err != nil {
		if cached != nil {
			return cached, true, fmt.Errorf("refresh failed; showing cached data from %s: %w", cached.FetchedAt.Format(time.RFC3339), err)
		}
		return nil, false, err
	}
	if err := c.cache.Save(title, page); err != nil {
		return nil, false, fmt.Errorf("write cache: %w", err)
	}
	return page, false, nil
}

func (c *Client) Cache() *Cache { return c.cache }

func (c *Client) fetch(ctx context.Context, title string) (*Page, error) {
	articleURL := *c.baseURL
	articleURL.Path = strings.TrimRight(articleURL.Path, "/") + "/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "deadlore/0.1 (+https://github.com/dorkitude/deadlore; one-page lookup)")
	request.Header.Set("Accept", "text/html,application/xhtml+xml")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", articleURL.String(), err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%q was not found on %s", title, c.baseURL.Host)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: unexpected HTTP status %s", articleURL.String(), response.Status)
	}

	page, err := ParsePage(response.Body, response.Request.URL.String(), time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", articleURL.String(), err)
	}
	return page, nil
}
