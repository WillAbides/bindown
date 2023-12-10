BINDOWN_REPO_URL="${BINDOWN_REPO_URL:-"https://github.com/WillAbides/bindown"}"

if [ -n "$BINDIR" ]; then
  bindown_bindir="$BINDIR"
fi

while getopts "b:cdh?x" arg; do
  case "$arg" in
    b) bindown_bindir="$OPTARG" ;;
    c) opt_use_checksum_path=1 ;;
    d) log_set_priority 10 ;;
    h | \?)
      echo "Usage: $0 [-b bindir] [-c] [-d] [-x]
  -b sets bindir or installation directory, Defaults to ./bin
  -c includes checksum in the output path
  -d turns on debug logging
  -x turns on bash debugging" >&2
      exit 2
      ;;
    x) set -x ;;
  esac
done

install_bindown "${bindown_tag:?}" "${bindown_checksums:?}" "$bindown_bindir" "$opt_use_checksum_path" "$BINDOWN_REPO_URL" > /dev/null
