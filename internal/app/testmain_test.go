package app

import (
	"os"
	"testing"

	"mojian/internal/testutil"
)

func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }
