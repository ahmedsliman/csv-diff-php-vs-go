<?php
// PHP: same algorithm as Go. Stream old.csv into an array keyed by EAN,
// stream new.csv once, then whatever is left in the array was removed.
// Usage: php -d memory_limit=-1 php/diff.php

declare(strict_types=1);

/** Yield one [EAN, price, quantity] row at a time; header skipped. */
function rows(string $path): Generator
{
    $fh = fopen($path, 'r') ?: throw new RuntimeException("Cannot open $path");
    try {
        fgets($fh);
        while (($line = fgets($fh)) !== false) {
            yield explode(',', rtrim($line, "\r\n"));
        }
    } finally {
        fclose($fh);
    }
}

function writeCsv(string $path, iterable $rows): int
{
    $fh = fopen($path, 'w') ?: throw new RuntimeException("Cannot create $path");
    fwrite($fh, "EAN,price,quantity\n");
    $count = 0;
    foreach ($rows as $row) {
        fwrite($fh, implode(',', $row) . "\n");
        $count++;
    }
    fclose($fh);
    return $count;
}

$outDir = 'output/php';
is_dir($outDir) || mkdir($outDir, 0777, true);

// $old[EAN] = "price,quantity" (one string per entry keeps memory down)
$old = [];
foreach (rows('data/old.csv') as [$ean, $price, $qty]) {
    $old[$ean] = "$price,$qty";
}

$added = $modified = [];
foreach (rows('data/new.csv') as $row) {
    [$ean, $price, $qty] = $row;
    $v = $old[$ean] ?? null;
    if ($v === null) {
        $added[] = $row;
    } elseif ($v !== "$price,$qty") {
        $modified[] = $row;
    }
    unset($old[$ean]); // whatever is left at the end was removed
}

$removed = (function () use ($old): Generator {
    foreach ($old as $ean => $v) {
        yield [$ean, $v];
    }
})();

$n = writeCsv("$outDir/new-products.csv", $added);
$r = writeCsv("$outDir/removed-products.csv", $removed);
$m = writeCsv("$outDir/modified-products.csv", $modified);

printf("New: %d  Removed: %d  Modified: %d\n", $n, $r, $m);
