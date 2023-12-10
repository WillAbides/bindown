#!/bin/sh

set -e

bindown_tag="v4.8.0"

bindown_checksums="
26fcbc738bf9bb910b070f236816b2dfe5bc9589be3a578135f37d950ebaf771  bindown_4.8.0_freebsd_amd64.tar.gz
2fa6460ebe8d7c6be33576acf5b63f7208780af72d758e82313f6c5772e097d5  bindown_4.8.0_linux_386.tar.gz
32e3fbfaecf41a1b2bced22c1842b3905f4e6de1e879a4db68402799c206415d  bindown_4.8.0_windows_386.exe
335802ed91fa6f040e10570479a6c817c7e42bd57fe98c959890a821099d3e1f  bindown_4.8.0_freebsd_arm64
372846f7edd9d93df0cb17889790f595f17cb083e093f3e6437d34e3253fd378  bindown_4.8.0_windows_amd64.exe
40acf94b7c69e5d4101cb46ea99641d302ff23579cd7ead29a5abfceb1a5d9ba  bindown_4.8.0_linux_arm64.tar.gz
66aca230d9aea549ecd3647082b63752f5bb5307ef6954a08cc0eaf9c70723f1  bindown_4.8.0_windows_amd64.tar.gz
752c78a926be1287312eea3c831e841e223de4780d1e4a8a813432d0a73f890b  bindown_4.8.0_linux_amd64.tar.gz
7f1f1c883beceb6ec3919484511fb20c3ceb41088e814d6fc234b015e98b78d9  bindown_4.8.0_darwin_arm64
7fdfbc007c0c285a498bf251bd4ab7469f334752581b45fda5ad6450ddd23377  bindown_4.8.0_windows_arm64.exe
95764bf76b54d5b13b9b8a801635d82447ee349c3545145ddd8a0a84246d66e2  bindown_4.8.0_freebsd_arm64.tar.gz
966087f13a6cf82804456119289ab982f2eee3ad04d8d4fb6ce74bd7eabdf74e  bindown_4.8.0_windows_386.tar.gz
9b29e37ba273bc0dca9c8227ee4b58153289073ede7d900e9c84ae3c71f3dff5  bindown_4.8.0_windows_arm64.tar.gz
a625900e52f4413bee3863062463cc24f9c0669841fd6bc9979ee599edd88f3e  bindown_4.8.0_freebsd_amd64
ba09df557edc4499f41ddadc26369d7f70ed20bfb8310662f1290e6a355343e8  bindown_4.8.0_darwin_amd64.tar.gz
cd7b917d2737fe9fa087aea172d9b581757e9b300fa1d1dbd83c1b765be05bdb  bindown_4.8.0_freebsd_386.tar.gz
d5d35274d4eab337c107940fc5b326c51f5bfd70d00924c79011684e2a0d4f22  bindown_4.8.0_freebsd_386
d71d6c436ad33bb3aa01468698b86d5423127a19f9b1c664e346cc502501d415  bindown_4.8.0_darwin_arm64.tar.gz
d9361698bc1571c34915496da9c624e89fa12d87731711efd2cbbc9136c6fa85  bindown_4.8.0_darwin_amd64
d93eae8638b96682d0e9b55bcbe92fecb296afd442e0526cc94ce0160c108c13  bindown_4.8.0_linux_arm64
ec3d19abd00fbf099a98edb64c569842fa5b909222fb10da86d668f5597885be  bindown_4.8.0_linux_amd64
fa7e87f49aa30e42485431bd9dd021a32924ab11e4d39065533e9bccce182de4  bindown_4.8.0_linux_386
"

cat /dev/null << EOF
------------------------------------------------------------------------
https://github.com/client9/shlib - portable posix shell functions
Public domain - http://unlicense.org
https://github.com/client9/shlib/blob/master/LICENSE.md
but credit (and pull requests) appreciated.
------------------------------------------------------------------------
EOF
is_command() {
  command -v "$1" > /dev/null
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
  echo "${arch}"
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
    hash=$(shasum -a 256 "$TARGET" 2> /dev/null) || return 1
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
  want=$(grep "${BASENAME}" "${checksums}" 2> /dev/null | tr '\t' ' ' | cut -d ' ' -f 1)
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
cat /dev/null << EOF
------------------------------------------------------------------------
End of functions from https://github.com/client9/shlib
------------------------------------------------------------------------
EOF

bindown_name() {
  if [ "$(uname_os)" = "windows" ]; then
    echo bindown.exe
  else
    echo bindown
  fi
}

already_installed() {
  version="$1"
  bindir="$2"
  use_checksum_path="$3"
  [ -f "$bindir/$(bindown_name)" ] || return 1
  if [ -n "$use_checksum_path" ]; then
    return
  fi
  "$bindir/$(bindown_name)" version 2> /dev/null | grep -q "$version"
}

install_bindown() {
  tag="$1"
  checksums="$2"
  bindir="$3"
  use_checksum_path="$4"
  repo_url="$5"

  version=${tag#v}
  tarball="bindown_${version}_$(uname_os)_$(uname_arch).tar.gz"
  tarball_url="$repo_url/releases/download/${tag}/${tarball}"

  if [ -n "$use_checksum_path" ]; then
    tarball_checksum="$(echo "$checksums" | grep "$tarball" | tr '\t' ' ' | cut -d ' ' -f 1)"
    bindir="${bindir}/${tarball_checksum}"
  fi

  echo "$bindir/$(bindown_name)"

  if already_installed "$version" "$bindir" "$use_checksum_path"; then
    log_info "bindown $version already installed in $bindir"
    return
  fi

  tmpdir=$(mktemp -d)
  echo "$checksums" > "${tmpdir}/checksums.txt"
  http_download "${tmpdir}/${tarball}" "${tarball_url}"
  hash_sha256_verify "${tmpdir}/${tarball}" "${tmpdir}/checksums.txt"
  (cd "${tmpdir}" && untar "${tarball}")
  test ! -d "${bindir}" && install -d "${bindir}"
  install "$tmpdir/$(bindown_name)" "${bindir}/"
  log_info "installed ${bindir}/$(bindown_name)"
  rm -rf "${tmpdir}"
}

bindown_bindir="./bin"

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
