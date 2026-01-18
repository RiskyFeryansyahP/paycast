package database

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/RiskyFeryansyahP/paycast/internal/store"
	"github.com/RiskyFeryansyahP/paycast/pkg/cmd"
	"github.com/RiskyFeryansyahP/paycast/pkg/logger"
	"github.com/creack/pty"
	"github.com/spf13/cobra"
)

var databaseCmd = &cobra.Command{
	GroupID: "basic",
	Use:     "db",
	Short:   "Manage database proxy configurations",
	Long:    "Add, remove, and run database proxies for the current context",
}

var dbRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start all configured database proxies",
	Long:  "Start database proxy connections for all databases configured in the current context",
	Run:   dbRun,
}

var dbAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new database proxy configuration",
	Long:  "Configure a new database proxy to be managed by paycast",
	Run:   dbAddRun,
}

var dbDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove a database proxy configuration",
	Long:  "Delete the specified database configuration from the current context",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Warn().Msg("This command is not yet implemented")
	},
}

func NewConfigCommand() *cobra.Command {
	var dbUser, dbName, tunnel string
	var port int32

	dbAddCmd.Flags().StringVarP(&dbUser, "db-user", "u", "", "Database user to log in as")
	dbAddCmd.Flags().StringVarP(&dbName, "db-name", "n", "", "Database name to log in to")
	dbAddCmd.Flags().StringVarP(&tunnel, "tunnel", "", "", "Open authenticated tunnel using database's client certificate so clients don't need to authenticate")
	dbAddCmd.Flags().Int32VarP(&port, "port", "p", 0, "Specifies the source port used by proxy db listener")
	_ = dbAddCmd.MarkFlagRequired("db-user")
	_ = dbAddCmd.MarkFlagRequired("tunnel")
	_ = dbAddCmd.MarkFlagRequired("port")

	databaseCmd.AddCommand(dbAddCmd, dbDeleteCmd, dbRunCmd)

	return databaseCmd
}

func dbAddRun(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	dbUser := cobraCmd.Flag("db-user").Value.String()
	dbName := cobraCmd.Flag("db-name").Value.String()
	tunnel := cobraCmd.Flag("tunnel").Value.String()
	portStr := cobraCmd.Flag("port").Value.String()

	port, _ := strconv.Atoi(portStr)

	exists, err := store.IsExist(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to check configuration file")
	}

	if !exists {
		logger.Fatal().
			Err(store.ErrConfigNotFound).
			Send()
	}

	config, err := store.Get(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to load configuration file")
	}

	currentContext := config.CurrentContext

	if currentContext == "" {
		logger.Fatal().
			Err(store.ErrNoContext).
			Send()
	}

	configContext := config.Contexts[currentContext]

	if len(configContext.Database) == 0 {
		configContext.Database = make(map[string]store.Database)
	}

	configContext.Database[tunnel] = store.Database{
		User:   dbUser,
		Tunnel: tunnel,
		Name:   dbName,
		Port:   int32(port),
	}
	config.Contexts[currentContext] = configContext

	err = store.Save(ctx, config)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to save configuration file")
	}

	logger.Info().
		Str("tunnel", tunnel).
		Str("database", dbName).
		Int("port", port).
		Msg("Database added successfully")
}

func dbRun(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	exists, err := store.IsExist(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to check configuration file")
	}

	if !exists {
		logger.Fatal().
			Err(store.ErrConfigNotFound).
			Send()
	}

	config, err := store.Get(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to load configuration file")
	}

	currentContext := config.CurrentContext

	if currentContext == "" {
		logger.Fatal().
			Err(store.ErrNoContext).
			Send()
	}

	configContext := config.Contexts[currentContext]

	now := time.Now()

	if now.After(configContext.Expiry) {
		updatedConfigContext, err := cmd.Relogin(ctx, &configContext)

		if err != nil {
			logger.Fatal().
				Err(err).
				Msg("Failed to relogin to set context when context expired")
		}

		config.Contexts[currentContext] = *updatedConfigContext

		err = store.Save(ctx, config)

		if err != nil {
			logger.Fatal().
				Err(err).
				Msg("Failed to save configuration file")
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			for _, v := range configContext.Database {
				go func(db store.Database) {
					dbUser := fmt.Sprintf("--db-user=%s", db.User)
					dbName := fmt.Sprintf("--db-name=%s", db.Name)
					tunnel := fmt.Sprintf("--tunnel=%s", db.Tunnel)
					port := fmt.Sprintf("--port=%d", db.Port)

					cmd := exec.Command("tsh", "proxy", "db", dbUser, dbName, tunnel, port)
					cmd.Env = append(os.Environ(), "TERM=dumb")

					ptyF, err := pty.Start(cmd)

					if err != nil {
						logger.Fatal().
							Err(err).
							Str("database", v.Name).
							Msg("Failed to start database proxy")
					}
					defer ptyF.Close()

					logger.Info().
						Str("user", v.User).
						Str("database", v.Name).
						Int("port", int(v.Port)).
						Str("host", "localhost").
						Msg("Database proxy started")

					err = cmd.Wait()

					if err != nil {
						logger.Error().
							Err(err).
							Str("database", v.Name).
							Msg("Database proxy terminated with error")
					}
				}(v)
			}

			duration := configContext.Expiry.Sub(now)

			timer := time.NewTimer(duration)

			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				updatedConfigContext, err := cmd.Relogin(ctx, &configContext)

				if err != nil {
					logger.Fatal().
						Err(err).
						Msg("Failed to relogin to set context when context expired")
				}

				config.Contexts[currentContext] = *updatedConfigContext

				err = store.Save(ctx, config)

				if err != nil {
					logger.Fatal().
						Err(err).
						Msg("Failed to save configuration file")
				}
			}
		}
	}()

	// Block until we receive our signal.
	<-c

	logger.Info().Msg("Shutting down database proxies...")
}
