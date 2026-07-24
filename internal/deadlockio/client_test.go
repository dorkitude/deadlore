package deadlockio

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dorkitude/deadlore/internal/wiki"
)

func TestClientFetchesParsesAndCachesHeroIndependently(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/heroes.json":
			fmt.Fprint(writer, `{"heroes":[{"slug":"haze","displayName":{"english":"Haze"}}],"source":{"buildId":"123","generatedAt":"2026-07-01T00:00:00Z"}}`)
		case "/heroes/haze.json":
			fmt.Fprint(writer, `{"slug":"haze","displayName":{"english":"Haze"},"playstyle":{"english":"A <b>stealth</b> hero."},"updatedAt":"2026-07-01T00:00:00Z","sourcePath":"game/haze","stats":{"EMaxHealth":730,"EBaseHealthRegen":2,"EMaxMoveSpeed":8.2,"ESprintSpeed":1.6},"weapon":{"dps":50.1,"sustainedDps":26.4,"bulletDamage":5.26,"roundsPerSecond":9.52,"clipSize":25,"reloadTime":2.35,"bulletSpeed":30000},"tags":[{"english":"Assassin"}],"abilities":[{"displayName":{"english":"Sleep Dagger"},"description":{"english":"Throw a <b>dagger</b>."}}]}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	page, cached, err := client.Get(context.Background(), "Haze", "hero", false)
	if err != nil {
		t.Fatal(err)
	}
	if cached || page.Title != "Haze" || page.Summary != "A stealth hero." {
		t.Fatalf("unexpected first page: cached=%v page=%#v", cached, page)
	}
	if got, found := pageFact(page, "Bullets per sec"); !found || got != "9.52" {
		t.Fatalf("rounds per second = %q, %v", got, found)
	}
	if len(page.Abilities) != 1 || page.Abilities[0].Description != "Throw a dagger." {
		t.Fatalf("unexpected abilities: %#v", page.Abilities)
	}

	page, cached, err = client.Get(context.Background(), "Haze", "hero", false)
	if err != nil || !cached || page.URL != "https://deadlock.io/en/heroes/haze" {
		t.Fatalf("unexpected cached page: cached=%v page=%#v err=%v", cached, page, err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (catalog and page)", requests)
	}
}

func TestCacheClearRemovesOnlyDeadlockIOEntries(t *testing.T) {
	client, err := NewClient(DefaultBaseURL, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := client.cache.save("hero:Haze", newTestPage("Haze")); err != nil {
		t.Fatal(err)
	}
	if err := client.cache.save("item:Leech", newTestPage("Leech")); err != nil {
		t.Fatal(err)
	}
	if err := client.ClearCache("Haze"); err != nil {
		t.Fatal(err)
	}
	if page, err := client.cache.load("hero:Haze"); err != nil || page != nil {
		t.Fatalf("hero page after clear = %#v, %v", page, err)
	}
	if page, err := client.cache.load("item:Leech"); err != nil || page == nil {
		t.Fatalf("item page after hero clear = %#v, %v", page, err)
	}
}

func pageFact(page *wiki.Page, label string) (string, bool) {
	for _, fact := range page.Facts {
		if fact.Label == label {
			return fact.Value, true
		}
	}
	return "", false
}

func newTestPage(title string) *wiki.Page { return &wiki.Page{Title: title} }
