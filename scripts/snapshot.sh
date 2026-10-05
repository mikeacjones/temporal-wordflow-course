#!/usr/bin/env bash
# Maintainer tool: copy app/ into solutions/NN, rewriting import paths.
#   scripts/snapshot.sh 03          app/ -> solutions/03
#   scripts/snapshot.sh 03 restore  solutions/03 -> app/
set -euo pipefail
cd "$(dirname "$0")/.."
n="$1"
if [[ "${2:-}" == "restore" ]]; then
  src="solutions/$n"; dst="app"; from="wordflow/solutions/$n/"; to="wordflow/app/"
else
  src="app"; dst="solutions/$n"; from="wordflow/app/"; to="wordflow/solutions/$n/"
fi
rm -rf "$dst"
mkdir -p "$dst"
cp -R "$src"/. "$dst"/
find "$dst" -name '*.go' -exec perl -pi -e "s#\Q$from\E#$to#g" {} +
gofmt -w "$dst"
