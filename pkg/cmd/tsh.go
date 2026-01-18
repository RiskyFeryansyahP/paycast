package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/RiskyFeryansyahP/paycast/internal/store"
	"github.com/RiskyFeryansyahP/paycast/pkg/logger"
	"github.com/creack/pty"
	"golang.org/x/term"
)

func Relogin(ctx context.Context, configContext *store.Context) (*store.Context, error) {
	cmd := exec.Command("tsh", "login", configContext.Cluster)
	cmd.Env = append(os.Environ(), "TERM=dumb")

	ptyF, err := pty.Start(cmd)

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to start terminal session for tsh login")

		return nil, err
	}
	defer ptyF.Close()

	scanner := bufio.NewScanner(ptyF)

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println(line)

		if strings.Contains(line, "password") {
			password, err := term.ReadPassword(int(syscall.Stdin))

			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to read password input")

				return nil, err
			}

			fmt.Fprintln(ptyF, string(password))
			continue
		}

		if strings.Contains(line, "OTP") {
			otp, err := term.ReadPassword(int(syscall.Stdin))

			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to read OTP input")

				return nil, err
			}

			fmt.Fprintln(ptyF, string(otp))
			continue
		}

		if !strings.Contains(line, "password") && !strings.Contains(line, "OTP") {
			if strings.Contains(line, "Profile") {
				profile := strings.TrimSpace(strings.Split(line, ": ")[1])
				configContext.Profile = profile
				continue
			}

			if strings.Contains(line, "Cluster") {
				cluster := strings.TrimSpace(strings.Split(line, ": ")[1])
				configContext.Cluster = cluster
				continue
			}

			if strings.Contains(line, "Valid") {
				valid := strings.TrimSpace(strings.Split(line, ": ")[1])
				validTime := strings.Split(valid, " ")[0:2]
				expiry := strings.Join(validTime, " ")

				expiryTime, err := time.Parse(time.DateTime, expiry)

				if err != nil {
					logger.Error().
						Err(err).
						Str("expiry", expiry).
						Msg("Failed to parse session expiry time from tsh output")

					return nil, err
				}

				configContext.Expiry = expiryTime

				continue
			}
		}
	}

	err = cmd.Wait()

	if err != nil {
		logger.Error().
			Err(err).
			Msg("tsh login command failed. Please check your credentials and try again")

		return nil, err
	}

	configContext, err = Status(ctx, configContext)

	if err != nil {
		return nil, err
	}

	return configContext, nil
}

func Status(ctx context.Context, configContext *store.Context) (*store.Context, error) {
	cmd := exec.Command("tsh", "status")
	cmd.Env = append(os.Environ(), "TERM=dumb")

	ptyF, err := pty.Start(cmd)

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to start terminal session for tsh login")

		return nil, err
	}
	defer ptyF.Close()

	scanner := bufio.NewScanner(ptyF)

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println(line)

		if !strings.Contains(line, "password") && !strings.Contains(line, "OTP") {
			if strings.Contains(line, "Profile") {
				profile := strings.TrimSpace(strings.Split(line, ": ")[1])
				configContext.Profile = profile
				continue
			}

			if strings.Contains(line, "Cluster") {
				cluster := strings.TrimSpace(strings.Split(line, ": ")[1])
				configContext.Cluster = cluster
				continue
			}

			if strings.Contains(line, "Valid") {
				valid := strings.TrimSpace(strings.Split(line, ": ")[1])
				validTime := strings.Split(valid, " ")[0:2]
				expiry := strings.Join(validTime, " ")

				expiryTime, err := time.Parse(time.DateTime, expiry)

				if err != nil {
					logger.Error().
						Err(err).
						Str("expiry", expiry).
						Msg("Failed to parse session expiry time from tsh output")

					return nil, err
				}

				configContext.Expiry = expiryTime

				continue
			}
		}
	}

	err = cmd.Wait()

	if err != nil {
		logger.Error().
			Err(err).
			Msg("tsh login command failed. Please check your credentials and try again")

		return nil, err
	}

	return configContext, nil
}
