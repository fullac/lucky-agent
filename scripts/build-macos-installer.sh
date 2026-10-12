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

mkdir -p "$root_dir/usr/local/lib/luckyagent" "$root_dir/usr/local/bin"
install -m 0755 "$bundle_dir/lh" "$root_dir/usr/local/lib/luckyagent/lh"

cat > "$root_dir/usr/local/bin/lh" <<'EOF'
#!/usr/bin/env sh
set -eu
exec /usr/local/lib/luckyagent/lh "$@"
EOF
chmod 0755 "$root_dir/usr/local/bin/lh"

mkdir -p "$output_dir"
pkgbuild --root "$root_dir" --identifier com.luckyagent.app --version "${version#v}" "$output_dir/LuckyAgent-${version#v}-macos-${arch}.pkg"
