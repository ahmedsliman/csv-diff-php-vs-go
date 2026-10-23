// gen writes data/old.csv and data/new.csv for the diff showdown.
//
// Every EAN is unique (a bijection of the row index), so the expected
// output counts are exact:
//
//	common   = -common rows in both files (1 in 10 modified in new.csv)
//	removed  = common / 5 rows only in old.csv
//	added    = common / 5 rows only in new.csv
//
// Usage: go run ./gen [-common 5000000]
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
)

// ean maps row index i to a unique, shuffled-looking 13-digit EAN.
// i*p mod m is a bijection because p shares no factor with m (2^12*3^2*5^12).
func ean(i int64) int64 {
	const m = 9_000_000_000_000
	const p = 1_234_567_891
	return 1_000_000_000_000 + (i*p)%m
}

func main() {
	common := flag.Int64("common", 5_000_000, "rows present in both files")
	flag.Parse()

	removed := *common / 5
	added := *common / 5

	if err := os.MkdirAll("data", 0o755); err != nil {
		panic(err)
	}
	oldF, oldW := create("data/old.csv")
	newF, newW := create("data/new.csv")

	r := rand.New(rand.NewSource(42))
	modified := 0

	for i := int64(0); i < *common+removed+added; i++ {
		cents := 100 + r.Int63n(99_900) // 1.00 .. 999.99
		qty := r.Int63n(10_001)
		e := ean(i)

		switch {
		case i < *common:
			fmt.Fprintf(oldW, "%d,%s,%d\n", e, price(cents), qty)
			if i%10 == 0 { // modified: price changes in new.csv
				cents++
				modified++
			}
			fmt.Fprintf(newW, "%d,%s,%d\n", e, price(cents), qty)
		case i < *common+removed:
			fmt.Fprintf(oldW, "%d,%s,%d\n", e, price(cents), qty)
		default:
			fmt.Fprintf(newW, "%d,%s,%d\n", e, price(cents), qty)
		}
	}

	for _, w := range []*bufio.Writer{oldW, newW} {
		if err := w.Flush(); err != nil {
			panic(err)
		}
	}
	oldF.Close()
	newF.Close()

	expected := fmt.Sprintf("New: %d  Removed: %d  Modified: %d\n", added, removed, modified)
	if err := os.WriteFile("data/expected.txt", []byte(expected), 0o644); err != nil {
		panic(err)
	}
	fmt.Print("Expected: ", expected)
}

func create(path string) (*os.File, *bufio.Writer) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriterSize(f, 1<<20)
	fmt.Fprintln(w, "EAN,price,quantity")
	return f, w
}

func price(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
