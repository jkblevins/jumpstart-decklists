package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

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

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Timestamp().Logger()

	inputPath := os.Args[1]
	f, err := os.Open(inputPath)
	if err != nil {
		log.Fatal().Err(err).Str("file", inputPath).Msg("failed to open input")
	}
	defer f.Close()

	rawDecks, err := parser.Parse(f)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse decklist")
	}

	if len(rawDecks) == 0 {
		log.Fatal().Msg("no decks found in input file")
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
	client := scryfall.NewClient(log)
	cards, err := client.FetchCards(names)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch card data")
	}

	// Organize decks.
	var decks []deck.Deck
	for _, rd := range rawDecks {
		d := deck.Organize(rd, cards)
		if len(d.Groups) == 0 {
			log.Warn().Str("deck", rd.Name).Msg("no valid cards, skipping")
			continue
		}
		decks = append(decks, d)
	}

	if len(decks) == 0 {
		log.Fatal().Msg("no valid decks after processing")
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
		log.Fatal().Err(err).Msg("failed to generate PDF")
	}

	log.Info().Str("output", outPath).Msg("PDF generated")
}
