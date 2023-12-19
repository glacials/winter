package main

import (
	"runtime/debug"

	"twos.dev/winter/cmd"
)

func main() {
	version := "development"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}
	cmd.Execute(version)
}
