package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type Profile struct {
	Name          string
	RoleARN       string
	MFASerial     string
	SourceProfile string
}

func awsConfigPath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".aws", "config")
}

func List() ([]Profile, error) {
	cfg, err := ini.Load(awsConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read ~/.aws/config: %w", err)
	}

	var profiles []Profile
	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == "DEFAULT" {
			continue
		}

		profileName := strings.TrimPrefix(name, "profile ")

		p := Profile{
			Name:          profileName,
			RoleARN:       section.Key("role_arn").String(),
			MFASerial:     section.Key("mfa_serial").String(),
			SourceProfile: section.Key("source_profile").String(),
		}
		profiles = append(profiles, p)
	}

	return profiles, nil
}

func Find(profiles []Profile, name string) (*Profile, error) {
	for _, p := range profiles {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("profile '%s' not found", name)
}

func CurrentProfile() string {
	if p := os.Getenv("AWS_PROFILE"); p != "" {
		return p
	}
	if p := os.Getenv("AWS_DEFAULT_PROFILE"); p != "" {
		return p
	}
	return "default"
}
