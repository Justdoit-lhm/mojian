package update

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func TestUpdateDisabledMessage(t *testing.T) {
	got := ErrorMessage(mygo.ErrUpdatesDisabled)
	if !strings.Contains(got, "cannot update itself") {
		t.Fatalf("disabled update message: %q", got)
	}
}
