package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildCommandShowsMetadataAndViewerLinkWithoutDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/assets/heroes":
			fmt.Fprint(writer, `[{"id":8,"name":"Haze"}]`)
		case "/builds":
			fmt.Fprint(writer, `[{"hero_build":{"hero_id":8,"hero_build_id":12345,"author_account_id":77,"last_updated_timestamp":1779219922,"name":"Rapid Haze","version":6,"description":"THIS MUST NOT APPEAR"},"num_favorites":42000,"num_weekly_favorites":900}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--no-color", "--cache-dir", t.TempDir(), "--deadlock-api-url", server.URL, "build", "Haze"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, expected := range []string{"Rapid Haze", "900 weekly favorites", "Build ID 12345", "https://deadlocklabs.gg/builds/12345/"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("output missing %q:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "THIS MUST NOT APPEAR") {
		t.Fatalf("build description leaked into output:\n%s", got)
	}
}

func TestBuildCommandJSONIsStructured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/assets/heroes":
			fmt.Fprint(writer, `[{"id":8,"name":"Haze"}]`)
		case "/builds":
			fmt.Fprint(writer, `[]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--json", "--cache-dir", t.TempDir(), "--deadlock-api-url", server.URL, "build", "Haze", "--sort", "recent"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	var response map[string]any
	if err := json.Unmarshal([]byte(got), &response); err != nil {
		t.Fatalf("output was not JSON: %v\n%s", err, got)
	}
	if response["type"] != "builds" || response["sort"] != "recent" || strings.Contains(got, "\x1b[") {
		t.Fatalf("unexpected JSON output:\n%s", got)
	}
}
