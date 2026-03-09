# apop

AWS Profile switcher with 1Password integration for macOS.

Switches AWS profiles using credentials stored in 1Password. No AWS secrets are stored locally.

## Prerequisites

- macOS
- [Homebrew](https://brew.sh/)
- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`)
- [AWS CLI](https://aws.amazon.com/cli/) (`aws`)
- [jq](https://jqlang.github.io/jq/) (`jq`)
- [fzf](https://github.com/junegunn/fzf) (optional, for interactive selection)

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/itouuuuuuuuu/apop.git ~/.apop
```

### 2. Install dependencies

```bash
brew install awscli jq fzf
brew install --cask 1password-cli
```

### 3. Add to your shell

Add the following to your `~/.zshrc`:

```bash
source ~/.apop/apop.sh
```

Then reload your shell:

```bash
source ~/.zshrc
```

## Setup

```bash
# Generate a sample config
apop init

# Edit with your settings
$EDITOR ~/.config/apop/config
```

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
```

## Configuration

`~/.config/apop/config`:

```bash
APOP_OP_ITEM_NAME="aws_profile"
APOP_AWS_REGION="ap-northeast-1"

# 1Password field labels (defaults shown)
# APOP_OP_FIELD_ACCESS_KEY_ID="aws_access_key_id"
# APOP_OP_FIELD_SECRET_ACCESS_KEY="aws_secret_access_key"
# APOP_OP_FIELD_MFA_SERIAL="mfa_serial"
```

## How It Works

1. Fetches AWS credentials (Access Key, Secret Key, MFA Serial) from 1Password
2. Selects a profile (fzf / direct name / ARN)
3. If MFA is required, retrieves TOTP from 1Password
4. Calls `aws sts assume-role` to obtain temporary credentials
5. Exports credentials as environment variables in the current shell
