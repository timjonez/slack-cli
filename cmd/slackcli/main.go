package main

import (
	"os"

	"github.com/timjonez/slack-cli/internal/cli"
)

func main() {
	app := cli.NewApp()
	os.Exit(app.Execute(os.Args[1:]))
}
