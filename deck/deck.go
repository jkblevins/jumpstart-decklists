// Package deck organizes parsed decklists into grouped, sorted structures
// suitable for rendering. It classifies cards by type, sorts within groups
// by mana cost, and determines the deck's dominant color identity.
package deck

import (
	"sort"
	"strings"

	"jumpforge/parser"
	"jumpforge/scryfall"
)

// typeOrder defines the display order for card type groups.
var typeOrder = []string{
	"Creature",
	"Planeswalker",
	"Instant",
	"Sorcery",
	"Enchantment",
	"Artifact",
	"Land",
}

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
	DominantColor string // W, U, B, R, G, M (multicolor), C (colorless)
	Groups        []TypeGroup
}

// classifyType maps a Scryfall type line to one of the canonical type group
// names. The first match in typeOrder wins, so "Artifact Creature" becomes
// "Creature" rather than "Artifact".
func classifyType(typeLine string) string {
	for _, t := range typeOrder {
		if strings.Contains(typeLine, t) {
			return t
		}
	}
	return "Land" // fallback
}

// dominantColor determines which single color appears most frequently across
// all card entries weighted by quantity. Returns "C" if no colors are present,
// or "M" if multiple colors are tied for the lead.
func dominantColor(entries []parser.CardEntry, cards map[string]*scryfall.Card) string {
	counts := make(map[string]int)
	for _, e := range entries {
		c, ok := cards[e.Name]
		if !ok {
			continue
		}
		for _, color := range c.ColorIdentity {
			counts[color] += e.Quantity
		}
	}

	if len(counts) == 0 {
		return "C"
	}

	// Find max count.
	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}

	// Check if a single color strictly dominates.
	var winners []string
	for color, count := range counts {
		if count == maxCount {
			winners = append(winners, color)
		}
	}

	if len(winners) > 1 {
		return "M" // multicolor -- no single color strictly dominates
	}

	return winners[0]
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
	for _, typeName := range typeOrder {
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

	return Deck{
		Name:          raw.Name,
		DominantColor: dominantColor(raw.Cards, cards),
		Groups:        groups,
	}
}
