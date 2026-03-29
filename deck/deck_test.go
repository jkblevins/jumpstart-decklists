package deck

import (
	"testing"

	"jumpstart-decklists/parser"
	"jumpstart-decklists/scryfall"
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
		{"Land Creature — Dryad", "Land"},
		{"Artifact Land", "Land"},
		{"Enchantment Land", "Land"},
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
		"Goblin Guide":   {Colors: []string{"R"}},
		"Lightning Bolt": {Colors: []string{"R"}},
		"Mountain":       {Colors: []string{}},
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
		"Goblin Guide":   {Colors: []string{"R"}},
		"Lightning Bolt": {Colors: []string{"R"}},
		"Mountain":       {Colors: []string{}},
		"Llanowar Elves": {Colors: []string{"G"}},
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
		"Mountain": {Colors: []string{}},
		"Forest":   {Colors: []string{}},
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
		"CardR": {Colors: []string{"R"}},
		"CardG": {Colors: []string{"G"}},
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
			Name:     "Goblin Guide",
			TypeLine: "Creature — Goblin Scout",
			CMC:      1,
			Colors:   []string{"R"},
		},
		"Lightning Bolt": {
			Name:     "Lightning Bolt",
			TypeLine: "Instant",
			CMC:      1,
			Colors:   []string{"R"},
		},
		"Mountain": {
			Name:     "Mountain",
			TypeLine: "Basic Land — Mountain",
			Colors:   []string{},
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

func TestDeckColorsWUBRGOrder(t *testing.T) {
	cards := map[string]*scryfall.Card{
		"CardG": {Colors: []string{"G"}},
		"CardW": {Colors: []string{"W"}},
		"CardR": {Colors: []string{"R"}},
	}
	entries := []parser.CardEntry{
		{Quantity: 1, Name: "CardG"},
		{Quantity: 1, Name: "CardW"},
		{Quantity: 1, Name: "CardR"},
	}
	got := deckColors(entries, cards)
	want := []string{"W", "R", "G"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %s, got %s", i, want[i], got[i])
		}
	}
}

func TestDeckColorsFiveColor(t *testing.T) {
	cards := map[string]*scryfall.Card{
		"Card1": {Colors: []string{"R", "G"}},
		"Card2": {Colors: []string{"W", "U", "B"}},
	}
	entries := []parser.CardEntry{
		{Quantity: 1, Name: "Card1"},
		{Quantity: 1, Name: "Card2"},
	}
	got := deckColors(entries, cards)
	want := []string{"W", "U", "B", "R", "G"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %s, got %s", i, want[i], got[i])
		}
	}
}

func TestOrganizeColorIdentity(t *testing.T) {
	raw := parser.RawDeck{
		Name:  "Test",
		Cards: []parser.CardEntry{{Quantity: 1, Name: "Bolt"}},
	}
	cards := map[string]*scryfall.Card{
		"Bolt": {Name: "Bolt", TypeLine: "Instant", CMC: 1, Colors: []string{"R"}},
	}
	d := Organize(raw, cards)
	if len(d.ColorIdentity) != 1 || d.ColorIdentity[0] != "R" {
		t.Errorf("expected ColorIdentity [R], got %v", d.ColorIdentity)
	}
}

func TestOrganizeMissingCards(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Missing",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Real Card"},
			{Quantity: 1, Name: "Fake Card"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Real Card": {Name: "Real Card", TypeLine: "Creature — Human", CMC: 1, Colors: []string{"W"}},
	}
	d := Organize(raw, cards)
	total := 0
	for _, g := range d.Groups {
		total += g.Count
	}
	if total != 1 {
		t.Errorf("expected 1 card (missing card skipped), got %d", total)
	}
}

func TestOrganizeEmpty(t *testing.T) {
	raw := parser.RawDeck{
		Name:  "Empty",
		Cards: []parser.CardEntry{{Quantity: 1, Name: "Nonexistent"}},
	}
	cards := map[string]*scryfall.Card{}
	d := Organize(raw, cards)
	if len(d.Groups) != 0 {
		t.Errorf("expected 0 groups for no valid cards, got %d", len(d.Groups))
	}
}

func TestLandAlphabeticalSort(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Lands",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Swamp"},
			{Quantity: 1, Name: "Forest"},
			{Quantity: 1, Name: "Island"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Swamp":  {Name: "Swamp", TypeLine: "Basic Land — Swamp", Colors: []string{}},
		"Forest": {Name: "Forest", TypeLine: "Basic Land — Forest", Colors: []string{}},
		"Island": {Name: "Island", TypeLine: "Basic Land — Island", Colors: []string{}},
	}
	d := Organize(raw, cards)
	lands := d.Groups[0]
	if lands.Cards[0].Name != "Forest" {
		t.Errorf("expected Forest first (alphabetical), got %q", lands.Cards[0].Name)
	}
	if lands.Cards[1].Name != "Island" {
		t.Errorf("expected Island second, got %q", lands.Cards[1].Name)
	}
	if lands.Cards[2].Name != "Swamp" {
		t.Errorf("expected Swamp third, got %q", lands.Cards[2].Name)
	}
}

