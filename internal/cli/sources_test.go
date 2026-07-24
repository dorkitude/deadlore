package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLookupUsesDeadlockIOOnlyAsFallback(t *testing.T) {
	t.Run("wiki result does not query fallback", func(t *testing.T) {
		wikiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			fmt.Fprint(writer, testWikiPage("Haze"))
		}))
		defer wikiServer.Close()

		ioRequests := 0
		ioServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ioRequests++
			http.Error(writer, "should not be called", http.StatusInternalServerError)
		}))
		defer ioServer.Close()

		output := runLookup(t, wikiServer.URL, ioServer.URL)
		if ioRequests != 0 {
			t.Fatalf("Deadlock.io requests = %d, want 0 when Wiki has the page", ioRequests)
		}
		if !strings.Contains(output, "Source: "+wikiServer.URL+"/Haze") || strings.Contains(output, "Source fallback") {
			t.Fatalf("unexpected Wiki output:\n%s", output)
		}
	})

	t.Run("wiki miss uses cached deadlock io fallback", func(t *testing.T) {
		wikiServer := httptest.NewServer(http.NotFoundHandler())
		defer wikiServer.Close()
		ioServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Path {
			case "/heroes.json":
				fmt.Fprint(writer, `{"heroes":[{"slug":"haze","displayName":{"english":"Haze"}}]}`)
			case "/heroes/haze.json":
				fmt.Fprint(writer, `{"slug":"haze","displayName":{"english":"Haze"},"playstyle":{"english":"Fallback hero."},"sourcePath":"game/haze","updatedAt":"2026-07-01T00:00:00Z"}`)
			default:
				http.NotFound(writer, request)
			}
		}))
		defer ioServer.Close()

		output := runLookup(t, wikiServer.URL, ioServer.URL)
		if !strings.Contains(output, "Source fallback") || !strings.Contains(output, "Haze · Deadlock.io") || !strings.Contains(output, "deadlock.io/en/heroes/haze") {
			t.Fatalf("unexpected fallback output:\n%s", output)
		}
		if strings.Contains(output, "Source comparison") {
			t.Fatalf("fallback output must not compare sources:\n%s", output)
		}
	})
}

func runLookup(t *testing.T, wikiURL, ioURL string) string {
	t.Helper()
	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--no-color", "--cache-dir", t.TempDir(), "--wiki-url", wikiURL, "--deadlockio-url", ioURL, "Haze"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func testWikiPage(title string) string {
	return `<!doctype html><html><body><h1 id="firstHeading">` + title + `</h1><div id="mw-content-text"><p>Wiki hero.</p></div></body></html>`
}
