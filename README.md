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

## Upgrading from 1.x to 2.x

**Breaking change**: apop 2.0 no longer exports `AWS_PROFILE` / `AWS_DEFAULT_PROFILE` after assuming a role, and unsets them if they were already set in the shell. This fixes spurious assume-role failures with the Terraform AWS provider and other profile-aware tools (see [Environment Variables Set by apop](#environment-variables-set-by-apop) for the rationale).

Migrate any prompt or script that referenced `AWS_PROFILE` for *display* purposes to the new `APOP_PROFILE` variable:

```diff
- PS1='[${AWS_PROFILE:-no-profile}] '
+ PS1='[${APOP_PROFILE:-no-profile}] '
```

Tools that read `AWS_PROFILE` as an AWS profile-selection hint (`aws`, `terraform`, `boto3`, etc.) are unaffected — they will see only the session credentials apop has exported.

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

> The examples below use long options. Each option also has a short form — run `apop -h` to see the full list.

```bash
# Interactive profile selection with fzf
apop

# Specify a profile name directly
apop my-profile

# Specify a Role ARN directly
apop arn:aws:iam::123456789012:role/MyRole

# Copy credentials to clipboard after assuming role
apop --copy
apop --copy my-profile

# Role chaining (assume another role using current session credentials)
apop --role-chain arn:aws:iam::999999999999:role/CrossAccountRole

# Role chaining + copy to clipboard
apop --copy --role-chain arn:aws:iam::999999999999:role/CrossAccountRole

# Open AWS Management Console in browser (uses current session, or selects profile interactively)
apop --browse

# Assume a specific profile and open console in browser
apop --browse my-profile

# Unset all environment variables set by apop in the current shell
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

### Environment Variables Set by apop

After a successful assume-role, apop exports:

- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` — the temporary STS credentials
- `AWS_REGION` — from `APOP_AWS_REGION`
- `AWS_ASSUMED_ROLE_ARN` — the role ARN that was assumed
- `APOP_PROFILE` — the profile name (only when a profile was used; cleared on direct ARN assumption)

apop deliberately does **not** export `AWS_PROFILE` / `AWS_DEFAULT_PROFILE`, and unsets them if they were inherited. AWS SDK Go v2's documented credential chain prefers environment-variable credentials, but some profile-aware tools — notably the Terraform AWS provider and the S3 backend — read `AWS_PROFILE` as a profile-selection hint and re-resolve `role_arn` / `source_profile` from `~/.aws/config`, which then fails because the source profile usually has no credentials of its own. Typical symptom:

```
Error: failed to load assume role arn:aws:iam::...:role/..., of profile default, <nil>
```

Clearing the two variables removes the ambiguity: every downstream tool sees exactly one credential context (the env vars apop just exported).

If you want to show the current profile in your prompt or in scripts, use `APOP_PROFILE` (e.g. `PS1='[$APOP_PROFILE] '`).

### Role Chaining

Use the `--role-chain` option to chain-assume another role using your current session credentials (no 1Password needed).
This is useful for cross-account access where you need to assume a role from an already-assumed role.

```bash
# First, assume a role as usual
apop my-profile

# Then chain to another account's role
apop --role-chain arn:aws:iam::999999999999:role/CrossAccountRole
```

### Browser Console

Use the `--browse` option to open the AWS Management Console in your default browser.
This uses the AWS Federation sign-in endpoint to generate a pre-authenticated console URL.

- If you already have an active session, `apop --browse` opens the console directly.
- If no session is active, `apop --browse` presents an interactive profile selector (fzf), assumes the selected role, and then opens the console.
- You can also specify a profile directly: `apop --browse my-profile`.

```bash
# Open console with current session (or select profile interactively if no session)
apop --browse

# Assume a specific profile and open console in one step
apop --browse my-profile
```

### Unsetting Credentials

Use `--unset` to clear every environment variable apop sets in the current shell. Useful when switching to a context that should not see apop-managed credentials.

The following are unset:

- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
- `AWS_REGION`
- `AWS_ASSUMED_ROLE_ARN`
- `AWS_PROFILE`, `AWS_DEFAULT_PROFILE` (also cleared by every assume-role; see "Environment Variables Set by apop")
- `APOP_PROFILE`
- `_APOP_LAST_TOTP_WINDOW` (apop's internal TOTP-window cache)

Variables apop never touches (e.g. `AWS_DEFAULT_REGION`, `AWS_SECURITY_TOKEN`, other `APOP_*` config vars) are left alone. Pre-existing values of the same names (for example an `AWS_REGION` you exported before running apop) are **not** restored — they are unset, since apop overwrote them when assuming a role.

```bash
apop --unset
# apop session credentials cleared
```

`--unset` cannot be combined with any other option or argument; doing so returns an error.

## License

[MIT](LICENSE)
