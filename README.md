# jumpforge

A Go CLI tool that takes a text decklist for a Magic: The Gathering Jumpstart pack and generates a printable PDF decklist card with a color-identity-matched border. Each card is sized at 63 x 88mm (standard MTG card dimensions) so it can be sleeved alongside the deck. See [Output](#output) for details.

![Example batch output showing six Jumpstart decklists](docs/example.png)

## Install

```bash
go install github.com/jkblevins/jumpforge@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Or build from source:

```bash
git clone https://github.com/jkblevins/jumpforge.git
cd jumpforge
go build -o jumpforge .
```

## Usage

```bash
jumpforge decklist.txt
```

### Input Format

Plain text file with one deck:

```
Goblin Rush

1 Goblin Guide
2 Lightning Bolt
1 Reckless Bushwhacker
3 Mountain
```

- First non-blank line is the deck name.
- Remaining lines: `<quantity> <card name>`.
- Blank lines and `//` comments are ignored.
- Optional color override: append `[X]` to the deck name (see [Color Override](#color-override)).

### Batch Mode

Separate multiple decks with `---`:

```
Goblin Rush

1 Goblin Guide
2 Lightning Bolt
3 Mountain
---
Forest Friends

1 Llanowar Elves
2 Giant Growth
4 Forest
```

Single deck produces a card-sized PDF. Multiple decks produce a letter-size PDF with a 3x3 grid.

## Output

Each card includes:

- A color-matched border and tinted background based on the deck's color identity
- Deck name centered in white on the color bar
- Cards grouped by type with bold headers
- Singles displayed by name only (e.g., `Gruul Signet`), multiples as `Name (N)` (e.g., `Mountain (5)`)

### Color Identity

The border and background color is determined by the deck's color identity:

- **Mono-colored** decks (all cards share one color) get that color's scheme
- **Multi-colored** decks (cards with 2+ distinct colors) get a gold/multicolor scheme
- **Colorless** decks (no colored cards) get a gray scheme

#### Color Override

You can override the auto-detected color by appending a color code in brackets to the deck name:

```
Azorius Senate 1 [W]

1 Swords to Plowshares
1 Counterspell
---
Azorius Senate 2 [U]

1 Counterspell
1 Swords to Plowshares
```

Valid codes: `W` (white), `U` (blue), `B` (black), `R` (red), `G` (green), `M` (multicolor), `C` (colorless). The bracket suffix is stripped from the deck name. The override only affects the border and background color — mana symbols in the title bar are still auto-detected.

To use literal brackets in a deck name, escape them with backslashes: `Goblins \[Part 1\]`.

### Card Grouping

Cards are grouped by type in this order:

1. Creatures
2. Planeswalkers
3. Instants
4. Sorceries
5. Enchantments
6. Artifacts
7. Lands

#### Multi-type cards

Cards with multiple types (e.g., "Artifact Creature") are classified by the most specific type using the priority order above. An Artifact Creature goes under Creatures, not Artifacts. An Enchantment Creature goes under Creatures, not Enchantments.

### Sorting

Within each group, cards are sorted by converted mana cost (ascending), then alphabetically by name for ties. Lands are sorted alphabetically only.

## Card Data

Card metadata is fetched from [Scryfall](https://scryfall.com/) and cached locally at `~/.cache/jumpforge/` for one week. API rate limits are respected (100ms between requests).
