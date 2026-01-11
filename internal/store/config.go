package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DIR       = ".paycast"
	FILE_NAME = "config"
)

// Error messages
var (
	ErrConfigNotFound = fmt.Errorf("no paycast configuration found\nRun 'paycast config set-context <name> <url> --proxy=<proxy> --auth=<auth> --user=<user>' to get started")
	ErrNoContext      = fmt.Errorf("no context configured\nRun 'paycast config set-context <name> <url> --proxy=<proxy> --auth=<auth> --user=<user>' to create a context")
)

// GetConfigPath returns the full path to the config file
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, DIR, FILE_NAME)
}

type Context struct {
	Database map[string]Database `json:"dbs"`
	Name     string              `json:"name"`
	Cluster  string              `json:"cluster"`
	Profile  string              `json:"profile"`
	Proxy    string              `json:"proxy"`
	Auth     string              `json:"auth"`
	User     string              `json:"user"`
	Expiry   time.Time           `json:"expiry"`
}

type Database struct {
	User   string `json:"user"`
	Tunnel string `json:"tunnel"`
	Name   string `json:"name"`
	Port   int32  `json:"port"`
}

type Config struct {
	Contexts       map[string]Context `json:"contexts"`
	CurrentContext string             `json:"current_context"`
}
