# PHP vs Go vs Linux

One task, three ways: compare two product CSV files with **6 million rows each** and find the new, removed and changed products.

## Results

| | Time | Memory |
|---|---|---|
| **Go** | 11.5 s | 1.15 GB |
| **PHP** | 24.4 s | 0.97 GB |
| **Linux** (`sort`, `comm`, `join`) | 28.4 s | 11 MB |

All three give the same answer: 1,000,000 new, 1,000,000 removed, 500,000 changed.

## What I learned

- **Go** is the fastest, about 2× faster than PHP with the same algorithm.
- **PHP** used less memory than Go here.
- **Linux** needs almost no memory, so it still works when the files don't fit in RAM.

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
