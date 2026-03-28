package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jumpforge/deck"
	"jumpforge/parser"
	"jumpforge/pdf"
	"jumpforge/scryfall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: jumpforge <decklist.txt>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	f, err := os.Open(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	rawDecks, err := parser.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing decklist: %v\n", err)
		os.Exit(1)
	}

	if len(rawDecks) == 0 {
		fmt.Fprintln(os.Stderr, "error: no decks found in input file")
		os.Exit(1)
	}

	// Collect all unique card names.
	nameSet := make(map[string]bool)
	for _, rd := range rawDecks {
		for _, c := range rd.Cards {
			nameSet[c.Name] = true
		}
	}
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}

	// Fetch card data from Scryfall.
	client := scryfall.NewClient()
	cards, err := client.FetchCards(names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching card data: %v\n", err)
		os.Exit(1)
	}

	// Organize decks.
	var decks []deck.Deck
	for _, rd := range rawDecks {
		d := deck.Organize(rd, cards)
		if len(d.Groups) == 0 {
			fmt.Fprintf(os.Stderr, "WARNING: deck %q has no valid cards, skipping\n", rd.Name)
			continue
		}
		decks = append(decks, d)
	}

	if len(decks) == 0 {
		fmt.Fprintln(os.Stderr, "error: no valid decks after processing")
		os.Exit(1)
	}

	// Render PDF.
	var outPath string
	if len(decks) == 1 {
		outPath = pdf.OutputFileName(decks[0].Name)
		err = pdf.RenderSingle(decks[0], outPath)
	} else {
		base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outPath = base + ".pdf"
		err = pdf.RenderBatch(decks, outPath)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outPath)
}
