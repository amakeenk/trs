package main

import (
	"os"

	"altlinux.space/amakeenk/trs/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
