package main

import (
	"fmt"
	"os"

	"github.com/user/chore-scheduler/internal/cli"
)

func main() {
	app := cli.New()
	if err := app.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
