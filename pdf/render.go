// Package pdf renders organized decklists as printable PDF cards using gopdf.
// Each card displays the deck name, a color-identity bar, and card entries
// grouped by type with quantities.
package pdf

import (
	_ "embed"
	"fmt"

	"github.com/signintech/gopdf"

	"jumpforge/deck"
)

//go:embed fonts/DejaVuSans.ttf
var fontRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var fontBold []byte

// Card dimensions in PDF points (1 inch = 72 points).
const (
	cardW = 178.6 // 63mm (MTG card width)
	cardH = 249.4 // 88mm (MTG card height)

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

// setupFonts registers the embedded DejaVu Sans regular and bold fonts with the PDF.
func setupFonts(pdf *gopdf.GoPdf) error {
	if err := pdf.AddTTFFontData("body", fontRegular); err != nil {
		return fmt.Errorf("add regular font: %w", err)
	}
	if err := pdf.AddTTFFontDataWithOption("body", fontBold, gopdf.TtfOption{Style: gopdf.Bold}); err != nil {
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

// cardLine formats a card entry: singles as "Card Name", multiples as "Card Name (N)".
func cardLine(c deck.DeckCard) string {
	if c.Quantity == 1 {
		return c.Name
	}
	return fmt.Sprintf("%s (%d)", c.Name, c.Quantity)
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
	// Inset by half the stroke width so the border stays within card bounds.
	half := outerBorderW / 2
	cr.pdf.SetFillColor(bg[0], bg[1], bg[2])
	cr.pdf.SetStrokeColor(b[0], b[1], b[2])
	cr.pdf.SetLineWidth(outerBorderW)
	cr.pdf.Rectangle(cr.x+half, cr.y+half, cr.x+cardW-half, cr.y+cardH-half, "DF", 0, 0)

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
	nameX := cr.x + innerInset + marginX
	// Text() uses baseline positioning. Offset down by ~75% of font size
	// to visually center within the color bar.
	nameY := barY + (colorBarH+fontTitle*0.5)/2
	cr.pdf.SetXY(nameX, nameY)
	cr.pdf.Text(name)

	// Reset text color for subsequent content.
	cr.pdf.SetTextColor(30, 30, 30)
}

// formatColorIdentity formats a slice of color letters as Scryfall-style braces.
func formatColorIdentity(colors []string) string {
	var s string
	for _, c := range colors {
		s += "{" + c + "}"
	}
	return s
}

// drawColorIdentity renders the deck's color identity right-aligned in the color bar.
func (cr *cardRenderer) drawColorIdentity(colors []string) {
	text := formatColorIdentity(colors)
	cr.pdf.SetFont("body", "B", fontHeader)
	cr.pdf.SetTextColor(255, 255, 255)

	barY := cr.y + innerInset + innerBorderW
	textW, _ := cr.pdf.MeasureTextWidth(text)
	textX := cr.x + cardW - innerInset - marginX - textW
	textY := barY + (colorBarH+fontHeader*0.5)/2
	cr.pdf.SetXY(textX, textY)
	cr.pdf.Text(text)

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
			line := cardLine(c)
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
	cr.drawColorIdentity(d.ColorIdentity)
	cr.drawGroups(d.Groups)
}
