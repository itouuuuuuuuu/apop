package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/itouuuuuuuuu/apop/internal/config"
	"github.com/itouuuuuuuuu/apop/internal/export"
	"github.com/itouuuuuuuuu/apop/internal/onepassword"
	"github.com/itouuuuuuuuu/apop/internal/profile"
	"github.com/itouuuuuuuuu/apop/internal/selector"
	"github.com/itouuuuuuuuu/apop/internal/sts"
)

var version = "dev"

//go:embed shell/apop.sh
var shellInit string

var roleARNRegex = regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[\w+=,.@/-]+$`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", config.DefaultConfigPath(), "config file path")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("apop %s\n", version)
		return nil
	}

	args := flag.Args()

	if len(args) > 0 && args[0] == "init" {
		if len(args) > 1 && args[1] == "--config" {
			return generateConfig(*configPath)
		}
		fmt.Print(shellInit)
		return nil
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	creds, err := onepassword.GetCredentials(cfg)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Using credentials from 1Password")

	if len(args) > 0 {
		arg := args[0]
		if roleARNRegex.MatchString(arg) {
			return doAssumeRole(cfg, creds, arg, "", "")
		}
		return assumeRoleByProfile(cfg, creds, arg)
	}

	return interactiveMode(cfg, creds)
}

func interactiveMode(cfg *config.Config, creds *onepassword.Credentials) error {
	profiles, err := profile.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no profiles found in ~/.aws/config")
	}

	selected, err := selector.Select(profiles)
	if err != nil {
		return err
	}

	prof, err := profile.Find(profiles, selected)
	if err != nil {
		return err
	}

	if prof.RoleARN == "" {
		return fmt.Errorf("profile '%s' has no role_arn configured", selected)
	}

	mfaSerial := creds.MFASerial
	if mfaSerial == "" {
		mfaSerial = prof.MFASerial
	}

	return doAssumeRole(cfg, creds, prof.RoleARN, selected, mfaSerial)
}

func assumeRoleByProfile(cfg *config.Config, creds *onepassword.Credentials, profileName string) error {
	profiles, err := profile.List()
	if err != nil {
		return err
	}

	prof, err := profile.Find(profiles, profileName)
	if err != nil {
		return err
	}

	if prof.RoleARN == "" {
		return fmt.Errorf("profile '%s' has no role_arn configured", profileName)
	}

	mfaSerial := creds.MFASerial
	if mfaSerial == "" {
		mfaSerial = prof.MFASerial
	}

	return doAssumeRole(cfg, creds, prof.RoleARN, profileName, mfaSerial)
}

func doAssumeRole(cfg *config.Config, creds *onepassword.Credentials, roleARN, profileName, mfaSerial string) error {
	if mfaSerial == "" {
		mfaSerial = creds.MFASerial
	}

	var totpCode string
	if mfaSerial != "" {
		var err error
		totpCode, err = onepassword.GetTOTPWithWait(cfg)
		if err != nil {
			return fmt.Errorf("failed to get TOTP: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Using TOTP from 1Password")
	}

	sessionName := "apop"
	if profileName != "" {
		sessionName = "apop-" + profileName
	} else {
		sessionName = fmt.Sprintf("apop-%d", os.Getpid())
	}

	input := &sts.AssumeRoleInput{
		RoleARN:     roleARN,
		SessionName: sessionName,
		MFASerial:   mfaSerial,
		TOTPCode:    totpCode,
		Credentials: &sts.Credentials{
			AccessKeyID:    creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
		},
	}

	result, err := sts.AssumeRole(input)
	if err != nil {
		return err
	}

	if err := export.WriteCredentialsFile(cfg.CredentialsFile, result, cfg.AWSRegion, profileName, roleARN); err != nil {
		return err
	}

	roleName := extractRoleName(roleARN)
	if profileName != "" {
		fmt.Fprintf(os.Stderr, "Successfully assumed role for profile: %s\n", profileName)
	} else {
		fmt.Fprintf(os.Stderr, "Successfully assumed role: %s\n", roleName)
	}

	cleanEnv := sts.CleanAWSEnv()
	sts.ShowCallerIdentity(result, cleanEnv)
	return nil
}

func generateConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(config.SampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Config file created: %s\nPlease edit it with your settings.\n", path)
	return nil
}

func extractRoleName(arn string) string {
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
}
