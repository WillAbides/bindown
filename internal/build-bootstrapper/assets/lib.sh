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
