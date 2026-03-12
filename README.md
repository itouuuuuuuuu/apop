# apop

AWS Profile switcher with 1Password integration for macOS.

Switches AWS profiles using credentials stored in 1Password. No AWS secrets are stored locally.

## Prerequisites

- macOS
- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`)
- [AWS CLI](https://aws.amazon.com/cli/) (`aws`)

## Installation

```bash
brew install itouuuuuuuuu/tap/apop
```

This automatically installs `jq` and `fzf` as dependencies. You also need to install the following manually:

```bash
brew install awscli
brew install --cask 1password-cli
```

Then add the following to your `~/.zshrc`:

```bash
source "$(brew --prefix)/share/apop/apop.sh"
```

Reload your shell:

```bash
source ~/.zshrc
```

## Setup

Generate a sample config and edit it:

```bash
apop init
$EDITOR ~/.config/apop/config
```

## Configuration

Config file: `~/.config/apop/config`

| Variable | Description | Default |
|---|---|---|
| `APOP_OP_ITEM_NAME` | 1Password item name containing AWS credentials | (required) |
| `APOP_AWS_REGION` | AWS region | (required) |
| `APOP_OP_FIELD_ACCESS_KEY_ID` | 1Password field label for Access Key ID | `aws_access_key_id` |
| `APOP_OP_FIELD_SECRET_ACCESS_KEY` | 1Password field label for Secret Access Key | `aws_secret_access_key` |
| `APOP_OP_FIELD_MFA_SERIAL` | 1Password field label for MFA Serial | `mfa_serial` |

## Usage

```bash
# Interactive profile selection with fzf
apop

# Specify a profile name directly
apop my-profile

# Specify a Role ARN directly
apop arn:aws:iam::123456789012:role/MyRole

# Copy credentials to clipboard after assuming role
apop -c
apop -c my-profile

# Role chaining (assume another role using current session credentials)
apop -r arn:aws:iam::999999999999:role/CrossAccountRole

# Role chaining + copy to clipboard
apop -c -r arn:aws:iam::999999999999:role/CrossAccountRole

# Show help
apop --help

# Show version
apop --version
```

## How It Works

1. Fetches AWS credentials (Access Key, Secret Key, MFA Serial) from 1Password
2. Selects a profile (fzf / direct name / ARN)
3. If MFA is required, retrieves TOTP from 1Password
4. Calls `aws sts assume-role` to obtain temporary credentials
5. Exports credentials as environment variables in the current shell

### Role Chaining

Use the `-r` option to chain-assume another role using your current session credentials (no 1Password needed).
This is useful for cross-account access where you need to assume a role from an already-assumed role.

```bash
# First, assume a role as usual
apop my-profile

# Then chain to another account's role
apop -r arn:aws:iam::999999999999:role/CrossAccountRole
```

## License

[MIT](LICENSE)
