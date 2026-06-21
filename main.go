package main

import (
	"os"

	"github.com/RiskyFeryansyahP/paycast/internal/cmd"
)

func main() {
	err := cmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
