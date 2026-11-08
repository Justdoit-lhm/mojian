// Package update checks Mojian's release feed and installs the signed
// archive MyGo verifies. The app keeps only the window's update state.
package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/egoist/mygo"
)

// updateRepo is the GitHub repository releases are published from.
const updateRepo = "ox0pine/mojian"

// githubUpdates reads update manifests from the latest GitHub release and
// downloads their files from the release's asset URLs. Going through
// releases/latest/download avoids the GitHub API, whose unauthenticated
// rate limit is easy to exhaust, and keeps a check to one request. The
// files are still checked against the updater's signed manifest.
type githubUpdates struct {
	base   string        // "https://github.com/ox0pine/mojian" when empty
	client *http.Client  // a 20 second client when nil
	delay  time.Duration // first retry delay; 1 second when zero
}

// Manifest returns the manifest of the running platform in the latest
// release.
func (g githubUpdates) Manifest(ctx context.Context, prereleases bool) (io.ReadCloser, error) {
	target := runtime.GOOS + "-" + runtime.GOARCH
	client := g.client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return g.get(ctx, g.url("/releases/latest/download/update-"+url.PathEscape(target)+".json"), client)
}

// Open downloads a file the manifest names.
func (g githubUpdates) Open(ctx context.Context, raw string) (io.ReadCloser, error) {
	client := g.client
	if client == nil {
		client = &http.Client{}
	}
	return g.get(ctx, raw, client)
}

func (g githubUpdates) url(path string) string {
	base := g.base
	if base == "" {
		base = "https://github.com/" + updateRepo
	}
	return base + path
}

// get fetches raw, trying again after a 5xx, 408, 429 or a connection
// error. GitHub's release download host answers 504 now and then; a few
// seconds are usually enough.
func (g githubUpdates) get(ctx context.Context, raw string, client *http.Client) (io.ReadCloser, error) {
	delay := g.delay
	if delay == 0 {
		delay = time.Second
	}
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				return resp.Body, nil
			}
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
			resp.Body.Close()
			last = fmt.Errorf("GET %s: %s", raw, resp.Status)
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusRequestTimeout && resp.StatusCode != http.StatusTooManyRequests {
				return nil, last
			}
		} else {
			last = err
		}
		if attempt == 4 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay << attempt):
		}
	}
	return nil, last
}

// UseGitHub points MyGo's updater at the release downloads instead of
// its API-backed feed.
func UseGitHub() {
	mygo.Updater.SetSource(githubUpdates{})
}
