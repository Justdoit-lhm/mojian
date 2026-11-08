package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo"
)

// updateRelease is a newer version the updater found. It keeps MyGo's
// Update, which installs the signed archive.
type updateRelease struct {
	Version string
	update  *mygo.Update
}

// checkMygoUpdate asks the release feed configured in mygo.json for a
// version newer than the running app.
func checkMygoUpdate(ctx context.Context) (*updateRelease, error) {
	up, err := mygo.Updater.Check(ctx)
	if err != nil {
		return nil, err
	}
	if up == nil {
		return nil, nil
	}
	return &updateRelease{Version: up.Version, update: up}, nil
}

// installMygoUpdate downloads and verifies the update. It returns when the
// new version is staged, or installed in place on Linux; the app keeps
// running the old version until App.Relaunch.
func installMygoUpdate(ctx context.Context, up *updateRelease) error {
	if up == nil || up.update == nil {
		return errNoUpdate
	}
	return up.update.Install(ctx, nil)
}

// errNoUpdate is reported when Install is asked for a release the checker
// did not keep.
var errNoUpdate = errors.New("no update was found")

// checkForUpdates checks the feed in the background, so the settings window
// never waits on the network. The buttons and the state change on the UI
// thread.
func (a *app) checkForUpdates() {
	if a.updateChecking || a.updateInstalling {
		return
	}
	a.updateChecking = true
	a.updateChecked = false
	a.updateErr = ""
	a.pendingUpdate = nil
	a.updateReady = false
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		up, err := checkMygoUpdate(ctx)
		a.onUpdateUI(func() {
			a.applyUpdateCheck(up, err)
			a.invalidate()
		})
	}()
}

// installAvailableUpdate downloads the release found by the last check and
// stages it. On macOS the new bundle replaces the app as it quits; on
// Linux the files are replaced in place and the running process keeps
// the old image until the next relaunch.
func (a *app) installAvailableUpdate() {
	up := a.pendingUpdate
	if up == nil || a.updateInstalling || a.updateReady {
		return
	}
	a.updateInstalling = true
	a.updateErr = ""
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err := installMygoUpdate(ctx, up)
		a.onUpdateUI(func() {
			a.applyUpdateInstall(err)
			a.invalidate()
		})
	}()
}

// applyUpdateCheck lands a check result: a failed check is not a claim
// that the app is current.
func (a *app) applyUpdateCheck(up *updateRelease, err error) {
	a.updateChecking = false
	a.updateChecked = err == nil
	if err != nil {
		a.updateErr = updateErrorMessage(err)
		return
	}
	a.pendingUpdate = up
}

// applyUpdateInstall lands an install result.
func (a *app) applyUpdateInstall(err error) {
	a.updateInstalling = false
	if err != nil {
		a.updateErr = updateErrorMessage(err)
		return
	}
	a.updateErr = ""
	a.updateReady = true
}

// restartForUpdate relaunches the app after an update was staged. The quit
// sequence saves open files and detaches terminals first.
func (a *app) restartForUpdate() {
	if !a.updateReady {
		return
	}
	relaunch := a.updateRelaunchFn
	if relaunch == nil {
		relaunch = func() { mygo.App.Relaunch() }
	}
	relaunch()
}

func updateErrorMessage(err error) string {
	if errors.Is(err, mygo.ErrUpdatesDisabled) {
		return "This build cannot update itself. Install the app in a folder you can write, or update it with its package manager."
	}
	return err.Error()
}

// updateSummary is the line under the update buttons.
func (a *app) updateSummary() string {
	switch {
	case a.updateChecking:
		return "Checking GitHub for a signed release…"
	case a.updateInstalling:
		if a.pendingUpdate != nil && a.pendingUpdate.Version != "" {
			return fmt.Sprintf("Downloading and installing Mojian %s…", a.pendingUpdate.Version)
		}
		return "Downloading and installing the update…"
	case a.updateReady:
		if a.pendingUpdate != nil && a.pendingUpdate.Version != "" {
			return fmt.Sprintf("Mojian %s is ready. Restart to apply it.", a.pendingUpdate.Version)
		}
		return "The update is ready. Restart to apply it."
	case a.pendingUpdate != nil:
		if a.pendingUpdate.Version != "" {
			return fmt.Sprintf("Mojian %s is available.", a.pendingUpdate.Version)
		}
		return "A new version is available."
	case a.updateErr != "":
		return ""
	case a.updateChecked:
		return "You are running the latest version."
	}
	return ""
}

// onUpdateUI runs fn where the app's view is built. A test has no window
// and runs it directly.
func (a *app) onUpdateUI(fn func()) {
	switch {
	case a.win != nil:
		a.win.Update(fn)
	case a.settingsWin != nil:
		a.settingsWin.Update(fn)
	default:
		fn()
	}
}
