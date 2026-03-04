package main

import (
	"os"

	"github.com/amakeenk/trs/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
