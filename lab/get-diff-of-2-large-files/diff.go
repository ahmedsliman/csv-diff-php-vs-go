package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
)

type attrs struct {
	price    string
	quantity string
}

func main() {
	dir := "."
	oldFile := dir + "/data/old.csv"
	newFile := dir + "/data/new.csv"
	outDir := dir + "/output/go"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		panic(err)
	}

	// old[EAN] = {price, quantity}
	old := make(map[string]attrs, 6_000_000)

	// --- Read old.csv ---------------------------------------------------
	f, err := os.Open(oldFile)
	if err != nil {
		panic(err)
	}
	r := csv.NewReader(bufio.NewReaderSize(f, 256*1024))
	if _, err := r.Read(); err != nil { // skip header
		panic(err)
	}
	for {
		row, err := r.Read()
		if err != nil {
			break
		}
		old[row[0]] = attrs{price: row[1], quantity: row[2]}
	}
	f.Close()

	// --- Read new.csv ---------------------------------------------------
	var newProducts, modifiedProducts [][]string
	seenInNew := make(map[string]bool, 6_000_000)

	f2, err := os.Open(newFile)
	if err != nil {
		panic(err)
	}
	r2 := csv.NewReader(bufio.NewReaderSize(f2, 256*1024))
	if _, err := r2.Read(); err != nil { // skip header
		panic(err)
	}
	for {
		row, err := r2.Read()
		if err != nil {
			break
		}
		ean := row[0]
		price := row[1]
		qty := row[2]

		seenInNew[ean] = true

		if v, ok := old[ean]; !ok {
			newProducts = append(newProducts, []string{ean, price, qty})
		} else if v.price != price || v.quantity != qty {
			modifiedProducts = append(modifiedProducts, []string{ean, price, qty})
		}
	}
	f2.Close()

	// --- Removed: in old but not seen in new ---------------------------
	var removedProducts [][]string
	for ean, v := range old {
		if !seenInNew[ean] {
			removedProducts = append(removedProducts, []string{ean, v.price, v.quantity})
		}
	}

	writeCSV := func(path string, rows [][]string) {
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		w := csv.NewWriter(f)
		if err := w.Write([]string{"EAN", "price", "quantity"}); err != nil {
			panic(err)
		}
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				panic(err)
			}
		}
		w.Flush()
	}

	writeCSV(outDir+"/new-products.csv", newProducts)
	writeCSV(outDir+"/removed-products.csv", removedProducts)
	writeCSV(outDir+"/modified-products.csv", modifiedProducts)

	fmt.Printf("Done. Output in %s/\n", outDir)
	fmt.Printf("New:      %d\n", len(newProducts))
	fmt.Printf("Removed:  %d\n", len(removedProducts))
	fmt.Printf("Modified: %d\n", len(modifiedProducts))
}
