package wiki

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheUsesRequestedTitleRatherThanResolvedPageTitle(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	page := &Page{Title: "Haze", URL: "https://deadlock.wiki/Sleep_Dagger", CacheVersion: CurrentCacheVersion}
	if err := cache.Save("Sleep Dagger", page); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cached, err := cache.Load("Sleep Dagger")
	if err != nil || cached == nil {
		t.Fatalf("Load requested title = (%#v, %v), want page", cached, err)
	}
	if cached.URL != page.URL {
		t.Fatalf("cached URL = %q, want %q", cached.URL, page.URL)
	}
	resolved, err := cache.Load("Haze")
	if err != nil {
		t.Fatalf("Load resolved title: %v", err)
	}
	if resolved != nil {
		t.Fatalf("resolved-title cache should be empty, got %#v", resolved)
	}
}

func TestClientCachesResolvedPageUnderRequestedTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/Sleep_Dagger":
			fmt.Fprint(writer, testPageHTML("Haze"))
		case "/Haze":
			fmt.Fprint(writer, testPageHTML("Haze"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	abilityPage, cached, err := client.Get(t.Context(), "Sleep Dagger", false)
	if err != nil {
		t.Fatalf("get ability page: %v", err)
	}
	if cached || abilityPage.URL != server.URL+"/Sleep_Dagger" {
		t.Fatalf("ability page = (%q, cached %t), want %q, cached false", abilityPage.URL, cached, server.URL+"/Sleep_Dagger")
	}

	heroPage, cached, err := client.Get(t.Context(), "Haze", false)
	if err != nil {
		t.Fatalf("get hero page: %v", err)
	}
	if cached || heroPage.URL != server.URL+"/Haze" {
		t.Fatalf("hero page = (%q, cached %t), want %q, cached false", heroPage.URL, cached, server.URL+"/Haze")
	}
}

func testPageHTML(title string) string {
	return fmt.Sprintf(`<!doctype html><html><body>
<h1 id="firstHeading">%s</h1>
<div id="mw-content-text"><div class="mw-parser-output"><p>Test page.</p></div></div>
</body></html>`, title)
}
