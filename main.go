package main

import (
	"time"

	"github.com/cloud66-oss/geo/cmd"
	"github.com/getsentry/sentry-go"
)

func main() {
	// sentry is initialized in cmd/root.go after config is loaded;
	// flush here on exit to ensure any buffered events are sent
	defer sentry.Flush(2 * time.Second)

	cmd.Execute()
}
