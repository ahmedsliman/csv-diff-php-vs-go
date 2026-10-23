#!/usr/bin/env bash
# Linux: no code, just coreutils. Sort both files by EAN, then
# comm finds keys on one side only and join + awk finds changed rows.
# Usage: bash linux/diff.sh
set -euo pipefail
export LC_ALL=C # byte-wise sort: faster, and comm/join need the same order

OUT=output/linux
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$OUT"

tail -n +2 data/old.csv | sort -t, -k1,1 > "$TMP/old"
tail -n +2 data/new.csv | sort -t, -k1,1 > "$TMP/new"
cut -d, -f1 "$TMP/old" > "$TMP/old.keys"
cut -d, -f1 "$TMP/new" > "$TMP/new.keys"

header() { echo "EAN,price,quantity"; }

{ header; comm -13 "$TMP/old.keys" "$TMP/new.keys" | join -t, - "$TMP/new"; } > "$OUT/new-products.csv"
{ header; comm -23 "$TMP/old.keys" "$TMP/new.keys" | join -t, - "$TMP/old"; } > "$OUT/removed-products.csv"
{ header; join -t, "$TMP/old" "$TMP/new" | awk -F, '$2 != $4 || $3 != $5 { print $1 "," $4 "," $5 }'; } > "$OUT/modified-products.csv"

count() { echo $(($(wc -l < "$OUT/$1-products.csv") - 1)); }
echo "New: $(count new)  Removed: $(count removed)  Modified: $(count modified)"
