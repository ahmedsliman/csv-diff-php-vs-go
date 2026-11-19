# PHP vs Go vs Linux

I'm comparing three ways to find new, removed and changed products in two CSV files with **6 million rows each**.

## Results so far

These numbers are from the generated test files. They may change with different data, machines or versions of Go, PHP and the Linux tools.

| | Time | Memory |
|---|---|---|
| **Go** | 11.5 s | 1.15 GB |
| **PHP** | 24.4 s | 0.97 GB |
| **Linux** (`sort`, `comm`, `join`) | 28.4 s | 11 MB |

All three give the same answer: 1,000,000 new, 1,000,000 removed, 500,000 changed.

## What I'm learning

- Go took about half the time of PHP here. Both follow the same steps, but the code differs in how it reads and stores rows.
- PHP used less memory here. I still need to try other ways of writing both versions before drawing bigger conclusions.
- The Linux version uses temporary files, and `sort` can use disk when the data doesn't fit in memory. I also need a better memory measurement: the 11 MB above doesn't reliably cover all the commands running together.

I'm still working on this. PHP already uses generators to read rows, but keeps the old file's data and some results in memory. Writing results as they're found might save more memory. I want to try that and see what happens to the time as well.

The test files are simple CSVs without quoted fields. The PHP and shell versions rely on that.

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
