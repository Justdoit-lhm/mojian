package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestGitHubUpdatesRetriesTransientFailures(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		want := "/releases/latest/download/update-" + runtime.GOOS + "-" + runtime.GOARCH + ".json"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_, _ = io.WriteString(w, `{"version":"0.2.2"}`)
	}))
	defer srv.Close()

	g := githubUpdates{base: srv.URL, delay: time.Millisecond}
	body, err := g.Manifest(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"version":"0.2.2"}` {
		t.Fatalf("manifest = %q", data)
	}
	if hits != 3 {
		t.Fatalf("requests = %d, want 3", hits)
	}
}

func TestGitHubUpdatesDoesNotRetryNotFound(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.NotFound(w, r)
	}))
	defer srv.Close()

	g := githubUpdates{base: srv.URL, delay: time.Millisecond}
	if _, err := g.Manifest(context.Background(), false); err == nil {
		t.Fatal("a missing manifest was accepted")
	}
	if hits != 1 {
		t.Fatalf("requests = %d, want 1", hits)
	}
}

func TestGitHubUpdatesOpensManifestFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/archive.tar.gz" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, "archive")
	}))
	defer srv.Close()

	g := githubUpdates{delay: time.Millisecond}
	body, err := g.Open(context.Background(), srv.URL+"/archive.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "archive" {
		t.Fatalf("archive = %q", data)
	}
}
