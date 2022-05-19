package main

import (
	"github.com/rsb/prolog/app/cli/prolog/cmd"
)

var build = "develop"

func main() {
	cmd.Execute(build)
}
