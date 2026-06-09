#!/usr/bin/env bash
set -euo pipefail

plugin_src="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
binary="tabularis-mongodb-plugin"
plugin_id="mongodb"

case "$(uname -s)" in
  Linux*)        plugins_dir="${XDG_DATA_HOME:-$HOME/.local/share}/tabularis/plugins" ;;
  Darwin*)       plugins_dir="$HOME/Library/Application Support/com.debba.tabularis/plugins" ;;
  MINGW*|MSYS*|CYGWIN*) plugins_dir="${APPDATA}/com.debba.tabularis/plugins" ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

dest_dir="$plugins_dir/$plugin_id"

echo "Building $binary..."
(cd "$plugin_src" && go build -trimpath -ldflags "-s -w" -o "$binary" ./cmd/tabularis-mongodb-plugin)

if [[ -d "$plugin_src/web" ]] && command -v npm >/dev/null 2>&1; then
  echo "Building UI extensions..."
  (cd "$plugin_src/web" && { [[ -d node_modules ]] || npm install; } && npm run build)
fi

mkdir -p "$dest_dir"
cp "$plugin_src/$binary" "$dest_dir/$binary.new"
chmod +x "$dest_dir/$binary.new"
mv -f "$dest_dir/$binary.new" "$dest_dir/$binary"
cp "$plugin_src/manifest.json" "$dest_dir/manifest.json"

if [[ -d "$plugin_src/ui" ]]; then
  rm -rf "$dest_dir/ui"
  cp -r "$plugin_src/ui" "$dest_dir/ui"
  echo "Copied UI extensions"
fi

echo "Installed to: $dest_dir"
echo "Restart Tabularis to load the plugin."
