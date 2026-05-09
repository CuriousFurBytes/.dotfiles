#!/usr/bin/env bash
# Installs the asyar-coffee-extension (sleep prevention) into Asyar's extensions
# directory by downloading the pre-built zip from the latest GitHub release.
set -euo pipefail

DEST="$HOME/.config/Asyar/extensions/org.asyar.coffee"

URL=$(curl -fsSL "https://api.github.com/repos/Xoshbin/asyar-coffee-extension/releases/latest" \
    | python3 -c "
import json, sys
for a in json.load(sys.stdin).get('assets', []):
    n = a.get('name', '')
    if n.startswith('org.asyar.coffee') and n.endswith('.zip'):
        print(a['browser_download_url']); break
")

[ -n "$URL" ] || { echo "error: asyar-coffee release asset not found" >&2; exit 1; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL -o "$TMP/ext.zip" "$URL"
unzip -qo "$TMP/ext.zip" -d "$TMP/extract"

# Unwrap single top-level directory if the zip includes one
src="$TMP/extract"
inner=$(ls -1 "$TMP/extract" | wc -l | tr -d ' ')
first=$(ls -1 "$TMP/extract" | head -1)
[ "$inner" -eq 1 ] && [ -d "$TMP/extract/$first" ] && src="$TMP/extract/$first"

mkdir -p "$DEST"
cp -r "$src/." "$DEST/"
echo "asyar-coffee-extension installed to $DEST"
