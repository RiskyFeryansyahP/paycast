package main

import (
	"log"

	"github.com/RiskyFeryansyahP/paycast/internal/config"
	"github.com/RiskyFeryansyahP/paycast/internal/database"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "paycast",
		Short: "Internal CLI tool for managing contexts and workflows",
		Long:  "Paycast helps manage authentication contexts, database proxies, and other development tools",
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	configCmd := config.NewConfigCommand()
	dbCmd := database.NewConfigCommand()

	rootCmd.AddGroup(&cobra.Group{ID: "basic", Title: "Basic Commands:"})
	rootCmd.AddCommand(configCmd, dbCmd)

	err := rootCmd.Execute()

	if err != nil {
		log.Fatal(err)
	}
}
