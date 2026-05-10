// Command blackphantom39 generates the GitHub profile README and the dark/light
// neofetch-style SVG cards. It is run by the daily GitHub Actions workflow as
// well as locally via `go run .`.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/blackphantom39/blackphantom39/internal/profile"
	"github.com/blackphantom39/blackphantom39/internal/svg"
	"github.com/blackphantom39/blackphantom39/internal/theme"
)

func main() {
	var (
		themeFile   = flag.String("theme", "theme.txt", "path to active theme file")
		profileFile = flag.String("profile", "profile.json", "path to profile JSON")
		asciiFile   = flag.String("ascii", "ascii.txt", "path to ASCII art file")
		outDark     = flag.String("dark", "dark.svg", "output path for the dark SVG")
		outLight    = flag.String("light", "light.svg", "output path for the light SVG")
	)
	flag.Parse()

	if err := run(*themeFile, *profileFile, *asciiFile, *outDark, *outLight); err != nil {
		log.Fatal(err)
	}
}

func run(themeFile, profileFile, asciiFile, outDark, outLight string) error {
	themeID, err := readActiveTheme(themeFile)
	if err != nil {
		return err
	}
	th, err := theme.Load(themeID)
	if err != nil {
		return err
	}
	pr, err := profile.Load(profileFile)
	if err != nil {
		return err
	}
	asciiLines, err := readASCII(asciiFile)
	if err != nil {
		return err
	}

	card := &svg.Card{
		Profile: pr,
		Theme:   th,
		ASCII:   asciiLines,
		Age:     pr.Age(time.Now()),
	}

	if err := os.WriteFile(outDark, card.RenderDark(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outDark, err)
	}
	if err := os.WriteFile(outLight, card.RenderLight(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outLight, err)
	}
	fmt.Printf("rendered %s + %s with theme %q\n", outDark, outLight, th.Name)
	return nil
}

func readActiveTheme(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", fmt.Errorf("%s is empty: expected a theme id", path)
	}
	return id, nil
}

func readASCII(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}
