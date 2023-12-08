package main

import (
	"twos.dev/winter/cmd"
)

// version is overridden by ldflags in the Makefile.
var version = "development"

func main() {
	if version == "" {
		// Catch the case where -ldflags="-X 'main.version='" is passed
		// (e.g. due to missing env var).
		version = "development"
	}
	cmd.Execute(version)
}
