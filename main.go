// Mojian is a quiet text editor in MyGo's native UI. The app lives in
// internal/app; this file only starts it.
//
//	go tool mygo dev    the app, rebuilt as the code changes
//	go test ./...       the model, the store and the view, without a window
package main

import "mojian/internal/app"

func main() { app.Run() }
