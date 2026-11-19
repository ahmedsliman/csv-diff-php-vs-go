# PHP vs Go vs Linux

One task, three ways: compare two product CSV files with **6 million rows each** and find the new, removed and changed products.

## Results

Reported results for the generated test data; timings depend on the machine, tool versions and workload.

| | Time | Memory |
|---|---|---|
| **Go** | 11.5 s | 1.15 GB |
| **PHP** | 24.4 s | 0.97 GB |
| **Linux** (`sort`, `comm`, `join`) | 28.4 s | 11 MB |

All three give the same answer: 1,000,000 new, 1,000,000 removed, 500,000 changed.

## What I learned

- **Go** finished fastest in this benchmark, taking about half the time of PHP. Both use the same overall algorithm, but their parsing and data structures differ.
- **PHP** used less memory in these results. That reflects these implementations, rather than a general rule about PHP and Go.
- **Linux tools** trade temporary disk space and sorting for keeping the full dataset in memory. `sort` can spill to disk, but the reported 11 MB is not a reliable measure of the whole pipeline's peak memory.

This compares three implementations on simple generated CSVs, not the languages in general. The PHP and shell versions assume unquoted, comma-separated fields.

## Try it

You need Go, PHP 8 and a Linux shell.

```bash
make data     # create the two CSV files
make run      # run Go, PHP and Linux
make verify   # check that all three agree
make bench    # measure time and memory
```

## Files

| Path | What it is |
|---|---|
| `go/main.go` | Go version |
| `php/diff.php` | PHP version |
| `linux/diff.sh` | Linux shell version |
| `gen/main.go` | creates the test data |
