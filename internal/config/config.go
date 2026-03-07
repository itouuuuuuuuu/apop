package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type OPFields struct {
	AccessKeyID    string `toml:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key"`
	MFASerial      string `toml:"mfa_serial"`
}

type Config struct {
	OPItemName      string   `toml:"op_item_name"`
	AWSRegion       string   `toml:"aws_region"`
	CredentialsFile string   `toml:"credentials_file"`
	LastTOTPFile    string   `toml:"last_totp_file"`
	OPFields        OPFields `toml:"op_fields"`
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		return filepath.Join(homeDir(), path[2:])
	}
	return path
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s\nRun 'apop init --config' to generate a sample config", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	hint := fmt.Sprintf("\nPlease edit your config: $EDITOR %s", path)

	if cfg.OPItemName == "" {
		return nil, fmt.Errorf("op_item_name is required in %s%s", path, hint)
	}
	if cfg.AWSRegion == "" {
		return nil, fmt.Errorf("aws_region is required in %s%s", path, hint)
	}
	if cfg.CredentialsFile == "" {
		return nil, fmt.Errorf("credentials_file is required in %s%s", path, hint)
	}
	if cfg.LastTOTPFile == "" {
		return nil, fmt.Errorf("last_totp_file is required in %s%s", path, hint)
	}
	if cfg.OPFields.AccessKeyID == "" || cfg.OPFields.SecretAccessKey == "" || cfg.OPFields.MFASerial == "" {
		return nil, fmt.Errorf("all op_fields (access_key_id, secret_access_key, mfa_serial) are required in %s%s", path, hint)
	}

	cfg.CredentialsFile = expandHome(cfg.CredentialsFile)
	cfg.LastTOTPFile = expandHome(cfg.LastTOTPFile)

	return &cfg, nil
}

func DefaultConfigPath() string {
	return filepath.Join(homeDir(), ".config", "apop", "config.toml")
}

const SampleConfig = `op_item_name = ""
aws_region = ""
credentials_file = "~/.apop-credentials"
last_totp_file = "~/.apop-last-totp"

[op_fields]
access_key_id = ""
secret_access_key = ""
mfa_serial = ""
`
