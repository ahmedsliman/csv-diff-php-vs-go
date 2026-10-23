#!/usr/bin/env bash
#
# diff.sh - Compare old.csv vs new.csv using Linux sort + comm + join + awk.
#
# Outputs (into output/linux/):
#   new-products.csv      EANs in new but not in old
#   removed-products.csv  EANs in old but not in new
#   modified-products.csv Same EAN in both, but price or quantity changed
#
# Usage: bash diff.sh
#
# Note: comm compares full lines, so we extract EANs first for the
# new/removed comparison, then join back to get full rows.

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OLD="$DIR/data/old.csv"
NEW="$DIR/data/new.csv"
OUT="$DIR/output/linux"
mkdir -p "$OUT"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Sort both files by EAN (field 1), strip headers.
sort -t, -k1,1 <(tail -n +2 "$OLD") > "$TMPDIR/old_sorted.csv"
sort -t, -k1,1 <(tail -n +2 "$NEW") > "$TMPDIR/new_sorted.csv"

# Extract just EANs for comm comparison (comm matches full lines).
cut -d, -f1 "$TMPDIR/old_sorted.csv" > "$TMPDIR/old_eans.txt"
cut -d, -f1 "$TMPDIR/new_sorted.csv" > "$TMPDIR/new_eans.txt"

# --- New products: EANs in new but not in old --------------------------
comm -13 "$TMPDIR/old_eans.txt" "$TMPDIR/new_eans.txt" > "$TMPDIR/new_eans_only.txt"
join -t, -1 1 -2 1 -o 2.1,2.2,2.3 \
    "$TMPDIR/new_eans_only.txt" "$TMPDIR/new_sorted.csv" > "$OUT/new-products.csv"

# --- Removed products: EANs in old but not in new ----------------------
comm -23 "$TMPDIR/old_eans.txt" "$TMPDIR/new_eans.txt" > "$TMPDIR/old_eans_only.txt"
join -t, -1 1 -2 1 -o 1.1,1.2,1.3 \
    "$TMPDIR/old_eans_only.txt" "$TMPDIR/old_sorted.csv" > "$OUT/removed-products.csv"

# --- Modified products: same EAN, different price or quantity ----------
join -t, -1 1 -2 1 -o 1.1,1.2,1.3,2.2,2.3 \
    "$TMPDIR/old_sorted.csv" "$TMPDIR/new_sorted.csv" \
  | awk -F, '$2 != $4 || $3 != $5 {print $1","$4","$5}' \
    > "$OUT/modified-products.csv"

# Add headers to all outputs.
for f in new removed modified; do
    sed -i '1i EAN,price,quantity' "$OUT/$f-products.csv"
done

echo "Done. Output in $OUT/"
echo "New:      $(tail -n +2 "$OUT/new-products.csv" | wc -l)"
echo "Removed:  $(tail -n +2 "$OUT/removed-products.csv" | wc -l)"
echo "Modified: $(tail -n +2 "$OUT/modified-products.csv" | wc -l)"
