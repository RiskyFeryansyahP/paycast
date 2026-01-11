package config

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/RiskyFeryansyahP/paycast/internal/store"
	"github.com/RiskyFeryansyahP/paycast/pkg/logger"
	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configCmd = &cobra.Command{
	GroupID: "basic",
	Use:     "config",
	Short:   "Manage authentication contexts",
	Long:    "Configure and manage authentication contexts for different environments",
}

var configSetContextCmd = &cobra.Command{
	Use:   "set-context <name> <teleport-url>",
	Short: "Create or update a context configuration",
	Long:  "Authenticate to Teleport and save the context configuration for future use",
	Args:  cobra.ExactArgs(2),
	Run:   setContextRun,
}

var configDeleteContextCmd = &cobra.Command{
	Use:   "delete-context <name>",
	Short: "Delete a context configuration",
	Long:  "Remove the specified context from the configuration",
	Args:  cobra.ExactArgs(1),
	Run:   deleteContextRun,
}

var configUseContextCmd = &cobra.Command{
	Use:   "use-context <name>",
	Short: "Switch to a different context",
	Long:  "Set the specified context as the current active context",
	Args:  cobra.ExactArgs(1),
	Run:   useContextRun,
}

func NewConfigCommand() *cobra.Command {
	var proxy, auth, user string

	configSetContextCmd.Flags().StringVarP(&proxy, "proxy", "p", "", "Teleport proxy address")
	configSetContextCmd.Flags().StringVarP(&auth, "auth", "a", "", "Specify the name of authentication connector to use")
	configSetContextCmd.Flags().StringVarP(&user, "user", "u", "", "Teleport user, defaults to current local")
	_ = configSetContextCmd.MarkFlagRequired("proxy")
	_ = configSetContextCmd.MarkFlagRequired("auth")
	_ = configSetContextCmd.MarkFlagRequired("user")

	configCmd.AddCommand(configSetContextCmd, configDeleteContextCmd, configUseContextCmd)

	return configCmd
}

func setContextRun(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	proxyFlagVal := cobraCmd.Flag("proxy").Value.String()
	authFlagVal := cobraCmd.Flag("auth").Value.String()
	userFlagVal := cobraCmd.Flag("user").Value.String()

	contextName := args[0]
	teleportURL := args[1]

	proxy := fmt.Sprintf("--proxy=%s", proxyFlagVal)
	auth := fmt.Sprintf("--auth=%s", authFlagVal)
	user := fmt.Sprintf("--user=%s", userFlagVal)

	cmd := exec.Command("tsh", "login", proxy, auth, user, teleportURL)
	cmd.Env = append(os.Environ(), "TERM=dumb")

	ptyF, err := pty.Start(cmd)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to start terminal session for tsh login")
	}
	defer ptyF.Close()

	scanner := bufio.NewScanner(ptyF)
	result := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println(line)

		if strings.Contains(line, "password") {
			password, err := term.ReadPassword(int(syscall.Stdin))

			if err != nil {
				logger.Fatal().
					Err(err).
					Msg("Failed to read password input")
			}

			fmt.Fprintln(ptyF, string(password))
			continue
		}

		if strings.Contains(line, "OTP") {
			otp, err := term.ReadPassword(int(syscall.Stdin))

			if err != nil {
				logger.Fatal().
					Err(err).
					Msg("Failed to read OTP input")
			}

			fmt.Fprintln(ptyF, string(otp))
			continue
		}

		if !strings.Contains(line, "password") && !strings.Contains(line, "OTP") {
			if strings.Contains(line, "Profile") {
				profile := strings.TrimSpace(strings.Split(line, ": ")[1])
				result["profile"] = profile
				continue
			}

			if strings.Contains(line, "Cluster") {
				cluster := strings.TrimSpace(strings.Split(line, ": ")[1])
				result["cluster"] = cluster
				continue
			}

			if strings.Contains(line, "Valid") {
				valid := strings.TrimSpace(strings.Split(line, ": ")[1])
				validTime := strings.Split(valid, " ")[0:2]
				result["valid"] = strings.Join(validTime, " ")
				continue
			}
		}
	}

	err = cmd.Wait()

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("tsh login command failed. Please check your credentials and try again")
	}

	isExists, err := store.IsExist(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to check configuration file")
	}

	expiry := result["valid"]
	expiryTime, err := time.Parse(time.DateTime, expiry)

	if err != nil {
		logger.Fatal().
			Err(err).
			Str("expiry", expiry).
			Msg("Failed to parse session expiry time from tsh output")
	}

	if !isExists {
		config := store.Config{
			CurrentContext: contextName,
			Contexts: map[string]store.Context{
				contextName: {
					Name:    contextName,
					Cluster: result["cluster"],
					Profile: result["profile"],
					Expiry:  expiryTime,
					Proxy:   proxyFlagVal,
					Auth:    authFlagVal,
					User:    userFlagVal,
				},
			},
		}

		err = store.New(ctx, config)

		if err != nil {
			logger.Fatal().
				Err(err).
				Msg("Failed to create configuration file")
		}

		logger.Info().
			Str("context", contextName).
			Str("cluster", result["cluster"]).
			Msg("Context created successfully")
		return
	}

	config, err := store.Get(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to load configuration file")
	}

	config.CurrentContext = contextName
	config.Contexts[contextName] = store.Context{
		Name:    contextName,
		Cluster: result["cluster"],
		Profile: result["profile"],
		Expiry:  expiryTime,
		Proxy:   proxyFlagVal,
		Auth:    authFlagVal,
		User:    userFlagVal,
	}

	err = store.Save(ctx, config)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to save configuration file")
	}

	logger.Info().
		Str("context", contextName).
		Str("cluster", result["cluster"]).
		Msg("Context updated successfully")
}

func deleteContextRun(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	exist, err := store.IsExist(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to check configuration file")
	}

	if !exist {
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

	contextName := args[0]

	_, ok := config.Contexts[contextName]

	if !ok {
		logger.Fatal().
			Err(fmt.Errorf("context '%s' not found", contextName)).
			Msg("Run 'paycast config set-context' to see available contexts")
	}

	delete(config.Contexts, contextName)

	err = store.Save(ctx, config)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to save configuration file")
	}

	logger.Info().
		Str("context", contextName).
		Msg("Context deleted successfully")
}

func useContextRun(cobraCmd *cobra.Command, args []string) {
	ctx := cobraCmd.Context()

	exist, err := store.IsExist(ctx)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to check configuration file")
	}

	if !exist {
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

	contextName := args[0]

	_, ok := config.Contexts[contextName]

	if !ok {
		logger.Fatal().
			Err(fmt.Errorf("context '%s' not found", contextName)).
			Msg("Run 'paycast config set-context' to see available contexts")
	}

	config.CurrentContext = contextName
	err = store.Save(ctx, config)

	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to save configuration file")
	}

	logger.Info().
		Str("context", contextName).
		Msg("Switched to context successfully")
}
