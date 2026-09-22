#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
for pkg in htmlcheck robots sitemap structureddata urlnorm content terms report; do
  go test "./internal/$pkg" -run '^$' -fuzz '^FuzzParse$' -fuzztime "${1:-5s}" -parallel 2
done
