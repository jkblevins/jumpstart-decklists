package deck

import (
	"testing"

	"jumpforge/parser"
	"jumpforge/scryfall"
)

func TestClassifyType(t *testing.T) {
	tests := []struct {
		typeLine string
		want     string
	}{
		{"Creature — Goblin Warrior", "Creature"},
		{"Artifact Creature — Golem", "Creature"},
		{"Enchantment Creature — God", "Creature"},
		{"Legendary Planeswalker — Nissa", "Planeswalker"},
		{"Instant", "Instant"},
		{"Sorcery", "Sorcery"},
		{"Enchantment — Aura", "Enchantment"},
		{"Artifact — Equipment", "Artifact"},
		{"Basic Land — Mountain", "Land"},
		{"Land", "Land"},
	}
	for _, tc := range tests {
		t.Run(tc.typeLine, func(t *testing.T) {
			got := classifyType(tc.typeLine)
			if got != tc.want {
				t.Errorf("classifyType(%q) = %q, want %q", tc.typeLine, got, tc.want)
			}
		})
	}
}

func TestDominantColorMono(t *testing.T) {
	cards := map[string]*scryfall.Card{
		"Goblin Guide":   {ColorIdentity: []string{"R"}},
		"Lightning Bolt": {ColorIdentity: []string{"R"}},
		"Mountain":       {ColorIdentity: []string{}},
	}
	entries := []parser.CardEntry{
		{Quantity: 1, Name: "Goblin Guide"},
		{Quantity: 2, Name: "Lightning Bolt"},
		{Quantity: 3, Name: "Mountain"},
	}
	got := dominantColor(deckColors(entries, cards))
	if got != "R" {
		t.Errorf("expected R (mono-red), got %q", got)
	}
}

func TestDominantColorMultipleColors(t *testing.T) {
	cards := map[string]*scryfall.Card{
		"Goblin Guide":   {ColorIdentity: []string{"R"}},
		"Lightning Bolt": {ColorIdentity: []string{"R"}},
		"Mountain":       {ColorIdentity: []string{}},
		"Llanowar Elves": {ColorIdentity: []string{"G"}},
	}
	entries := []parser.CardEntry{
		{Quantity: 1, Name: "Goblin Guide"},
		{Quantity: 2, Name: "Lightning Bolt"},
		{Quantity: 3, Name: "Mountain"},
		{Quantity: 1, Name: "Llanowar Elves"},
	}
	// Both R and G appear => multicolor
	got := dominantColor(deckColors(entries, cards))
	if got != "M" {
		t.Errorf("expected M (multicolor), got %q", got)
	}
}

func TestDominantColorColorless(t *testing.T) {
	cards := map[string]*scryfall.Card{
		"Mountain": {ColorIdentity: []string{}},
		"Forest":   {ColorIdentity: []string{}},
	}
	entries := []parser.CardEntry{
		{Quantity: 3, Name: "Mountain"},
		{Quantity: 3, Name: "Forest"},
	}
	got := dominantColor(deckColors(entries, cards))
	if got != "C" {
		t.Errorf("expected C (colorless), got %q", got)
	}
}

func TestDominantColorMulticolor(t *testing.T) {
	cards := map[string]*scryfall.Card{
		"CardR": {ColorIdentity: []string{"R"}},
		"CardG": {ColorIdentity: []string{"G"}},
	}
	entries := []parser.CardEntry{
		{Quantity: 2, Name: "CardR"},
		{Quantity: 2, Name: "CardG"},
	}
	// Tied: no single color strictly dominates => multicolor
	got := dominantColor(deckColors(entries, cards))
	if got != "M" {
		t.Errorf("expected M (multicolor), got %q", got)
	}
}

func TestOrganize(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Test Deck",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Goblin Guide"},
			{Quantity: 2, Name: "Lightning Bolt"},
			{Quantity: 3, Name: "Mountain"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Goblin Guide": {
			Name:          "Goblin Guide",
			TypeLine:      "Creature — Goblin Scout",
			CMC:           1,
			ColorIdentity: []string{"R"},
		},
		"Lightning Bolt": {
			Name:          "Lightning Bolt",
			TypeLine:      "Instant",
			CMC:           1,
			ColorIdentity: []string{"R"},
		},
		"Mountain": {
			Name:          "Mountain",
			TypeLine:      "Basic Land — Mountain",
			ColorIdentity: []string{},
		},
	}
	d := Organize(raw, cards)

	if d.Name != "Test Deck" {
		t.Errorf("expected name 'Test Deck', got %q", d.Name)
	}
	if d.DominantColor != "R" {
		t.Errorf("expected dominant color R, got %q", d.DominantColor)
	}
	// Should have 3 groups: Creature, Instant, Land
	if len(d.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(d.Groups))
	}
	if d.Groups[0].TypeName != "Creature" {
		t.Errorf("expected first group Creature, got %q", d.Groups[0].TypeName)
	}
	if d.Groups[1].TypeName != "Instant" {
		t.Errorf("expected second group Instant, got %q", d.Groups[1].TypeName)
	}
	if d.Groups[2].TypeName != "Land" {
		t.Errorf("expected third group Land, got %q", d.Groups[2].TypeName)
	}
}

func TestSortWithinGroup(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Sort Test",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Goblin Chainwhirler"},
			{Quantity: 1, Name: "Goblin Guide"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Goblin Chainwhirler": {
			Name: "Goblin Chainwhirler", TypeLine: "Creature — Goblin Warrior",
			CMC: 3, ColorIdentity: []string{"R"},
		},
		"Goblin Guide": {
			Name: "Goblin Guide", TypeLine: "Creature — Goblin Scout",
			CMC: 1, ColorIdentity: []string{"R"},
		},
	}
	d := Organize(raw, cards)
	creatures := d.Groups[0]
	if creatures.Cards[0].Name != "Goblin Guide" {
		t.Errorf("expected Goblin Guide first (CMC 1), got %q", creatures.Cards[0].Name)
	}
}
