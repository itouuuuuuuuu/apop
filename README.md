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

# Open AWS Management Console in browser (uses current session, or selects profile interactively)
apop -b

# Assume a specific profile and open console in browser
apop -b my-profile

# Unset all environment variables set by apop in the current shell
apop -u
apop --unset

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

### Browser Console

Use the `-b` option to open the AWS Management Console in your default browser.
This uses the AWS Federation sign-in endpoint to generate a pre-authenticated console URL.

- If you already have an active session, `apop -b` opens the console directly.
- If no session is active, `apop -b` presents an interactive profile selector (fzf), assumes the selected role, and then opens the console.
- You can also specify a profile directly: `apop -b my-profile`.

```bash
# Open console with current session (or select profile interactively if no session)
apop -b

# Assume a specific profile and open console in one step
apop -b my-profile
```

### Unsetting Credentials

Use `-u` (or `--unset`) to clear every environment variable apop sets in the current shell. Useful when switching to a context that should not see apop-managed credentials.

The following are unset:

- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
- `AWS_REGION`
- `AWS_ASSUMED_ROLE_ARN`
- `AWS_PROFILE`, `AWS_DEFAULT_PROFILE`
- `_APOP_LAST_TOTP_WINDOW` (apop's internal TOTP-window cache)

Variables apop never touches (e.g. `AWS_DEFAULT_REGION`, `AWS_SECURITY_TOKEN`, `APOP_*`) are left alone. Pre-existing values of the same names (for example an `AWS_REGION` you exported before running apop) are **not** restored — they are unset, since apop overwrote them when assuming a role.

```bash
apop -u
# apop session credentials cleared
```

`-u` cannot be combined with any other option or argument; doing so returns an error.

## License

[MIT](LICENSE)
