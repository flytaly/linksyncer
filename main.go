package main

import (
	_ "embed"

	"github.com/flytaly/linksyncer/cmd"
)

//go:embed version
var version string

func main() {
	cmd.Execute(version)
}