func TestGroupCount(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Counts",
		Cards: []parser.CardEntry{
			{Quantity: 3, Name: "Bolt"},
			{Quantity: 2, Name: "Shock"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Bolt":  {Name: "Bolt", TypeLine: "Instant", CMC: 1, Colors: []string{"R"}},
		"Shock": {Name: "Shock", TypeLine: "Instant", CMC: 1, Colors: []string{"R"}},
	}
	d := Organize(raw, cards)
	if d.Groups[0].Count != 5 {
		t.Errorf("expected count 5, got %d", d.Groups[0].Count)
	}
}

func TestSortSameCMCAlphabetical(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Tiebreak",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Zephyr"},
			{Quantity: 1, Name: "Alpha"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Zephyr": {Name: "Zephyr", TypeLine: "Creature — Elemental", CMC: 2, Colors: []string{"U"}},
		"Alpha":  {Name: "Alpha", TypeLine: "Creature — Human", CMC: 2, Colors: []string{"W"}},
	}
	d := Organize(raw, cards)
	if d.Groups[0].Cards[0].Name != "Alpha" {
		t.Errorf("expected Alpha first (alphabetical tiebreak), got %q", d.Groups[0].Cards[0].Name)
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
			CMC: 3, Colors: []string{"R"},
		},
		"Goblin Guide": {
			Name: "Goblin Guide", TypeLine: "Creature — Goblin Scout",
			CMC: 1, Colors: []string{"R"},
		},
	}
	d := Organize(raw, cards)
	creatures := d.Groups[0]
	if creatures.Cards[0].Name != "Goblin Guide" {
		t.Errorf("expected Goblin Guide first (CMC 1), got %q", creatures.Cards[0].Name)
	}
}

func TestOrganizeWithColorOverride(t *testing.T) {
	raw := parser.RawDeck{
		Name:          "Azorius Senate",
		ColorOverride: "W",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Swords"},
			{Quantity: 1, Name: "Counterspell"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Swords":       {Name: "Swords", TypeLine: "Instant", CMC: 1, Colors: []string{"W"}},
		"Counterspell": {Name: "Counterspell", TypeLine: "Instant", CMC: 2, Colors: []string{"U"}},
	}
	d := Organize(raw, cards)
	if d.DominantColor != "W" {
		t.Errorf("expected DominantColor 'W' (override), got %q", d.DominantColor)
	}
	// ColorIdentity should still be auto-detected
	if len(d.ColorIdentity) != 2 {
		t.Fatalf("expected 2 colors in identity, got %d", len(d.ColorIdentity))
	}
}

func TestOrganizeWithoutColorOverride(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Azorius Senate",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Swords"},
			{Quantity: 1, Name: "Counterspell"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Swords":       {Name: "Swords", TypeLine: "Instant", CMC: 1, Colors: []string{"W"}},
		"Counterspell": {Name: "Counterspell", TypeLine: "Instant", CMC: 2, Colors: []string{"U"}},
	}
	d := Organize(raw, cards)
	if d.DominantColor != "M" {
		t.Errorf("expected DominantColor 'M' (auto-detected multicolor), got %q", d.DominantColor)
	}
}

func TestLandCreatureDisplaysUnderLandsLast(t *testing.T) {
	raw := parser.RawDeck{
		Name: "Dryad Test",
		Cards: []parser.CardEntry{
			{Quantity: 1, Name: "Dryad Arbor"},
			{Quantity: 1, Name: "Goblin Guide"},
			{Quantity: 1, Name: "Lightning Bolt"},
		},
	}
	cards := map[string]*scryfall.Card{
		"Dryad Arbor":    {Name: "Dryad Arbor", TypeLine: "Land Creature — Dryad", CMC: 0, Colors: []string{"G"}},
		"Goblin Guide":   {Name: "Goblin Guide", TypeLine: "Creature — Goblin Scout", CMC: 1, Colors: []string{"R"}},
		"Lightning Bolt": {Name: "Lightning Bolt", TypeLine: "Instant", CMC: 1, Colors: []string{"R"}},
	}
	d := Organize(raw, cards)
	if len(d.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(d.Groups))
	}
	// Creatures first, then Instants, then Lands last (display order)
	if d.Groups[0].TypeName != "Creature" {
		t.Errorf("expected first group Creature, got %q", d.Groups[0].TypeName)
	}
	if d.Groups[1].TypeName != "Instant" {
		t.Errorf("expected second group Instant, got %q", d.Groups[1].TypeName)
	}
	if d.Groups[2].TypeName != "Land" {
		t.Errorf("expected third group Land, got %q", d.Groups[2].TypeName)
	}
	// Dryad Arbor should be under Lands, not Creatures
	if d.Groups[2].Cards[0].Name != "Dryad Arbor" {
		t.Errorf("expected Dryad Arbor under Lands, got %q", d.Groups[2].Cards[0].Name)
	}
}

func TestValidColor(t *testing.T) {
	valid := []string{"W", "U", "B", "R", "G", "M", "C"}
	for _, c := range valid {
		if !ValidColor(c) {
			t.Errorf("ValidColor(%q) = false, want true", c)
		}
	}
	invalid := []string{"w", "X", "WU", "", "Gold", "1"}
	for _, c := range invalid {
		if ValidColor(c) {
			t.Errorf("ValidColor(%q) = true, want false", c)
		}
	}
}
