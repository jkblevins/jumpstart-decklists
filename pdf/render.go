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

	colorBarH  = 11
	borderW    = 0.75
	marginX    = 7
	marginY    = 5
	fontTitle  = 9
	fontHeader = 7.5
	fontBody   = 7
	lineHeight = 9
)

// colorMap maps single-letter color identities to RGB values used for the
// top color bar on each card.
var colorMap = map[string][3]uint8{
	"W": {212, 175, 55},  // White: gold/cream
	"U": {14, 104, 171},  // Blue: blue
	"B": {59, 47, 74},    // Black: dark purple
	"R": {211, 32, 41},   // Red: red
	"G": {0, 115, 62},    // Green: green
	"M": {212, 175, 55},  // Multicolor: gold
	"C": {158, 158, 158}, // Colorless: gray
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

// cardRenderer holds the state needed to draw a single decklist card.
type cardRenderer struct {
	pdf  *gopdf.GoPdf
	x, y float64 // card origin (upper-left corner)
	curY float64 // vertical cursor for text placement
}

// drawBorder draws the thin outer border around the card.
func (cr *cardRenderer) drawBorder() {
	cr.pdf.SetStrokeColor(40, 40, 40)
	cr.pdf.SetLineWidth(borderW)
	cr.pdf.Rectangle(cr.x, cr.y, cr.x+cardW, cr.y+cardH, "D", 0, 0)
}

// drawColorBar fills the top bar with the deck's dominant color.
func (cr *cardRenderer) drawColorBar(color string) {
	rgb := colorMap["C"]
	if c, ok := colorMap[color]; ok {
		rgb = c
	}
	cr.pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
	cr.pdf.Rectangle(cr.x+borderW, cr.y+borderW, cr.x+cardW-borderW, cr.y+borderW+colorBarH, "F", 0, 0)
}

// drawTitle renders the deck name centered in bold below the color bar.
func (cr *cardRenderer) drawTitle(name string) {
	cr.pdf.SetFont("body", "B", fontTitle)
	cr.pdf.SetTextColor(30, 30, 30)

	nameW, _ := cr.pdf.MeasureTextWidth(name)
	nameX := cr.x + (cardW-nameW)/2
	cr.pdf.SetXY(nameX, cr.curY)
	cr.pdf.Text(name)
	cr.curY += lineHeight + 2
}

// drawGroups renders all type groups with headers and indented card entries.
func (cr *cardRenderer) drawGroups(groups []deck.TypeGroup) {
	for _, g := range groups {
		header := fmt.Sprintf("%s (%d):", pluralType(g.TypeName), g.Count)
		cr.pdf.SetFont("body", "B", fontHeader)
		cr.pdf.SetXY(cr.x+marginX, cr.curY)
		cr.pdf.Text(header)
		cr.curY += lineHeight

		cr.pdf.SetFont("body", "", fontBody)
		for _, c := range g.Cards {
			var line string
			if basicLands[c.Name] {
				line = fmt.Sprintf("%s (%d)", c.Name, c.Quantity)
			} else {
				line = fmt.Sprintf("%d %s", c.Quantity, c.Name)
			}
			cr.pdf.SetXY(cr.x+marginX+5, cr.curY)
			cr.pdf.Text(line)
			cr.curY += lineHeight
		}
		cr.curY += 2
	}
}

// renderCard draws a single decklist card onto the PDF at the given offset.
func renderCard(p *gopdf.GoPdf, d deck.Deck, x, y float64) {
	cr := &cardRenderer{
		pdf:  p,
		x:    x,
		y:    y,
		curY: y + borderW + colorBarH + marginY,
	}
	cr.drawBorder()
	cr.drawColorBar(d.DominantColor)
	cr.drawTitle(d.Name)
	cr.drawGroups(d.Groups)
}
