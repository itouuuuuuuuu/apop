apop() {
  local cred_file="${HOME}/.apop-credentials"
  command apop "$@"
  local rc=$?
  if [ $rc -eq 0 ] && [ -f "$cred_file" ]; then
    source "$cred_file"
    rm -f "$cred_file"
  fi
  return $rc
}
