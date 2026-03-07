package export

import (
	"fmt"
	"os"

	"github.com/itouuuuuuuuu/apop/internal/sts"
)

func WriteCredentialsFile(path string, creds *sts.AssumeRoleOutput, region, profileName, roleARN string) error {
	content := fmt.Sprintf("export AWS_ACCESS_KEY_ID='%s'\n", creds.AccessKeyID)
	content += fmt.Sprintf("export AWS_SECRET_ACCESS_KEY='%s'\n", creds.SecretAccessKey)
	content += fmt.Sprintf("export AWS_SESSION_TOKEN='%s'\n", creds.SessionToken)
	content += fmt.Sprintf("export AWS_REGION='%s'\n", region)

	if profileName != "" {
		content += fmt.Sprintf("export AWS_PROFILE='%s'\n", profileName)
		content += fmt.Sprintf("export AWS_DEFAULT_PROFILE='%s'\n", profileName)
	}
	if roleARN != "" {
		content += fmt.Sprintf("export AWS_ASSUMED_ROLE_ARN='%s'\n", roleARN)
	}

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	return nil
}
