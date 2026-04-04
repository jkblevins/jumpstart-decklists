// layout.go defines the spatial dimensions, spacing, color schemes, and page
// composition for decklist cards. It answers "where and how big" — card size,
// margins, grid arrangement, and color-to-scheme mapping. Rendering logic
// (drawing shapes, text, and images) lives in render.go.
package pdf

import (
	"fmt"
	"strings"

	"github.com/signintech/gopdf"

	"jumpstart-decklists/deck"
)

// Card dimensions in PDF points (1 inch = 72 points).
const (
	cardW = 172.9 // 61mm (1mm inset from MTG card edge for rounded corners)
	cardH = 243.7 // 86mm (1mm inset from MTG card edge for rounded corners)

	pageW = 792 // 11 inches (US Letter landscape)
	pageH = 612 // 8.5 inches (US Letter landscape)
)

// Card structure constants.
const (
	outerBorderW = 2.5  // thick outer frame
	innerBorderW = 0.5  // thin inner frame line
	innerInset   = 5.0  // distance from outer edge to inner frame
	colorBarH    = 16.0 // top color identity bar (holds deck title)
	marginX      = 8.0  // text left margin from inner frame
	marginY      = 10.0 // text top margin from color bar
)

// Typography constants.
const (
	fontTitle    = 10.0
	fontHeader   = 8.0
	fontBody     = 7.5
	lineHeight   = 9.0
	indentX      = 5.0 // extra indent for card entries under headers
	groupSpacing = 2.0 // vertical space between type groups
)

// Mana icon constants.
const (
	iconSize = 12.0 // icon dimensions in PDF points
	iconGap  = 2.0  // space between icons
)

// Grid layout constants for batch mode.
const (
	gridCols = 4   // card columns per page in batch mode
	gridRows = 2   // card rows per page in batch mode
	cardGap  = 0.0 // no gap — bleed areas fill the space between cards
	bleed    = 8.5 // ~3mm bleed around each card for cutting margin
)

// colorScheme defines the border/bar color and background tint for a color identity.
type colorScheme struct {
	border [3]uint8 // color bar and outer border
	bg     [3]uint8 // card background fill
}

// colorMap maps single-letter color identities to their visual scheme.
var colorMap = map[string]colorScheme{
	"W": {border: [3]uint8{190, 180, 155}, bg: [3]uint8{252, 250, 245}}, // White: silvery-warm border, ivory bg
	"U": {border: [3]uint8{14, 104, 171}, bg: [3]uint8{235, 244, 252}},  // Blue: blue border, light blue bg
	"B": {border: [3]uint8{50, 40, 50}, bg: [3]uint8{240, 238, 240}},    // Black: near-black border, light gray-purple bg
	"R": {border: [3]uint8{211, 32, 41}, bg: [3]uint8{250, 240, 238}},   // Red: red border, light pink bg
	"G": {border: [3]uint8{0, 115, 62}, bg: [3]uint8{238, 248, 238}},    // Green: green border, light green bg
	"M": {border: [3]uint8{185, 150, 28}, bg: [3]uint8{252, 248, 232}},  // Multicolor: rich gold border, warm gold bg
	"C": {border: [3]uint8{158, 158, 158}, bg: [3]uint8{245, 245, 245}}, // Colorless: gray border, light gray bg
}

// RenderSingle creates a card-sized PDF containing a single decklist card
// and writes it to outPath.
func RenderSingle(d deck.Deck, outPath string) error {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: cardW + 2*bleed, H: cardH + 2*bleed},
	})
	if err := setupFonts(pdf); err != nil {
		return fmt.Errorf("setup fonts: %w", err)
	}
	pdf.AddPage()
	renderCard(pdf, d, bleed, bleed)
	return pdf.WritePdf(outPath)
}

// RenderBatch creates a landscape letter-sized PDF with decklist cards arranged
// in a 4x2 grid. If more than 8 decks are provided, additional pages are added.
func RenderBatch(decks []deck.Deck, outPath string) error {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: pageW, H: pageH},
	})
	if err := setupFonts(pdf); err != nil {
		return fmt.Errorf("setup fonts: %w", err)
	}

	perPage := gridCols * gridRows

	// Each cell includes the card plus bleed on all sides.
	cellW := cardW + 2*bleed
	cellH := cardH + 2*bleed

	// Center the grid on the page.
	gridW := float64(gridCols) * cellW
	gridH := float64(gridRows) * cellH
	offsetX := (pageW - gridW) / 2
	offsetY := (pageH - gridH) / 2

	for i, d := range decks {
		if i%perPage == 0 {
			pdf.AddPage()
		}
		slot := i % perPage
		col := slot % gridCols
		row := slot / gridCols
		// Position card content at bleed offset within each cell.
		x := offsetX + float64(col)*cellW + bleed
		y := offsetY + float64(row)*cellH + bleed
		renderCard(pdf, d, x, y)
	}

	return pdf.WritePdf(outPath)
}

// OutputFileName converts a deck name to a PDF filename by lowercasing,
// replacing spaces with dashes, and appending ".pdf".
func OutputFileName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-")) + ".pdf"
}
