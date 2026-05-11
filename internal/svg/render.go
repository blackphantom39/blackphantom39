// Package svg renders the neofetch-style profile card as an SVG document.
//
// The card has a fixed width and grows vertically with row count. ASCII art
// fills the left column; key/value rows fill the right column. Two variants
// are emitted per render (dark + light) using the active theme's palette pair.
package svg

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/blackphantom39/blackphantom39/internal/profile"
	"github.com/blackphantom39/blackphantom39/internal/theme"
)

const (
	width         = 880
	padding       = 24
	lineH         = 20
	asciiLineH    = 14
	fontSize      = 14
	asciiFontSize = 11
	asciiX        = 24
	asciiY0       = 30
	infoX         = 440
	infoY0        = 36
	keyWidth      = 12 // characters reserved for key column
)

// Card bundles everything needed to render one profile SVG.
type Card struct {
	Profile *profile.Profile
	Theme   *theme.Theme
	ASCII   []string
	Age     int
}

// RenderDark returns the SVG bytes for the dark variant.
func (c *Card) RenderDark() []byte { return c.render(c.Theme.Dark) }

// RenderLight returns the SVG bytes for the light variant.
func (c *Card) RenderLight() []byte { return c.render(c.Theme.Light) }

type row struct {
	key   string
	value string
	blank bool
}

func (c *Card) rows() []row {
	p := c.Profile
	return []row{
		{key: "Name", value: p.Name},
		{key: "Age", value: fmt.Sprintf("%d years", c.Age)},
		{key: "Title", value: p.Title},
		{key: "Location", value: p.Location},
		{blank: true},
		{key: "OS", value: p.OS},
		{key: "WM", value: p.WM},
		{key: "Shell", value: p.Shell},
		{key: "Terminal", value: p.Terminal},
		{key: "Editor", value: p.Editor},
		{blank: true},
		{key: "Languages", value: strings.Join(p.Languages, ", ")},
		{key: "Frameworks", value: strings.Join(p.Frameworks, ", ")},
		{key: "Databases", value: strings.Join(p.Databases, ", ")},
		{key: "Tools", value: strings.Join(p.Tools, ", ")},
		{blank: true},
		{key: "Interests", value: strings.Join(p.Interests, ", ")},
		{key: "GitHub", value: p.GitHub},
		{key: "GPG", value: p.GPGKey},
	}
}

func (c *Card) render(p theme.Palette) []byte {
	rows := c.rows()

	// 2 header lines + N row lines, then bottom padding.
	infoH := infoY0 + (2+len(rows))*lineH + padding
	asciiH := asciiY0 + len(c.ASCII)*asciiLineH + padding
	height := infoH
	if asciiH > height {
		height = asciiH
	}

	var b strings.Builder
	fmt.Fprintln(&b, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" font-size="%d">`+"\n",
		width, height, width, height, fontSize)

	fmt.Fprintf(&b, `<style>
text { font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Consolas, monospace; }
.bg { fill: %s }
.fg { fill: %s }
.mt { fill: %s }
.as { fill: %s }
.a1 { fill: %s }
.a2 { fill: %s }
.a3 { fill: %s }
.b  { font-weight: 700 }
</style>
`, p.Bg, p.Fg, p.Muted, p.ASCII, p.Accent1, p.Accent2, p.Accent3)

	fmt.Fprintf(&b, `<rect class="bg" width="%d" height="%d" rx="10" />`+"\n", width, height)

	// ASCII column. Rendered at a smaller font size than the info column so
	// a denser ASCII grid (e.g. 60 cols × 30 rows) still fits on the left.
	if len(c.ASCII) > 0 {
		fmt.Fprintf(&b, `<g class="as" font-size="%d">`+"\n", asciiFontSize)
		for i, line := range c.ASCII {
			y := asciiY0 + i*asciiLineH
			fmt.Fprintf(&b, `  <text x="%d" y="%d" xml:space="preserve">%s</text>`+"\n",
				asciiX, y, xmlEscape(line))
		}
		fmt.Fprintln(&b, `</g>`)
	}

	// Header line: user@host
	y := infoY0
	fmt.Fprintf(&b,
		`<text x="%d" y="%d"><tspan class="a1 b">%s</tspan><tspan class="fg">@</tspan><tspan class="a1 b">%s</tspan></text>`+"\n",
		infoX, y, xmlEscape(c.Profile.User), xmlEscape(c.Profile.Host))

	// Separator line, length matches user@host
	y += lineH
	sepLen := utf8.RuneCountInString(c.Profile.User) + 1 + utf8.RuneCountInString(c.Profile.Host)
	fmt.Fprintf(&b, `<text x="%d" y="%d" class="mt">%s</text>`+"\n",
		infoX, y, strings.Repeat("─", sepLen))

	// Rows
	for _, r := range rows {
		y += lineH
		if r.blank {
			continue
		}
		dots := keyWidth - utf8.RuneCountInString(r.key)
		if dots < 1 {
			dots = 1
		}
		sep := strings.Repeat(".", dots) + ":"
		fmt.Fprintf(&b,
			`<text x="%d" y="%d"><tspan class="a2 b">%s</tspan><tspan class="mt" xml:space="preserve">%s </tspan><tspan class="fg">%s</tspan></text>`+"\n",
			infoX, y, xmlEscape(r.key), sep, xmlEscape(r.value))
	}

	fmt.Fprintln(&b, `</svg>`)
	return []byte(b.String())
}

// xmlEscape escapes the five XML-significant characters.
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}
