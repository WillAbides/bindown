#!/bin/sh
set -e

TAG="v3.14.0"

CHECKSUMS="$(cat <<EOT
0c74e635373eb04c0bcc17ab76323b788ff68dddb856e08d6c5424309d04a372  bindown_3.14.0_linux_amd64.tar.gz
39123afe05c34a124f885cbc33e46851ead5b0b40e3fef30ffc3e6e1aca95377  bindown_3.14.0_linux_386
508c30fcbd273edd7924adfab31343e6d97b1caa0c4776bccd17a8e85f0d1fd7  bindown_3.14.0_darwin_arm64.tar.gz
52c777207a9869561ba85dc8d0669c4b3f6da6cf93a9e167985a33e89462f9fc  bindown_3.14.0_windows_amd64.exe
6681435f6ad83b49f14a98c7e75267a9e8c645c555febfc3cdb888c1da305648  bindown_3.14.0_linux_arm64
692bfbdeda62bc6bf27d4c43be3d26269df1a37f7c3687725ca909778e7d4a50  bindown_3.14.0_windows_arm64.tar.gz
760591f3e796ffb354f5eb1f265bab15756b39f78c584b21cd74e070ae13224c  bindown_3.14.0_windows_amd64.tar.gz
7b3b6eb0c3503fb093d23ce40a359c87f08aaa5c005a05d5d7da3e35677b022c  bindown_3.14.0_windows_386.exe
89d47c90f911b1af19488dd77c9083343684a9ea3b274f6fa2e27318f6963a8c  bindown_3.14.0_linux_386.tar.gz
99ed7f0996e989043db122ed40108d18eee1b4f0f62d152f9650150c067c2a26  bindown_3.14.0_windows_arm64.exe
b5c63d4ba54c144a914023c283ad37416acd5442459e9ba4984718388c6361d8  bindown_3.14.0_linux_arm64.tar.gz
bad06c8ab8e97efec0bd8d9246e1ad1614a0ef6803375c119e9865cafce268f7  bindown_3.14.0_darwin_arm64
cb1b4aed11c10c01106f7e02ae77f6f53018f1525f509846bb765805a2f8cee7  bindown_3.14.0_linux_amd64
d0fabf24bda8fda4623eea92556f3d77e7683862f1f2cf8080a71612e93c38dc  bindown_3.14.0_darwin_amd64
e0e3ba2d6e6bc8a7b1e2b137ff86875a429f9d4795cfcc6e886675172a9e618a  bindown_3.14.0_windows_386.tar.gz
f581bf1be92a02e81e6c6e375607688b322969d602fb139df3e0aec72bb8e678  bindown_3.14.0_darwin_amd64.tar.gz
EOT
)"


cat /dev/null <<EOF
------------------------------------------------------------------------
https://github.com/client9/shlib - portable posix shell functions
Public domain - http://unlicense.org
https://github.com/client9/shlib/blob/master/LICENSE.md
but credit (and pull requests) appreciated.
------------------------------------------------------------------------
EOF
is_command() {
  command -v "$1" >/dev/null
}
echoerr() {
  echo "$@" 1>&2
}
log_prefix() {
  echo "$0"
}
_logp=6
log_set_priority() {
  _logp="$1"
}
log_priority() {
  if test -z "$1"; then
    echo "$_logp"
    return
  fi
  [ "$1" -le "$_logp" ]
}
log_tag() {
  case $1 in
    0) echo "emerg" ;;
    1) echo "alert" ;;
    2) echo "crit" ;;
    3) echo "err" ;;
    4) echo "warning" ;;
    5) echo "notice" ;;
    6) echo "info" ;;
    7) echo "debug" ;;
    *) echo "$1" ;;
  esac
}
log_debug() {
  log_priority 7 || return 0
  echoerr "$(log_prefix)" "$(log_tag 7)" "$@"
}
log_info() {
  log_priority 6 || return 0
  echoerr "$(log_prefix)" "$(log_tag 6)" "$@"
}
log_err() {
  log_priority 3 || return 0
  echoerr "$(log_prefix)" "$(log_tag 3)" "$@"
}
log_crit() {
  log_priority 2 || return 0
  echoerr "$(log_prefix)" "$(log_tag 2)" "$@"
}
uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*) os="windows" ;;
    mingw*) os="windows" ;;
    msys_nt*) os="windows" ;;
  esac
  echo "$os"
}
uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
  esac
  echo ${arch}
}
uname_os_check() {
  os=$(uname_os)
  case "$os" in
    darwin) return 0 ;;
    dragonfly) return 0 ;;
    freebsd) return 0 ;;
    linux) return 0 ;;
    android) return 0 ;;
    nacl) return 0 ;;
    netbsd) return 0 ;;
    openbsd) return 0 ;;
    plan9) return 0 ;;
    solaris) return 0 ;;
    windows) return 0 ;;
  esac
  log_crit "uname_os_check '$(uname -s)' got converted to '$os' which is not a GOOS value. Please file bug at https://github.com/client9/shlib"
  return 1
}
uname_arch_check() {
  arch=$(uname_arch)
  case "$arch" in
    386) return 0 ;;
    amd64) return 0 ;;
    arm64) return 0 ;;
    armv5) return 0 ;;
    armv6) return 0 ;;
    armv7) return 0 ;;
    ppc64) return 0 ;;
    ppc64le) return 0 ;;
    mips) return 0 ;;
    mipsle) return 0 ;;
    mips64) return 0 ;;
    mips64le) return 0 ;;
    s390x) return 0 ;;
    amd64p32) return 0 ;;
  esac
  log_crit "uname_arch_check '$(uname -m)' got converted to '$arch' which is not a GOARCH value.  Please file bug report at https://github.com/client9/shlib"
  return 1
}
untar() {
  tarball=$1
  case "${tarball}" in
    *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" ;;
    *.tar) tar --no-same-owner -xf "${tarball}" ;;
    *.zip) unzip "${tarball}" ;;
    *)
      log_err "untar unknown archive format for ${tarball}"
      return 1
      ;;
  esac
}
http_download_curl() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    code=$(curl -w '%{http_code}' -sL -o "$local_file" "$source_url")
  else
    code=$(curl -w '%{http_code}' -sL -H "$header" -o "$local_file" "$source_url")
  fi
  if [ "$code" != "200" ]; then
    log_debug "http_download_curl received HTTP status $code"
    return 1
  fi
  return 0
}
http_download_wget() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    wget -q -O "$local_file" "$source_url"
  else
    wget -q --header "$header" -O "$local_file" "$source_url"
  fi
}
http_download() {
  log_debug "http_download $2"
  if is_command curl; then
    http_download_curl "$@"
    return
  elif is_command wget; then
    http_download_wget "$@"
    return
  fi
  log_crit "http_download unable to find wget or curl"
  return 1
}
hash_sha256() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha256sum; then
    hash=$(gsha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha256sum; then
    hash=$(sha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 256 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl -dst openssl dgst -sha256 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f a
  else
    log_crit "hash_sha256 unable to find command to compute sha-256 hash"
    return 1
  fi
}
hash_sha256_verify() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    log_err "hash_sha256_verify checksum file not specified in arg2"
    return 1
  fi
  BASENAME=${TARGET##*/}
  want=$(grep "${BASENAME}" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)
  if [ -z "$want" ]; then
    log_err "hash_sha256_verify unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_sha256_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}
cat /dev/null <<EOF
------------------------------------------------------------------------
End of functions from https://github.com/client9/shlib
------------------------------------------------------------------------
EOF

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