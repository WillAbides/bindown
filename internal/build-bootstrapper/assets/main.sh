FORMAT=tar.gz
GITHUB_DOWNLOAD=https://github.com/WillAbides/bindown/releases/download

usage() {
  this=$1
  cat <<EOT
Usage: $this [-b bindir] [-d]

Usage: $this [-b] bindir [-d]
  -b sets bindir or installation directory, Defaults to ./bin
  -d turns on debug logging

EOT
  exit 2
}

parse_args() {
  #BINDIR is ./bin unless set be ENV
  # over-ridden by flag below

  BINDIR=${BINDIR:-./bin}
  while getopts "b:dh?x" arg; do
    case "$arg" in
      b) BINDIR="$OPTARG" ;;
      d) log_set_priority 10 ;;
      h | \?) usage "$0" ;;
      x) set -x ;;
    esac
  done
  shift $((OPTIND - 1))
}

execute() {
  tmpdir=$(mktemp -d)
  echo "$CHECKSUMS" > "${tmpdir}/checksums.txt"
  log_debug "downloading files into ${tmpdir}"
  http_download "${tmpdir}/${TARBALL}" "${TARBALL_URL}"
  hash_sha256_verify "${tmpdir}/${TARBALL}" "${tmpdir}/checksums.txt"
  srcdir="${tmpdir}"
  (cd "${tmpdir}" && untar "${TARBALL}")
  test ! -d "${BINDIR}" && install -d "${BINDIR}"
  binexe="bindown"
  if [ "$OS" = "windows" ]; then
    binexe="${binexe}.exe"
  fi
  install "${srcdir}/${binexe}" "${BINDIR}/"
  log_info "installed ${BINDIR}/${binexe}"
  rm -rf "${tmpdir}"
}

OS=$(uname_os)
ARCH=$(uname_arch)

uname_os_check "$OS"
uname_arch_check "$ARCH"

parse_args "$@"

VERSION=${TAG#v}
NAME=bindown_${VERSION}_${OS}_${ARCH}
TARBALL=${NAME}.${FORMAT}
TARBALL_URL=${GITHUB_DOWNLOAD}/${TAG}/${TARBALL}

execute
