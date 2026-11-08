package update

import (
	"context"
	"errors"

	"github.com/egoist/mygo"
)

// Release is a newer version the updater found. It keeps MyGo's Update,
// which installs the signed archive.
type Release struct {
	Version string
	update  *mygo.Update
}

// Check asks the release feed configured in mygo.json for a version
// newer than the running app.
func Check(ctx context.Context) (*Release, error) {
	up, err := mygo.Updater.Check(ctx)
	if err != nil {
		return nil, err
	}
	if up == nil {
		return nil, nil
	}
	return &Release{Version: up.Version, update: up}, nil
}

// Install downloads and verifies the update. It returns when the new
// version is staged, or installed in place on Linux; the app keeps
// running the old version until App.Relaunch.
func Install(ctx context.Context, up *Release) error {
	if up == nil || up.update == nil {
		return ErrNoUpdate
	}
	return up.update.Install(ctx, nil)
}

// ErrNoUpdate is reported when Install is asked for a release the checker
// did not keep.
var ErrNoUpdate = errors.New("no update was found")

// ErrorMessage is the user-facing text for a failed check or install.
func ErrorMessage(err error) string {
	if errors.Is(err, mygo.ErrUpdatesDisabled) {
		return "This build cannot update itself. Install the app in a folder you can write, or update it with its package manager."
	}
	return err.Error()
}
