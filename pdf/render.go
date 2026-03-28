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

	outerBorderW = 2.5  // thick outer frame
	innerBorderW = 0.5  // thin inner frame line
	innerInset   = 5.0  // distance from outer edge to inner frame
	colorBarH    = 16.0 // top color identity bar (holds deck title)
	marginX      = 8.0  // text left margin from inner frame
	marginY      = 10.0 // text top margin from color bar
	fontTitle    = 10.0
	fontHeader   = 8.0
	fontBody     = 7.5
	lineHeight   = 9.0
	indentX      = 5.0  // extra indent for card entries under headers
	groupSpacing = 2.0  // vertical space between type groups
)

// colorScheme defines the border/bar color and background tint for a color identity.
type colorScheme struct {
	border [3]uint8 // color bar and outer border
	bg     [3]uint8 // card background fill
}

// colorMap maps single-letter color identities to their visual scheme.
var colorMap = map[string]colorScheme{
	"W": {border: [3]uint8{170, 145, 80}, bg: [3]uint8{245, 240, 225}},  // White: darker gold border, warm cream bg
	"U": {border: [3]uint8{14, 104, 171}, bg: [3]uint8{215, 232, 245}},  // Blue: blue border, light blue bg
	"B": {border: [3]uint8{50, 40, 50}, bg: [3]uint8{225, 220, 225}},    // Black: near-black border, light gray-purple bg
	"R": {border: [3]uint8{211, 32, 41}, bg: [3]uint8{245, 225, 220}},   // Red: red border, light pink bg
	"G": {border: [3]uint8{0, 115, 62}, bg: [3]uint8{220, 238, 220}},    // Green: green border, light green bg
	"M": {border: [3]uint8{170, 145, 80}, bg: [3]uint8{245, 238, 220}},  // Multicolor: darker gold border, warm bg
	"C": {border: [3]uint8{158, 158, 158}, bg: [3]uint8{235, 235, 235}}, // Colorless: gray border, light gray bg
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
	pdf    *gopdf.GoPdf
	x, y   float64      // card origin (upper-left corner)
	curY   float64       // vertical cursor for text placement
	scheme colorScheme
}

// drawFrame draws the outer border, tinted background, and inner frame line.
func (cr *cardRenderer) drawFrame() {
	b := cr.scheme.border
	bg := cr.scheme.bg

	// Tinted background with color-matched border.
	cr.pdf.SetFillColor(bg[0], bg[1], bg[2])
	cr.pdf.SetStrokeColor(b[0], b[1], b[2])
	cr.pdf.SetLineWidth(outerBorderW)
	cr.pdf.Rectangle(cr.x, cr.y, cr.x+cardW, cr.y+cardH, "DF", 0, 0)

	// Inner frame line in a darker shade of the border color.
	cr.pdf.SetStrokeColor(b[0]/2, b[1]/2, b[2]/2)
	cr.pdf.SetLineWidth(innerBorderW)
	cr.pdf.Rectangle(cr.x+innerInset, cr.y+innerInset, cr.x+cardW-innerInset, cr.y+cardH-innerInset, "D", 0, 0)
}

// drawColorBar fills the top bar with the deck's border color, inside the inner frame.
func (cr *cardRenderer) drawColorBar() {
	b := cr.scheme.border
	cr.pdf.SetFillColor(b[0], b[1], b[2])
	barX := cr.x + innerInset + innerBorderW
	barY := cr.y + innerInset + innerBorderW
	barW := cardW - 2*(innerInset+innerBorderW)
	cr.pdf.RectFromUpperLeftWithStyle(barX, barY, barW, colorBarH, "F")
}

// drawTitle renders the deck name centered in white on top of the color bar.
func (cr *cardRenderer) drawTitle(name string) {
	cr.pdf.SetFont("body", "B", fontTitle)
	cr.pdf.SetTextColor(255, 255, 255)

	barY := cr.y + innerInset + innerBorderW
	nameW, _ := cr.pdf.MeasureTextWidth(name)
	nameX := cr.x + (cardW-nameW)/2
	// Text() uses baseline positioning. Offset down by ~75% of font size
	// to visually center within the color bar.
	nameY := barY + (colorBarH+fontTitle*0.5)/2
	cr.pdf.SetXY(nameX, nameY)
	cr.pdf.Text(name)

	// Reset text color for subsequent content.
	cr.pdf.SetTextColor(30, 30, 30)
}

// drawGroups renders all type groups with headers and indented card entries.
func (cr *cardRenderer) drawGroups(groups []deck.TypeGroup) {
	textLeft := cr.x + innerInset + marginX

	for _, g := range groups {
		header := fmt.Sprintf("%s:", pluralType(g.TypeName))
		cr.pdf.SetFont("body", "B", fontHeader)
		cr.pdf.SetTextColor(30, 30, 30)
		cr.pdf.SetXY(textLeft, cr.curY)
		cr.pdf.Text(header)
		cr.curY += lineHeight

		cr.pdf.SetFont("body", "", fontBody)
		cr.pdf.SetTextColor(50, 50, 50)
		for _, c := range g.Cards {
			var line string
			if basicLands[c.Name] {
				line = fmt.Sprintf("%s (%d)", c.Name, c.Quantity)
			} else {
				line = fmt.Sprintf("%d %s", c.Quantity, c.Name)
			}
			cr.pdf.SetXY(textLeft+indentX, cr.curY)
			cr.pdf.Text(line)
			cr.curY += lineHeight
		}
		cr.curY += groupSpacing
	}
}

// renderCard draws a single decklist card onto the PDF at the given offset.
func renderCard(p *gopdf.GoPdf, d deck.Deck, x, y float64) {
	scheme := colorMap["C"]
	if s, ok := colorMap[d.DominantColor]; ok {
		scheme = s
	}
	cr := &cardRenderer{
		pdf:    p,
		x:      x,
		y:      y,
		curY:   y + innerInset + innerBorderW + colorBarH + marginY,
		scheme: scheme,
	}
	cr.drawFrame()
	cr.drawColorBar()
	cr.drawTitle(d.Name)
	cr.drawGroups(d.Groups)
}
