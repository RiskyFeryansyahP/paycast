package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/RiskyFeryansyahP/paycast/pkg/logger"
)

func New(ctx context.Context, config Config) error {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get user home directory")

		return err
	}

	configDir := filepath.Join(homeDir, DIR)
	configFile := filepath.Join(configDir, FILE_NAME)

	err = os.MkdirAll(configDir, 0700)

	if err != nil {
		logger.Error().
			Err(err).
			Str("path", configDir).
			Msg("Failed to create configuration directory")

		return err
	}

	configData, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to serialize configuration data")

		return err
	}

	err = os.WriteFile(configFile, configData, 0600)

	if err != nil {
		logger.Error().
			Err(err).
			Str("path", configFile).
			Msg("Failed to write configuration file")

		return err
	}

	return nil
}

func Save(ctx context.Context, config Config) error {
	configFile, err := getConfigFile()

	if err != nil {
		return err
	}

	configData, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to serialize configuration data")

		return err
	}

	err = os.WriteFile(configFile, configData, 0600)

	if err != nil {
		logger.Error().
			Err(err).
			Str("path", configFile).
			Msg("Failed to write configuration file")

		return err
	}

	return nil
}

func Get(ctx context.Context) (Config, error) {
	configFile, err := getConfigFile()

	if err != nil {
		return Config{}, err
	}

	configData, err := os.ReadFile(configFile)

	if err != nil {
		return Config{}, err
	}

	var config Config

	err = json.Unmarshal(configData, &config)

	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func IsExist(ctx context.Context) (bool, error) {
	configFile, err := getConfigFile()

	if err != nil {
		return false, err
	}

	_, err = os.Stat(configFile)

	if err != nil && os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func getConfigFile() (string, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get user home directory")

		return "", err
	}

	configDir := filepath.Join(homeDir, DIR)
	configFile := filepath.Join(configDir, FILE_NAME)

	return configFile, nil
}
