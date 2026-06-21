package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/RiskyFeryansyahP/paycast/internal/config"
	"github.com/RiskyFeryansyahP/paycast/internal/database"
	"github.com/spf13/cobra"
)

var (
	version = "dev"

	rootCmd = &cobra.Command{
		Use:   "paycast",
		Short: "Internal CLI tool for managing contexts and workflows",
		Long:  "Paycast helps manage authentication contexts, database proxies, and other development tools",
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version the CLI installed",
		Run: func(cmd *cobra.Command, args []string) {
			info, ok := debug.ReadBuildInfo()

			if ok && info.Main.Version != "" {
				fmt.Println(info.Main.Version)
				return
			}

			fmt.Println(version)
		},
	}
)

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	configCmd := config.NewConfigCommand()
	dbCmd := database.NewConfigCommand()

	rootCmd.AddGroup(&cobra.Group{ID: "basic", Title: "Basic Commands:"})
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd, dbCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
