#!/usr/bin/env php
<?php
declare(strict_types=1);

#
# diff.php - Compare old.csv vs new.csv using PHP.
#
# Strategy: sort both files externally by EAN, then stream-compare line by line.
# This uses O(1) memory instead of loading all 6M rows into a PHP array.
#
# Outputs (into output/php/):
#   new-products.csv
#   removed-products.csv
#   modified-products.csv
#
# Usage: php diff.php
#

$dir = __DIR__;
$oldFile = $dir . '/data/old.csv';
$newFile = $dir . '/data/new.csv';
$outDir = $dir . '/output/php';
if (!is_dir($outDir)) {
    mkdir($outDir, 0777, true);
}

$tmp = $dir . '/data/.tmp';
if (!is_dir($tmp)) {
    mkdir($tmp, 0777, true);
}

$oldSorted = $tmp . '/old_sorted.csv';
$newSorted = $tmp . '/new_sorted.csv';

// Sort both files by EAN (field 1), stripping headers first.
$oldHeader = null;
$newHeader = null;

$oh = fopen($oldFile, 'r');
$oldHeader = fgets($oh);
fclose($oh);

$nh = fopen($newFile, 'r');
$newHeader = fgets($nh);
fclose($nh);

$cmdOld = sprintf(
    'tail -n +2 %s | sort -t, -k1,1 > %s',
    escapeshellarg($oldFile),
    escapeshellarg($oldSorted)
);
exec($cmdOld);

$cmdNew = sprintf(
    'tail -n +2 %s | sort -t, -k1,1 > %s',
    escapeshellarg($newFile),
    escapeshellarg($newSorted)
);
exec($cmdNew);

// Stream-compare the two sorted files.
$oh = fopen($oldSorted, 'r');
$nh = fopen($newSorted, 'r');

$newFp = fopen("$outDir/new-products.csv", 'w');
$remFp = fopen("$outDir/removed-products.csv", 'w');
$modFp = fopen("$outDir/modified-products.csv", 'w');

fputcsv($newFp, ['EAN', 'price', 'quantity'], ',', '"', '\\');
fputcsv($remFp, ['EAN', 'price', 'quantity'], ',', '"', '\\');
fputcsv($modFp, ['EAN', 'price', 'quantity'], ',', '"', '\\');

$newCount = $remCount = $modCount = 0;

$oldLine = fgets($oh);
$newLine = fgets($nh);

while ($oldLine !== false || $newLine !== false) {
    if ($oldLine === false) {
        // Everything remaining in new is new
        fputcsv($newFp, explode(',', rtrim($newLine, "\n")), ',', '"', '\\');
        $newCount++;
        $newLine = fgets($nh);
        continue;
    }
    if ($newLine === false) {
        // Everything remaining in old is removed
        fputcsv($remFp, explode(',', rtrim($oldLine, "\n")), ',', '"', '\\');
        $remCount++;
        $oldLine = fgets($oh);
        continue;
    }

    $oldParts = explode(',', rtrim($oldLine, "\n"));
    $newParts = explode(',', rtrim($newLine, "\n"));

    $oldEan = $oldParts[0];
    $newEan = $newParts[0];

    if ($oldEan === $newEan) {
        // Same product - check if price or quantity changed
        if ($oldParts[1] !== $newParts[1] || $oldParts[2] !== $newParts[2]) {
            fputcsv($modFp, [$newEan, $newParts[1], $newParts[2]], ',', '"', '\\');
            $modCount++;
        }
        $oldLine = fgets($oh);
        $newLine = fgets($nh);
    } elseif ($oldEan < $newEan) {
        // Old EAN not in new → removed
        fputcsv($remFp, $oldParts, ',', '"', '\\');
        $remCount++;
        $oldLine = fgets($oh);
    } else {
        // New EAN not in old → added
        fputcsv($newFp, $newParts, ',', '"', '\\');
        $newCount++;
        $newLine = fgets($nh);
    }
}

fclose($oh);
fclose($nh);
fclose($newFp);
fclose($remFp);
fclose($modFp);

// Cleanup temp sorted files
@unlink($oldSorted);
@unlink($newSorted);
@rmdir($tmp);

echo "Done. Output in $outDir/\n";
echo "New:      $newCount\n";
echo "Removed:  $remCount\n";
echo "Modified: $modCount\n";
