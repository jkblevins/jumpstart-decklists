// Package parser reads MTG decklist text files into structured data.
package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// CardEntry represents a single card and its quantity in a decklist.
type CardEntry struct {
	Quantity int
	Name     string
}

// RawDeck represents is a parsed decklist with a name and card entries.
type RawDeck struct {
	Name          string
	ColorOverride string // "", "W", "U", "B", "R", "G", "M", or "C"
	Cards         []CardEntry
}

// Parse reads a decklist from r and returns the parsed decks.
// Multiple decks in one input are separated by "---" lines.
// Blank lines and lines starting with "//" are ignored.
func Parse(r io.Reader) ([]RawDeck, error) {
	scanner := bufio.NewScanner(r)
	var decks []RawDeck
	var current *RawDeck

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// check for deck separator
		if line == "---" {
			if current != nil {
				decks = append(decks, *current)
				current = nil
			}
			continue
		}

		// Ignore comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// begin new deck
		if current == nil {
			name, colorOverride, err := parseDeckName(line)
			if err != nil {
				return nil, err
			}
			current = &RawDeck{Name: name, ColorOverride: colorOverride}
			continue
		}

		// add cards to deck
		card, err := parseCardLine(line)
		if err != nil {
			return nil, err
		}
		current.Cards = append(current.Cards, card)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// add new deck to list of decks
	if current != nil {
		decks = append(decks, *current)
	}

	return decks, nil
}

// validColorOverrides is the set of accepted color codes for deck color override.
var validColorOverrides = map[string]bool{
	"W": true, "U": true, "B": true, "R": true, "G": true,
	"M": true, "C": true,
}

// colorOverrideRe matches an unescaped [X] suffix at the end of a deck name line.
var colorOverrideRe = regexp.MustCompile(`\s+\[([A-Z]+)\]\s*$`)

// parseDeckName extracts the deck name and optional color override from a
// name line. Escaped brackets (\[ and \]) are unescaped in the final name.
// Returns an error if the override code is not a valid color.
func parseDeckName(line string) (name string, colorOverride string, err error) {
	// Only match unescaped trailing brackets: find the last match that
	// isn't preceded by a backslash.
	if loc := colorOverrideRe.FindStringIndex(line); loc != nil {
		// Check the character before the match isn't a backslash
		if loc[0] == 0 || line[loc[0]-1] != '\\' {
			sub := colorOverrideRe.FindStringSubmatch(line[loc[0]:])
			colorOverride = sub[1]
			if !validColorOverrides[colorOverride] {
				name = strings.TrimSpace(line[:loc[0]])
				name = strings.ReplaceAll(name, `\[`, "[")
				name = strings.ReplaceAll(name, `\]`, "]")
				return "", "", fmt.Errorf("deck %q: invalid color override %q (valid: W, U, B, R, G, M, C)", name, colorOverride)
			}
			name = strings.TrimSpace(line[:loc[0]])
		}
	}

	if name == "" {
		name = line
	}

	// Unescape brackets
	name = strings.ReplaceAll(name, `\[`, "[")
	name = strings.ReplaceAll(name, `\]`, "]")

	return name, colorOverride, nil
}

// parseCardLine splits a line like "2 Lightning Bolt" into a CardEntry.
func parseCardLine(line string) (CardEntry, error) {
	spaceIdx := strings.IndexByte(line, ' ')
	if spaceIdx == -1 {
		return CardEntry{}, fmt.Errorf("invalid card line: %q", line)
	}

	// parse quantity
	qty, err := strconv.Atoi(line[:spaceIdx])
	if err != nil {
		return CardEntry{}, fmt.Errorf("invalid quantity in line %q: %w", line, err)
	}

	// parse name
	name := strings.TrimSpace(line[spaceIdx+1:])
	return CardEntry{Quantity: qty, Name: name}, nil
}
