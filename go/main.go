// Go: load old.csv into a map, stream new.csv once, then sweep for removals.
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

type attrs struct {
	price    string
	quantity string
}

func main() {
	outDir := "output/go"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	// old[EAN] = {price, quantity}
	old := make(map[string]attrs, 6_000_000)
	eachRow("data/old.csv", func(row []string) {
		old[row[0]] = attrs{price: row[1], quantity: row[2]}
	})

	var added, modified [][]string
	eachRow("data/new.csv", func(row []string) {
		v, ok := old[row[0]]
		switch {
		case !ok:
			added = append(added, row)
		case v.price != row[1] || v.quantity != row[2]:
			modified = append(modified, row)
		}
		delete(old, row[0]) // whatever is left at the end was removed
	})

	removed := make([][]string, 0, len(old))
	for ean, v := range old {
		removed = append(removed, []string{ean, v.price, v.quantity})
	}

	writeCSV(outDir+"/new-products.csv", added)
	writeCSV(outDir+"/removed-products.csv", removed)
	writeCSV(outDir+"/modified-products.csv", modified)

	fmt.Printf("New: %d  Removed: %d  Modified: %d\n", len(added), len(removed), len(modified))
}

// eachRow streams a CSV file (header skipped) and calls fn for every row.
func eachRow(path string, fn func([]string)) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReaderSize(f, 256*1024))
	if _, err := r.Read(); err != nil { // header
		panic(err)
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		fn(row)
	}
}

func writeCSV(path string, rows [][]string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := csv.NewWriter(f) // buffers internally
	w.Write([]string{"EAN", "price", "quantity"})
	w.WriteAll(rows) // flushes and returns the first write error
	if err := w.Error(); err != nil {
		panic(err)
	}
}
