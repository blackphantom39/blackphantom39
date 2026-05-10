// Package profile loads the static facts shown on the neofetch card from
// profile.json.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Profile is the static profile information rendered on the card.
type Profile struct {
	User       string   `json:"user"`
	Host       string   `json:"host"`
	Name       string   `json:"name"`
	Birthdate  string   `json:"birthdate"`
	Location   string   `json:"location"`
	Title      string   `json:"title"`
	OS         string   `json:"os"`
	Shell      string   `json:"shell"`
	Editor     string   `json:"editor"`
	WM         string   `json:"wm"`
	Terminal   string   `json:"terminal"`
	Languages  []string `json:"languages"`
	Frameworks []string `json:"frameworks"`
	Databases  []string `json:"databases"`
	Tools      []string `json:"tools"`
	Interests  []string `json:"interests"`
	GPGKey     string   `json:"gpgKey"`
	GPGUrl     string   `json:"gpgUrl"`
	GitHub     string   `json:"github"`
}

// Load reads and parses the profile JSON file at path.
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &p, nil
}

// Age returns the integer years between the profile's birthdate and now.
// Returns 0 if no birthdate is set.
func (p *Profile) Age(now time.Time) int {
	if p.Birthdate == "" {
		return 0
	}
	birth, err := time.Parse("2006-01-02", p.Birthdate)
	if err != nil {
		return 0
	}
	years := now.Year() - birth.Year()
	if now.YearDay() < birth.YearDay() {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}
