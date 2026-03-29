// Package deck organizes parsed decklists into grouped, sorted structures
// suitable for rendering. It classifies cards by type, sorts within groups
// by mana cost, and determines the deck's dominant color identity.
package deck

import (
	"sort"
	"strings"

	"jumpstart-decklists/parser"
	"jumpstart-decklists/scryfall"
)

// DeckCard represents a single card entry within a type group.
type DeckCard struct {
	Name     string
	Quantity int
	CMC      float64
}

// TypeGroup holds all cards of a single type (e.g., "Creature") along with
// the total quantity of cards in the group.
type TypeGroup struct {
	TypeName string
	Cards    []DeckCard
	Count    int // total quantity
}

// Deck is a fully organized decklist ready for rendering, with cards grouped
// by type and a dominant color determined from color identity frequencies.
type Deck struct {
	Name          string
	DominantColor string   // W, U, B, R, G, M (multicolor), C (colorless)
	ColorIdentity []string // distinct colors in WUBRG order, or {"C"} for colorless
	Groups        []TypeGroup
}

// Organize takes a raw parsed decklist and a map of Scryfall card data, then
// returns a fully organized Deck with cards grouped by type in display order,
// sorted by CMC within each group (lands sorted alphabetically), and the
// deck's dominant color computed from color identity frequencies.
func Organize(raw parser.RawDeck, cards map[string]*scryfall.Card) Deck {
	grouped := make(map[string][]DeckCard)

	for _, entry := range raw.Cards {
		card, ok := cards[entry.Name]
		if !ok {
			continue
		}
		typeName := classifyType(card.TypeLine)
		grouped[typeName] = append(grouped[typeName], DeckCard{
			Name:     card.Name,
			Quantity: entry.Quantity,
			CMC:      card.CMC,
		})
	}

	// Sort within each group.
	for typeName, groupCards := range grouped {
		if typeName == "Land" {
			sort.Slice(groupCards, func(i, j int) bool {
				return groupCards[i].Name < groupCards[j].Name
			})
		} else {
			sort.Slice(groupCards, func(i, j int) bool {
				if groupCards[i].CMC != groupCards[j].CMC {
					return groupCards[i].CMC < groupCards[j].CMC
				}
				return groupCards[i].Name < groupCards[j].Name
			})
		}
		grouped[typeName] = groupCards
	}

	// Build groups in display order.
	var groups []TypeGroup
	for _, typeName := range displayOrder {
		if groupCards, ok := grouped[typeName]; ok {
			total := 0
			for _, c := range groupCards {
				total += c.Quantity
			}
			groups = append(groups, TypeGroup{
				TypeName: typeName,
				Cards:    groupCards,
				Count:    total,
			})
		}
	}

	colors := deckColors(raw.Cards, cards)
	dominant := dominantColor(colors)
	if raw.ColorOverride != "" {
		dominant = raw.ColorOverride
	}
	return Deck{
		Name:          raw.Name,
		DominantColor: dominant,
		ColorIdentity: colors,
		Groups:        groups,
	}
}

// displayOrder defines the display order for card type groups.
var displayOrder = []string{
	"Creature",
	"Planeswalker",
	"Instant",
	"Sorcery",
	"Enchantment",
	"Artifact",
	"Land",
}

// classifyPriority defines the order in which types are checked when a card
// has multiple types. Land is checked first so that creature lands (e.g.,
// Dryad Arbor) are classified as lands.
var classifyPriority = []string{
	"Land",
	"Creature",
	"Planeswalker",
	"Instant",
	"Sorcery",
	"Enchantment",
	"Artifact",
}

// wubrg defines the canonical MTG color wheel order for sorting.
var wubrg = []string{"W", "U", "B", "R", "G"}

// validColors is the set of accepted color codes for deck color override.
var validColors = map[string]bool{
	"W": true, "U": true, "B": true, "R": true, "G": true,
	"M": true, "C": true,
}

// ValidColor reports whether s is a valid deck color code.
func ValidColor(s string) bool {
	return validColors[s]
}

// classifyType maps a Scryfall type line to one of the canonical type group
// names. Land takes priority so that creature lands are classified as lands.
func classifyType(typeLine string) string {
	for _, t := range classifyPriority {
		if strings.Contains(typeLine, t) {
			return t
		}
	}
	return "Land" // fallback
}

// deckColors collects distinct casting cost colors from all cards and returns
// them in WUBRG order. Uses Colors (casting cost) rather than ColorIdentity
// (which includes activated abilities). Returns {"C"} for colorless decks.
func deckColors(entries []parser.CardEntry, cards map[string]*scryfall.Card) []string {
	present := make(map[string]bool)
	for _, e := range entries {
		c, ok := cards[e.Name]
		if !ok {
			continue
		}
		for _, color := range c.Colors {
			present[color] = true
		}
	}

	if len(present) == 0 {
		return []string{"C"}
	}

	var colors []string
	for _, c := range wubrg {
		if present[c] {
			colors = append(colors, c)
		}
	}
	return colors
}

// dominantColor determines the deck's color identity. Returns "C" if no colors
// are present, "M" if two or more distinct colors appear, or the single color
// letter if the deck is mono-colored.
func dominantColor(colors []string) string {
	if len(colors) == 1 && colors[0] == "C" {
		return "C"
	}
	if len(colors) == 1 {
		return colors[0]
	}
	return "M"
}
