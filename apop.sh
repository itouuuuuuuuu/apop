#!/bin/bash
# apop - AWS Profile switcher with 1Password
# Source this file in your .zshrc or .bashrc:
#   source /path/to/apop.sh

APOP_VERSION="1.0.0"
APOP_CONFIG="${APOP_CONFIG:-$HOME/.config/apop/config}"

apop() {
  # Parse options
  local _apop_copy_to_clipboard=false
  local _apop_role_chain_arn=""
  local args=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      init|--version|--help|-h)
        args+=("$1")
        shift
        ;;
      -c)
        _apop_copy_to_clipboard=true
        shift
        ;;
      -r)
        if [[ -z "${2:-}" ]]; then
          echo "Error: -r requires a role ARN argument" >&2
          return 1
        fi
        _apop_role_chain_arn="$2"
        shift 2
        ;;
      *)
        args+=("$1")
        shift
        ;;
    esac
  done
  set -- "${args[@]}"

  case "${1:-}" in
    init)
      _apop_init
      return
      ;;
    --version)
      echo "apop $APOP_VERSION"
      return
      ;;
    --help|-h)
      _apop_usage
      return
      ;;
  esac

  # Load config
  if [[ ! -f "$APOP_CONFIG" ]]; then
    echo "Error: config file not found: $APOP_CONFIG" >&2
    echo "Create it: apop init" >&2
    return 1
  fi
  source "$APOP_CONFIG"

  # Trim whitespace from config values
  APOP_OP_ITEM_NAME="${APOP_OP_ITEM_NAME## }"; APOP_OP_ITEM_NAME="${APOP_OP_ITEM_NAME%% }"
  APOP_AWS_REGION="${APOP_AWS_REGION## }"; APOP_AWS_REGION="${APOP_AWS_REGION%% }"

  # Validate config
  if [[ -z "${APOP_OP_ITEM_NAME:-}" ]]; then
    echo "Error: APOP_OP_ITEM_NAME is required in $APOP_CONFIG" >&2
    echo "Edit the config: \$EDITOR $APOP_CONFIG" >&2
    return 1
  fi
  if [[ -z "${APOP_AWS_REGION:-}" ]]; then
    echo "Error: APOP_AWS_REGION is required in $APOP_CONFIG" >&2
    echo "Edit the config: \$EDITOR $APOP_CONFIG" >&2
    return 1
  fi

  # Role chaining: use current session credentials to assume another role
  if [[ -n "${_apop_role_chain_arn:-}" ]]; then
    _apop_chain_role "$_apop_role_chain_arn"
    return
  fi

  # Determine role (credentials fetched lazily after profile selection)
  if [[ $# -gt 0 ]]; then
    if [[ "$1" =~ ^arn:aws:iam::[0-9]{12}:role/ ]]; then
      _apop_assume_role "$1" ""
    else
      _apop_assume_profile "$1"
    fi
  else
    _apop_interactive
  fi
}

_apop_usage() {
  cat >&2 <<EOF
Usage: apop [-c] [-r role-arn] [profile-name | role-arn]

Options:
  -c           Copy credentials to clipboard after assuming role
  -r role-arn  Chain-assume a role using current session credentials

Commands:
  init         Generate sample config file
  --version    Show version
  --help       Show this help

Examples:
  apop                                                    # Interactive selection with fzf
  apop my-profile                                         # Direct profile switch
  apop arn:aws:iam::123456789012:role/MyRole               # Direct ARN assumption
  apop -c                                                 # Interactive + copy to clipboard
  apop -c my-profile                                      # Direct switch + copy to clipboard
  apop -r arn:aws:iam::999999999999:role/CrossRole         # Role chaining
  apop -c -r arn:aws:iam::999999999999:role/CrossRole      # Role chaining + clipboard
EOF
}

_apop_init() {
  if [[ -f "$APOP_CONFIG" ]]; then
    echo "Error: config file already exists: $APOP_CONFIG" >&2
    echo "Edit it: \$EDITOR $APOP_CONFIG" >&2
    return 1
  fi
  mkdir -p "$(dirname "$APOP_CONFIG")"
  cat > "$APOP_CONFIG" <<'CONF'
APOP_OP_ITEM_NAME=""
APOP_AWS_REGION=""

# 1Password field labels (defaults shown)
# APOP_OP_FIELD_ACCESS_KEY_ID="aws_access_key_id"
# APOP_OP_FIELD_SECRET_ACCESS_KEY="aws_secret_access_key"
# APOP_OP_FIELD_MFA_SERIAL="mfa_serial"
CONF
  echo "Config file created: $APOP_CONFIG" >&2
  echo "Edit it: \$EDITOR $APOP_CONFIG" >&2
}

_apop_interactive() {
  local profiles
  profiles=$(_apop_list_profiles)
  if [[ -z "$profiles" ]]; then
    echo "Error: no profiles found in ~/.aws/config" >&2
    echo "Add a profile: \$EDITOR ~/.aws/config" >&2
    return 1
  fi

  local selected
  selected=$(_apop_select_profile "$profiles")
  if [[ -z "$selected" ]]; then
    echo "Error: profile selection cancelled" >&2
    return 1
  fi

  _apop_assume_profile "$selected"
}

_apop_assume_profile() {
  local profile_name="$1"

  local aws_config="${AWS_CONFIG_FILE:-$HOME/.aws/config}"
  local role_arn mfa_serial_from_config
  eval "$(_apop_get_profile_values "$aws_config" "$profile_name" role_arn mfa_serial)"

  if [[ -z "$role_arn" ]]; then
    echo "Error: profile '$profile_name' has no role_arn configured" >&2
    echo "Add role_arn: \$EDITOR $aws_config" >&2
    return 1
  fi

  _apop_assume_role "$role_arn" "$profile_name" "$mfa_serial"
}

_apop_assume_role() {
  local role_arn="$1" profile_name="$2" config_mfa_serial="${3:-}"

  _apop_check_deps op:1password-cli aws:awscli jq:jq || return 1

  # Get credentials from 1Password
  local op_fields="${APOP_OP_FIELD_ACCESS_KEY_ID:-aws_access_key_id},${APOP_OP_FIELD_SECRET_ACCESS_KEY:-aws_secret_access_key},${APOP_OP_FIELD_MFA_SERIAL:-mfa_serial}"
  local op_json
  if ! op_json=$(OP_BIOMETRIC_UNLOCK_ENABLED=true op item get "$APOP_OP_ITEM_NAME" --fields "$op_fields" --reveal --format json 2>&1); then
    echo "Error: failed to get credentials from 1Password" >&2
    echo "$op_json" >&2
    return 1
  fi
  echo "Using credentials from 1Password" >&2

  # Parse all credential fields in one jq call
  local access_key_id secret_access_key mfa_serial
  local field_ak="${APOP_OP_FIELD_ACCESS_KEY_ID:-aws_access_key_id}"
  local field_sk="${APOP_OP_FIELD_SECRET_ACCESS_KEY:-aws_secret_access_key}"
  local field_mfa="${APOP_OP_FIELD_MFA_SERIAL:-mfa_serial}"

  eval "$(echo "$op_json" | jq -r --arg ak "$field_ak" --arg sk "$field_sk" --arg mfa "$field_mfa" '
    reduce .[] as $f ({};
      if $f.label == $ak then .ak = $f.value
      elif $f.label == $sk then .sk = $f.value
      elif $f.label == $mfa then .mfa = $f.value
      else . end
    ) | "access_key_id=\(.ak // "")\nsecret_access_key=\(.sk // "")\nmfa_serial=\(.mfa // "")"
  ')"

  if [[ -z "$access_key_id" || -z "$secret_access_key" ]]; then
    echo "Error: AWS credentials not found in 1Password" >&2
    return 1
  fi

  # Fallback: use mfa_serial from AWS config if not in 1Password
  if [[ -z "$mfa_serial" ]]; then
    mfa_serial="$config_mfa_serial"
  fi

  local session_name="${role_arn##*/}"

  local sts_args=(sts assume-role --role-arn "$role_arn" --role-session-name "$session_name")

  # Get TOTP if MFA is configured
  if [[ -n "$mfa_serial" ]]; then
    local totp
    if ! totp=$(_apop_get_totp); then
      return 1
    fi
    sts_args+=(--serial-number "$mfa_serial" --token-code "$totp")
  fi

  # Call STS (with MFA retry)
  local sts_json sts_rc
  sts_json=$(_apop_call_sts "${sts_args[@]}")
  sts_rc=$?

  # Retry with new TOTP if MFA failed
  if [[ $sts_rc -ne 0 && -n "$mfa_serial" && "$sts_json" == *"MultiFactorAuthentication"* ]]; then
    local now=$(date +%s)
    local wait_sec=$(( 30 - now % 30 + 1 ))
    echo "MFA code expired or already used. Waiting ${wait_sec}s for next TOTP..." >&2
    sleep "$wait_sec"

    local new_totp
    if ! new_totp=$(_apop_get_totp); then
      return 1
    fi

    # Replace token-code in sts_args
    local i
    for (( i=0; i<${#sts_args[@]}; i++ )); do
      if [[ "${sts_args[$i]}" == "--token-code" ]]; then
        sts_args[$((i+1))]="$new_totp"
        break
      fi
    done

    echo "Retrying with new TOTP..." >&2
    sts_json=$(_apop_call_sts "${sts_args[@]}")
    sts_rc=$?
  fi

  if [[ $sts_rc -ne 0 ]]; then
    echo "Error: failed to assume role" >&2
    echo "$sts_json" >&2
    echo "Check the role ARN and ensure your credentials have sts:AssumeRole permission" >&2
    return 1
  fi

  # Record TOTP window for deduplication
  if [[ -n "$mfa_serial" ]]; then
    _APOP_LAST_TOTP_WINDOW=$(( $(date +%s) / 30 ))
  fi

  _apop_apply_credentials "$sts_json" "$role_arn"
  if [[ -n "$profile_name" ]]; then
    export AWS_PROFILE="$profile_name"
    export AWS_DEFAULT_PROFILE="$profile_name"
    echo "Successfully assumed role for profile: $profile_name" >&2
  else
    echo "Successfully assumed role: ${role_arn##*/}" >&2
  fi
  _apop_finalize
}

_apop_chain_role() {
  local role_arn="$1"

  # Verify current session credentials exist
  if [[ -z "${AWS_ACCESS_KEY_ID:-}" || -z "${AWS_SECRET_ACCESS_KEY:-}" || -z "${AWS_SESSION_TOKEN:-}" ]]; then
    echo "Error: no active AWS session credentials found" >&2
    echo "First assume a role with: apop <profile-name>" >&2
    return 1
  fi

  _apop_check_deps aws:awscli jq:jq || return 1

  local parent_role="${AWS_ASSUMED_ROLE_ARN##*/}"
  local parent_account="${AWS_ASSUMED_ROLE_ARN#*:*:*:*:}"
  parent_account="${parent_account%%:*}"
  local session_name="${parent_role}@${parent_account}"

  echo "Chain-assuming role: ${role_arn##*/}" >&2

  local sts_json
  if ! sts_json=$(aws sts assume-role \
    --role-arn "$role_arn" \
    --role-session-name "$session_name" \
    2>&1); then
    echo "Error: failed to chain-assume role" >&2
    echo "$sts_json" >&2
    echo "Check the role ARN and ensure your current role has sts:AssumeRole permission" >&2
    return 1
  fi

  _apop_apply_credentials "$sts_json" "$role_arn"
  echo "Successfully chain-assumed role: ${role_arn##*/}" >&2
  _apop_finalize
}

# --- Helpers ---

_apop_check_deps() {
  local cmd; for cmd in "$@"; do
    if ! command -v "${cmd%%:*}" &>/dev/null; then
      echo "Error: ${cmd%%:*} is not installed" >&2
      echo "Install it: brew install ${cmd##*:}" >&2
      return 1
    fi
  done
}

_apop_apply_credentials() {
  local sts_json="$1" role_arn="$2"
  eval "$(jq -r '.Credentials |
    "export AWS_ACCESS_KEY_ID=\(.AccessKeyId)\nexport AWS_SECRET_ACCESS_KEY=\(.SecretAccessKey)\nexport AWS_SESSION_TOKEN=\(.SessionToken)"
  ' <<< "$sts_json")"
  export AWS_REGION="$APOP_AWS_REGION"
  export AWS_ASSUMED_ROLE_ARN="$role_arn"
}

_apop_finalize() {
  if [[ "$_apop_copy_to_clipboard" == true ]]; then
    printf 'AWS_ACCESS_KEY_ID=%s\nAWS_SECRET_ACCESS_KEY=%s\nAWS_REGION=%s\nAWS_SESSION_TOKEN=%s\n' \
      "$AWS_ACCESS_KEY_ID" "$AWS_SECRET_ACCESS_KEY" "$AWS_REGION" "$AWS_SESSION_TOKEN" \
      | pbcopy
    echo "Credentials copied to clipboard" >&2
  fi
  AWS_PAGER="" aws sts get-caller-identity --output table >&2
}

_apop_get_totp() {
  # Wait if still in the same TOTP window as last use
  local now=$(date +%s)
  local current_window=$(( now / 30 ))
  if [[ "${_APOP_LAST_TOTP_WINDOW:-0}" == "$current_window" ]]; then
    local wait_sec=$(( 30 - now % 30 + 1 ))
    echo "Waiting ${wait_sec}s for next TOTP window..." >&2
    sleep "$wait_sec"
  fi

  local totp
  if ! totp=$(OP_BIOMETRIC_UNLOCK_ENABLED=true op item get "$APOP_OP_ITEM_NAME" --otp 2>&1); then
    echo "Error: failed to get TOTP from 1Password" >&2
    echo "$totp" >&2
    echo "Ensure OTP is configured: op item get $APOP_OP_ITEM_NAME --otp" >&2
    return 1
  fi
  echo "Using TOTP from 1Password" >&2
  echo "$totp"
}

_apop_call_sts() {
  env -u AWS_SESSION_TOKEN -u AWS_SECURITY_TOKEN -u AWS_PROFILE -u AWS_DEFAULT_PROFILE \
    AWS_ACCESS_KEY_ID="$access_key_id" AWS_SECRET_ACCESS_KEY="$secret_access_key" \
    aws "$@" 2>&1
}

_apop_list_profiles() {
  local aws_config="${AWS_CONFIG_FILE:-$HOME/.aws/config}"
  if [[ ! -f "$aws_config" ]]; then
    echo "Error: $aws_config not found" >&2
    echo "Create it: aws configure" >&2
    return 1
  fi
  awk '/^\[profile /{gsub(/\[profile |\]/, ""); print}' "$aws_config"
}

_apop_get_profile_value() {
  local config_file="$1" profile_name="$2" key="$3"
  awk -v profile="$profile_name" -v key="$key" '
    /^\[profile / { current = $0; gsub(/\[profile |\]/, "", current) }
    current == profile && index($0, key "=") || current == profile && index($0, key " =") {
      sub(/^[^=]*=[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); print; exit
    }
  ' "$config_file"
}

_apop_get_profile_values() {
  local config_file="$1" profile_name="$2"
  shift 2
  local keys="$*"
  awk -v profile="$profile_name" -v keys="$keys" '
    BEGIN { n = split(keys, ka, " ") }
    /^\[profile / { current = $0; gsub(/\[profile |\]/, "", current) }
    current == profile {
      for (i = 1; i <= n; i++) {
        k = ka[i]
        if (index($0, k "=") || index($0, k " =")) {
          val = $0; sub(/^[^=]*=[[:space:]]*/, "", val); sub(/[[:space:]]*$/, "", val)
          printf "%s='\''%s'\''\n", k, val
        }
      }
    }
  ' "$config_file"
}

_apop_select_profile() {
  local profiles="$1"
  local current="${AWS_PROFILE:-${AWS_DEFAULT_PROFILE:-default}}"

  if command -v fzf &>/dev/null; then
    echo "$profiles" | fzf --height 40% --reverse --header "Current: $current | Select AWS Profile"
  else
    echo "Current: $current" >&2
    echo "Select AWS Profile:" >&2
    echo >&2

    local i=1
    local profile
    while IFS= read -r profile; do
      local marker="  "
      [[ "$profile" == "$current" ]] && marker="> "
      echo "${marker}${i}) ${profile}" >&2
      ((i++))
    done <<< "$profiles"

    echo >&2
    read -rp "Enter number: " idx
    if [[ "$idx" =~ ^[0-9]+$ ]] && (( idx >= 1 && idx <= i - 1 )); then
      echo "$profiles" | sed -n "${idx}p"
    else
      echo "Error: invalid selection: enter a number between 1 and $((i - 1))" >&2
      return 1
    fi
  fi
}
