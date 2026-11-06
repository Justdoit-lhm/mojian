# Tests

Go requires every `_test.go` file to live in the directory of the package it
tests, so white-box tests of the window stay beside `internal/app` and compile
with its unexported state. Putting those files here would turn them into a
different package that cannot reach that state.

This folder is for black-box integration tests and the helpers they share:
real folders, real Git repositories, and the CLI-facing behavior of the app.
`go test ./...` runs both suites.
