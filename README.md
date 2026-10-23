# PHP vs Go vs Linux

The same job done three ways: **diff two 6-million-row product CSV files** and find which products were added, removed or changed.

| Stack | Approach | Time | Peak memory |
|---|---|---|---|
| **Go** | hash map, one pass | **11.5 s** | 1.15 GB |
| **PHP 8** | hash map + generators, one pass | 24.4 s | 0.97 GB |
| **Linux** | `sort` + `comm` + `join` + `awk` | 28.4 s | **11 MB** |

All three produce identical output: 1,000,000 new, 1,000,000 removed and 500,000 modified products, checked by `make verify`.

<sub>One run each on 8 cores and 32 GB RAM, Go 1.25, PHP 8.5, GNU coreutils. Your numbers will differ, so run `make bench`.</sub>

## Takeaways

- **Go is about 2× faster than PHP** with the same algorithm. It's compiled, and its map lookups are cheap.
- **PHP isn't the memory hog people expect.** Storing each row as one `"price,qty"` string keeps its array smaller than Go's map of structs.
- **Linux uses almost no memory.** `sort` spills to disk, so this approach still works when the files are bigger than RAM.
- **Pick by constraint, not by language:** Go for a recurring job, PHP to stay inside an existing PHP codebase, the shell for a one-off run on a small server.

## Run it

Needs Go 1.25+, PHP 8.x and GNU coreutils.

```bash
make data     # generate data/old.csv + data/new.csv (~150 MB each)
make run      # run all three
make verify   # all stacks must match the expected counts
make bench    # time + peak memory per stack
```

Use `make data COMMON=50000` for a quick small run.

## Layout

```
gen/main.go     test data generator (unique keys, so counts are exact)
go/main.go      Go
php/diff.php    PHP
linux/diff.sh   Linux shell
```
