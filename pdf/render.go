// Package pdf renders organized decklists as printable PDF cards using gopdf.
// Each card displays the deck name, a color-identity bar, and card entries
// grouped by type with quantities.
package pdf

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/signintech/gopdf"

	"jumpforge/deck"
)

// Card dimensions in PDF points (1 inch = 72 points).
const (
	cardW = 180 // 2.5 inches
	cardH = 252 // 3.5 inches

	pageW = 612 // 8.5 inches (US Letter)
	pageH = 792 // 11 inches (US Letter)

	colorBarH   = 11
	borderW     = 0.75
	marginX     = 7
	marginY     = 5
	fontTitle    = 9
	fontHeader   = 7.5
	fontBody     = 7
	lineHeight   = 9
)

// colorMap maps single-letter color identities to RGB values used for the
// top color bar on each card.
var colorMap = map[string][3]uint8{
	"W": {212, 175, 55},  // gold/cream
	"U": {14, 104, 171},  // blue
	"B": {59, 47, 74},    // dark purple
	"R": {211, 32, 41},   // red
	"G": {0, 115, 62},    // green
	"M": {212, 175, 55},  // gold (multicolor)
	"C": {158, 158, 158}, // gray
}

// basicLands lists the five basic land names for special display formatting.
var basicLands = map[string]bool{
	"Plains":   true,
	"Island":   true,
	"Swamp":    true,
	"Mountain": true,
	"Forest":   true,
}

// fontsDir returns the absolute path to the fonts directory, located relative
// to this source file.
func fontsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filepath.Dir(filename)), "fonts")
}

// setupFonts registers the DejaVu Sans regular and bold fonts with the PDF.
func setupFonts(pdf *gopdf.GoPdf) error {
	dir := fontsDir()
	regular := filepath.Join(dir, "DejaVuSans.ttf")
	bold := filepath.Join(dir, "DejaVuSans-Bold.ttf")

	if err := pdf.AddTTFFont("body", regular); err != nil {
		return fmt.Errorf("add regular font: %w", err)
	}
	if err := pdf.AddTTFFontWithOption("body", bold, gopdf.TtfOption{Style: gopdf.Bold}); err != nil {
		return fmt.Errorf("add bold font: %w", err)
	}
	return nil
}

// pluralType converts a singular card type name to its plural form for
// display in group headers.
func pluralType(t string) string {
	switch t {
	case "Sorcery":
		return "Sorceries"
	case "Land":
		return "Lands"
	default:
		return t + "s"
	}
}

// renderCard draws a single decklist card onto the PDF at the given x/y
// offset. It renders the outer border, color bar, deck name, and all card
// entries grouped by type.
func renderCard(pdf *gopdf.GoPdf, d deck.Deck, x, y float64) {
	// Outer border (stroke only).
	pdf.SetStrokeColor(40, 40, 40)
	pdf.SetLineWidth(borderW)
	pdf.Rectangle(x, y, x+cardW, y+cardH, "D", 0, 0)

	// Color bar.
	rgb := colorMap["C"]
	if c, ok := colorMap[d.DominantColor]; ok {
		rgb = c
	}
	pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
	pdf.Rectangle(x+borderW, y+borderW, x+cardW-borderW, y+borderW+colorBarH, "F", 0, 0)

	// Deck name centered below color bar.
	curY := y + borderW + colorBarH + marginY
	pdf.SetFont("body", "B", fontTitle)
	pdf.SetTextColor(30, 30, 30)

	nameW, _ := pdf.MeasureTextWidth(d.Name)
	nameX := x + (cardW-nameW)/2
	pdf.SetXY(nameX, curY)
	pdf.Text(d.Name)
	curY += lineHeight + 2

	// Card groups.
	for _, g := range d.Groups {
		// Group header.
		header := fmt.Sprintf("%s (%d):", pluralType(g.TypeName), g.Count)
		pdf.SetFont("body", "B", fontHeader)
		pdf.SetXY(x+marginX, curY)
		pdf.Text(header)
		curY += lineHeight

		// Card entries.
		pdf.SetFont("body", "", fontBody)
		for _, c := range g.Cards {
			var line string
			if basicLands[c.Name] {
				line = fmt.Sprintf("%s (%d)", c.Name, c.Quantity)
			} else {
				line = fmt.Sprintf("%d %s", c.Quantity, c.Name)
			}
			pdf.SetXY(x+marginX+5, curY)
			pdf.Text(line)
			curY += lineHeight
		}
		curY += 2 // spacing between groups
	}
}
