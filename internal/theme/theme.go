// Package theme loads named theme palettes shipped under internal/theme/themes.
//
// A theme bundles a coordinated dark + light palette pair. The active theme is
// chosen by writing its id (matching the filename without .json) into theme.txt
// at the repo root.
package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed themes/*.json
var themesFS embed.FS

// Palette holds the colours used to render one variant (dark or light) of the
// neofetch SVG card.
type Palette struct {
	Bg      string `json:"bg"`
	Fg      string `json:"fg"`
	Muted   string `json:"muted"`
	ASCII   string `json:"ascii"`
	Accent1 string `json:"accent1"`
	Accent2 string `json:"accent2"`
	Accent3 string `json:"accent3"`
}

// Theme is a coordinated dark + light pair.
type Theme struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Dark  Palette `json:"dark"`
	Light Palette `json:"light"`
}

// Load reads the theme with the given id from the embedded themes directory.
func Load(id string) (*Theme, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("theme id is empty")
	}
	data, err := fs.ReadFile(themesFS, "themes/"+id+".json")
	if err != nil {
		return nil, fmt.Errorf("loading theme %q: %w", id, err)
	}
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing theme %q: %w", id, err)
	}
	if t.ID != id {
		return nil, fmt.Errorf("theme id mismatch: file %q has id %q", id, t.ID)
	}
	return &t, nil
}

// List returns the ids of all bundled themes, sorted alphabetically.
func List() ([]string, error) {
	entries, err := themesFS.ReadDir("themes")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".json") {
			ids = append(ids, strings.TrimSuffix(n, ".json"))
		}
	}
	sort.Strings(ids)
	return ids, nil
}
