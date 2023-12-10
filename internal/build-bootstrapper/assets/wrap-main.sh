BINDOWN_REPO_URL="${BINDOWN_REPO_URL:-"https://github.com/WillAbides/bindown"}"

script_dir="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)"
bindown_bindir="$script_dir/$bindown_bindir"

log_set_priority 3 # log at error level

bindown_exec="$(install_bindown "${bindown_tag:?}" "${bindown_checksums:?}" "$bindown_bindir" "1" "$BINDOWN_REPO_URL")"

BINDOWN_WRAPPED="${BINDOWN_WRAPPED:-"$script_dir/$(basename "$0")"}"
export BINDOWN_WRAPPED
exec "$bindown_exec" "$@"
