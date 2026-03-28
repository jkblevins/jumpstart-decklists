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

- A colored top bar matching the deck's dominant color identity
- Deck name centered in bold
- Cards grouped by type (Creature, Planeswalker, Instant, Sorcery, Enchantment, Artifact, Land)
- Sorted by mana cost within each group
- Basic lands displayed as `Mountain (3)` instead of `3 Mountain`

## Card Data

Card metadata is fetched from [Scryfall](https://scryfall.com/) and cached locally at `~/.cache/jumpforge/` for one week. API rate limits are respected (100ms between requests).

## License

MIT
