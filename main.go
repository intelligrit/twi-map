package main

import (
	"os"

	"github.com/robertmeta/twi-map/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
