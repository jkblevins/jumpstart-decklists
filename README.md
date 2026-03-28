# jumpforge

A Go CLI tool that takes a text decklist for a Magic: The Gathering Jumpstart pack and generates a printable PDF decklist card with a color-identity-matched border.

## Install

```bash
go install jumpforge@latest
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

Single deck produces a card-sized PDF (2.5" x 3.5"). Multiple decks produce a letter-size PDF with a 3x3 grid.

## Output

Each card includes:

- A color-matched border and tinted background based on the deck's color identity
- Deck name centered in white on the color bar
- Cards grouped by type with bold headers
- Basic lands displayed as `Mountain (3)` instead of `3 Mountain`

### Color Identity

The border and background color is determined by the deck's color identity:

- **Mono-colored** decks (all cards share one color) get that color's scheme
- **Multi-colored** decks (cards with 2+ distinct colors) get a gold/multicolor scheme
- **Colorless** decks (no colored cards) get a gray scheme

### Card Grouping

Cards are grouped by type in this order:

1. Creatures
2. Planeswalkers
3. Instants
4. Sorceries
5. Enchantments
6. Artifacts
7. Lands

Cards with multiple types (e.g., "Artifact Creature") are classified by the most specific type using the priority order above. An Artifact Creature goes under Creatures, not Artifacts. An Enchantment Creature goes under Creatures, not Enchantments.

### Sorting

Within each group, cards are sorted by converted mana cost (ascending), then alphabetically by name for ties. Lands are sorted alphabetically only.

## Card Data

Card metadata is fetched from [Scryfall](https://scryfall.com/) and cached locally at `~/.cache/jumpforge/` for one week. API rate limits are respected (100ms between requests).
