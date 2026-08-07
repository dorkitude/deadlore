package deadlockapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListBuildsResolvesHeroAndCachesResponses(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/assets/heroes":
			fmt.Fprint(writer, `[{"id":7,"name":"Mo & Krill"},{"id":8,"name":"Haze"}]`)
		case "/builds":
			if got, want := request.URL.Query().Get("hero_id"), "7"; got != want {
				t.Fatalf("hero_id = %q, want %q", got, want)
			}
			if got, want := request.URL.Query().Get("sort_by"), string(SortAllTime); got != want {
				t.Fatalf("sort_by = %q, want %q", got, want)
			}
			if got, want := request.URL.Query().Get("build_language"), "English"; got != want {
				t.Fatalf("build_language = %q, want %q", got, want)
			}
			fmt.Fprint(writer, `[{"hero_build":{"hero_id":7,"hero_build_id":12345,"author_account_id":77,"last_updated_timestamp":1779219922,"publish_timestamp":1779219000,"name":"Big Mo","version":6},"num_favorites":42000,"num_weekly_favorites":900}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hero, builds, cached, _, err := client.ListBuilds(context.Background(), "Mo and Krill", ListOptions{Limit: 3, Sort: SortAllTime, Language: "English"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if cached || hero != "Mo & Krill" || len(builds) != 1 {
		t.Fatalf("unexpected first result: hero=%q builds=%#v cached=%v", hero, builds, cached)
	}
	build := builds[0]
	if build.ID != 12345 || build.Favorites != 42000 || build.WeeklyFavorites != 900 || build.ViewerURL != "https://deadlocklabs.gg/builds/12345/" {
		t.Fatalf("unexpected build: %#v", build)
	}

	_, builds, cached, _, err = client.ListBuilds(context.Background(), "Mo & Krill", ListOptions{Limit: 3, Sort: SortAllTime, Language: "English"}, false)
	if err != nil || !cached || len(builds) != 1 {
		t.Fatalf("unexpected cached result: builds=%#v cached=%v err=%v", builds, cached, err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestListBuildsRejectsUnknownHeroAndInvalidLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `[{"id":8,"name":"Haze"}]`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := client.ListBuilds(context.Background(), "Nope", ListOptions{}, false); err == nil {
		t.Fatal("unknown hero did not fail")
	}
	if _, _, _, _, err := client.ListBuilds(context.Background(), "Haze", ListOptions{Limit: 101}, false); err == nil {
		t.Fatal("invalid limit did not fail")
	}
}
