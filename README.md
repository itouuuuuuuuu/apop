# apop

AWS Profile switcher with 1Password integration.

A CLI tool that switches AWS profiles using credentials stored in 1Password.

## Prerequisites

- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`)
- [AWS CLI](https://aws.amazon.com/cli/) (`aws`)
- [fzf](https://github.com/junegunn/fzf) (optional, for interactive selection)

## Installation

```bash
go install github.com/itouuuuuuuuu/apop@latest
```

Or build from source:

```bash
go build -o apop .
sudo mv apop /usr/local/bin/
```

## Shell Integration

Add the following to your `.zshrc` or `.bashrc`:

```bash
eval "$(command apop init)"
```

This registers `apop` as a shell function so that credentials obtained via AssumeRole are automatically exported into your current shell session.

## Usage

```bash
# Interactive profile selection with fzf
apop

# Specify a profile name directly
apop my-profile

# Specify a Role ARN directly
apop arn:aws:iam::123456789012:role/MyRole
```

## Configuration

`~/.config/apop/config.toml`:

```toml
op_item_name = "awsp"
aws_region = "ap-northeast-1"
credentials_file = "~/.apop-credentials"
last_totp_file = "~/.apop-last-totp"

[op_fields]
access_key_id = "aws_access_key_id"
secret_access_key = "aws_secret_access_key"
mfa_serial = "mfa_serial"
```

If the config file does not exist, the defaults shown above are used.

## How It Works

1. Fetches AWS credentials (Access Key, Secret Key, MFA Serial) from 1Password in a single call
2. Selects a profile (fzf / direct name / ARN)
3. If MFA is required, automatically retrieves TOTP from 1Password (waits for the next code if already used)
4. Calls `aws sts assume-role` to obtain temporary credentials
5. Writes credentials to a file, which the shell function `source`s to export environment variables

## Flags

```
--config <path>    Config file path (default: ~/.config/apop/config.toml)
--version          Show version
--help             Show help
```
