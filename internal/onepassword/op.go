package onepassword

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/itouuuuuuuuu/apop/internal/config"
)

type Credentials struct {
	AccessKeyID    string
	SecretAccessKey string
	MFASerial      string
}

type opField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func CheckOPInstalled() error {
	if _, err := exec.LookPath("op"); err != nil {
		return fmt.Errorf("1Password CLI (op) is not installed\nRun: brew install 1password-cli")
	}
	return nil
}

func GetCredentials(cfg *config.Config) (*Credentials, error) {
	if err := CheckOPInstalled(); err != nil {
		return nil, err
	}

	fields := fmt.Sprintf("%s,%s,%s",
		cfg.OPFields.AccessKeyID,
		cfg.OPFields.SecretAccessKey,
		cfg.OPFields.MFASerial,
	)

	cmd := exec.Command("op", "item", "get", cfg.OPItemName,
		"--fields", fields, "--reveal", "--format", "json")
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from 1Password: %w\nRun: eval \"$(op signin)\"", err)
	}

	var opFields []opField
	if err := json.Unmarshal(out, &opFields); err != nil {
		return nil, fmt.Errorf("failed to parse 1Password response: %w", err)
	}

	getValue := func(label string) string {
		for _, f := range opFields {
			if f.Label == label {
				return f.Value
			}
		}
		return ""
	}

	creds := &Credentials{
		AccessKeyID:    getValue(cfg.OPFields.AccessKeyID),
		SecretAccessKey: getValue(cfg.OPFields.SecretAccessKey),
		MFASerial:      getValue(cfg.OPFields.MFASerial),
	}

	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return nil, fmt.Errorf("AWS credentials not found in 1Password")
	}

	return creds, nil
}

func getTOTP(cfg *config.Config) (string, error) {
	cmd := exec.Command("op", "item", "get", cfg.OPItemName, "--otp")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get TOTP from 1Password: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func GetTOTPWithWait(cfg *config.Config) (string, error) {
	totp, err := getTOTP(cfg)
	if err != nil {
		return "", err
	}

	lastTOTP := readLastTOTP(cfg.LastTOTPFile)

	if totp != lastTOTP {
		saveLastTOTP(cfg.LastTOTPFile, totp)
		return totp, nil
	}

	// Wait for next TOTP window: calculate exact seconds remaining
	now := time.Now().Unix()
	waitSec := 30 - (now % 30) + 1
	fmt.Fprintf(os.Stderr, "TOTP code was already used. Waiting for next code (%d seconds)...\n", waitSec)

	time.Sleep(time.Duration(waitSec) * time.Second)

	totp, err = getTOTP(cfg)
	if err != nil {
		return "", err
	}

	if totp == lastTOTP {
		return "", fmt.Errorf("failed to get new TOTP code")
	}

	saveLastTOTP(cfg.LastTOTPFile, totp)
	return totp, nil
}

func readLastTOTP(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveLastTOTP(path, totp string) {
	_ = os.WriteFile(path, []byte(totp), 0600)
}
