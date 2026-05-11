// Package readme renders the project README.md from a markdown template at
// templates/README.md.tmpl using text/template.
//
// The template receives the full profile, active theme and computed age so it
// can interpolate the dynamic bits (avatar alt, GPG key) without being tied
// to a fixed prose body — the prose lives in the template itself.
package readme

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/blackphantom39/blackphantom39/internal/profile"
	"github.com/blackphantom39/blackphantom39/internal/theme"
)

// Data is the value passed to the template.
type Data struct {
	Profile *profile.Profile
	Theme   *theme.Theme
	Age     int
}

// Render parses templatePath and writes the rendered markdown to outPath.
func Render(templatePath, outPath string, data Data) error {
	src, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", templatePath, err)
	}
	t, err := template.New("readme").Option("missingkey=error").Parse(string(src))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", templatePath, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("rendering %s: %w", templatePath, err)
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	return nil
}
