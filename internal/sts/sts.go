package sts

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Credentials struct {
	AccessKeyID    string
	SecretAccessKey string
}

type AssumeRoleInput struct {
	RoleARN     string
	SessionName string
	MFASerial   string
	TOTPCode    string
	Credentials *Credentials
}

type AssumeRoleOutput struct {
	AccessKeyID    string
	SecretAccessKey string
	SessionToken   string
}

type stsResponse struct {
	Credentials struct {
		AccessKeyId    string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		SessionToken   string `json:"SessionToken"`
	} `json:"Credentials"`
}

func AssumeRole(input *AssumeRoleInput) (*AssumeRoleOutput, error) {
	args := []string{
		"sts", "assume-role",
		"--role-arn", input.RoleARN,
		"--role-session-name", input.SessionName,
	}

	if input.MFASerial != "" && input.TOTPCode != "" {
		args = append(args, "--serial-number", input.MFASerial, "--token-code", input.TOTPCode)
	}

	cmd := exec.Command("aws", args...)

	env := cleanAWSEnv()
	if input.Credentials != nil {
		env = append(env,
			"AWS_ACCESS_KEY_ID="+input.Credentials.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY="+input.Credentials.SecretAccessKey,
		)
	}
	cmd.Env = env
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error assuming role: %w", err)
	}

	var resp stsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse STS response: %w", err)
	}

	return &AssumeRoleOutput{
		AccessKeyID:    resp.Credentials.AccessKeyId,
		SecretAccessKey: resp.Credentials.SecretAccessKey,
		SessionToken:   resp.Credentials.SessionToken,
	}, nil
}

func ShowCallerIdentity(creds *AssumeRoleOutput, baseEnv []string) {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--output", "table")
	cmd.Env = append(baseEnv,
		"AWS_ACCESS_KEY_ID="+creds.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY="+creds.SecretAccessKey,
		"AWS_SESSION_TOKEN="+creds.SessionToken,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func CleanAWSEnv() []string {
	return cleanAWSEnv()
}

func cleanAWSEnv() []string {
	removeKeys := map[string]bool{
		"AWS_ACCESS_KEY_ID":     true,
		"AWS_SECRET_ACCESS_KEY": true,
		"AWS_SESSION_TOKEN":     true,
		"AWS_SECURITY_TOKEN":    true,
	}

	var env []string
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if !removeKeys[key] {
			env = append(env, e)
		}
	}
	return env
}
