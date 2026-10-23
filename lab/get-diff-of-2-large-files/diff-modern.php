#!/usr/bin/env php
<?php
declare(strict_types=1);

#
# diff-modern.php - Modern PHP (generator-based) version of the large-file diff.
#
# Strategy:
#   1. Stream old.csv into a hash map keyed by EAN via a CSV row generator.
#   2. Stream new.csv once via the same generator; for each row do an O(1)
#      map lookup: missing -> new product, price/qty differ -> modified.
#   3. Any map key never seen in new.csv -> removed product.
#
# This mirrors the Go implementation (diff.go) but in modern PHP, using
# generators to stream rows without materializing entire files in memory.
#
# Complexity: O(n) time, O(n) memory (the old.csv map).
#
# Outputs (into output/php-modern/):
#   new-products.csv, removed-products.csv, modified-products.csv
#
# Usage: php -d memory_limit=8G diff-modern.php

ini_set('memory_limit', '8192M');

/**
 * Yield one CSV data row at a time from $path (header is skipped).
 * Each yielded row is a [EAN, price, quantity] string array.
 * Never loads the whole file into memory.
 */
function csvRows(string $path): Generator {
    $handle = fopen($path, 'r');
    if ($handle === false) {
        throw new RuntimeException(sprintf('Cannot open %s', $path));
    }
    try {
        fgets($handle); // skip header row
        while (($line = fgets($handle)) !== false) {
            yield explode(',', rtrim($line, "\r\n"));
        }
    } finally {
        fclose($handle);
    }
}

/**
 * Write $rows (each an array of strings) to $path with a header row.
 * Returns the number of data rows written.
 */
function writeCsv(string $path, array $header, iterable $rows): int {
    $handle = fopen($path, 'w');
    if ($handle === false) {
        throw new RuntimeException(sprintf('Cannot create %s', $path));
    }
    $count = 0;
    try {
        fputcsv($handle, $header, ',', '"', '\\');
        foreach ($rows as $row) {
            fputcsv($handle, $row, ',', '"', '\\');
            $count++;
        }
    } finally {
        fclose($handle);
    }
    return $count;
}

$dir = __DIR__;
$oldFile = $dir . '/data/old.csv';
$newFile = $dir . '/data/new.csv';
$outDir = $dir . '/output/php-modern';
if (!is_dir($outDir)) {
    mkdir($outDir, 0777, true);
}

$wallStart = microtime(true);

// --- 1. Load old.csv into a hash map (streamed via generator) ---------
// $old[EAN] = "price,quantity" (stored as a single string to avoid object
// overhead; compared as one concatenated string for speed).
$old = [];
foreach (csvRows($oldFile) as $row) {
    $old[$row[0]] = $row[1] . ',' . $row[2];
}

$oldLoaded = microtime(true);

// --- 2. Stream new.csv once, classify each row ------------------------
$newProducts = [];
$modifiedProducts = [];
$seenInNew = [];

foreach (csvRows($newFile) as $row) {
    $ean = $row[0];
    $seenInNew[$ean] = true;

    $v = $old[$ean] ?? null;
    if ($v === null) {
        $newProducts[] = $row;
    } elseif ($v !== $row[1] . ',' . $row[2]) {
        $modifiedProducts[] = $row;
    }
}

$newStreamed = microtime(true);

// --- 3. Removed: keys in old never seen in new ------------------------
$removedProducts = [];
foreach ($old as $ean => $v) {
    if (!isset($seenInNew[$ean])) {
        [$price, $qty] = explode(',', $v, 2);
        $removedProducts[] = [$ean, $price, $qty];
    }
}

$header = ['EAN', 'price', 'quantity'];

$newCount = writeCsv("$outDir/new-products.csv", $header, $newProducts);
$remCount = writeCsv("$outDir/removed-products.csv", $header, $removedProducts);
$modCount = writeCsv("$outDir/modified-products.csv", $header, $modifiedProducts);

$wallEnd = microtime(true);

$peakMemEmalloc = memory_get_peak_usage();
$peakMemReal = memory_get_peak_usage(true);

printf("Done. Output in %s/\n", $outDir);
printf("New:      %d\n", $newCount);
printf("Removed:  %d\n", $remCount);
printf("Modified: %d\n", $modCount);
printf("Wall time: %.4fs\n", $wallEnd - $wallStart);
printf("  load old.csv : %.4fs\n", $oldLoaded - $wallStart);
printf("  stream new.csv: %.4fs\n", $newStreamed - $oldLoaded);
printf("Peak memory (emalloc): %.2f MB\n", $peakMemEmalloc / 1024 / 1024);
printf("Peak memory (real)   : %.2f MB\n", $peakMemReal / 1024 / 1024);
