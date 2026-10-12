#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <version> <arch> <bundle-dir> <output-dir>" >&2
  exit 2
fi

version=$1
arch=$2
bundle_dir=$3
output_dir=$4
root_dir=$(mktemp -d)
trap 'rm -rf "$root_dir"' EXIT

case "$arch" in
  amd64|arm64) ;;
  *) echo "unsupported Debian architecture: $arch" >&2; exit 1 ;;
esac

mkdir -p "$root_dir/DEBIAN" "$root_dir/opt/luckyagent" "$root_dir/usr/bin"
install -m 0755 "$bundle_dir/lh" "$root_dir/opt/luckyagent/lh"

cat > "$root_dir/DEBIAN/control" <<EOF
Package: luckyagent
Version: ${version#v}
Section: utils
Priority: optional
Architecture: $arch
Maintainer: LuckyAgent
Description: LuckyAgent CLI and HTTP API
 Agent runtime with local chat, HTTP API, memory, RAG, and message gateways.
EOF

cat > "$root_dir/usr/bin/lh" <<'EOF'
#!/usr/bin/env sh
set -eu
exec /opt/luckyagent/lh "$@"
EOF
chmod 0755 "$root_dir/usr/bin/lh"

mkdir -p "$output_dir"
dpkg-deb --build "$root_dir" "$output_dir/luckyagent_${version#v}_${arch}.deb"
