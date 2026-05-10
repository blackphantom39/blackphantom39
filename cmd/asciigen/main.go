// Command asciigen converts an image to a block-character ASCII art file.
//
// It is a one-shot helper run locally when the avatar changes; the resulting
// ascii.txt is committed and consumed by the SVG generator. Not part of CI.
//
// Usage:
//
//	go run ./cmd/asciigen -in _refs/avatar.png -out ascii.txt
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"strings"
)

// ramp is the density ramp (sparse → dense) used to paint the silhouette.
const ramp = " ░▒▓█"

func main() {
	var (
		in     = flag.String("in", "", "input image path (PNG or JPEG)")
		out    = flag.String("out", "ascii.txt", "output ASCII path")
		cols   = flag.Int("cols", 40, "output width in characters")
		rows   = flag.Int("rows", 0, "output height in characters (0 = auto from aspect)")
		mode   = flag.String("mode", "sketch", "density mode: silhouette | sketch | lit")
		alphaT = flag.Int("alpha-threshold", 32, "alpha below this value is treated as transparent (0–255)")
	)
	flag.Parse()

	if *in == "" {
		log.Fatal("missing -in")
	}
	if err := run(*in, *out, *cols, *rows, *mode, uint32(*alphaT)); err != nil {
		log.Fatal(err)
	}
}

func run(inPath, outPath string, cols, rows int, mode string, alphaT uint32) error {
	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", inPath, err)
	}

	bb := img.Bounds()
	srcW, srcH := bb.Dx(), bb.Dy()

	// Monospace cells are roughly 2× taller than wide; halve the row count
	// so the rendered image keeps its source aspect.
	if rows == 0 {
		rows = (cols * srcH) / (srcW * 2)
		if rows < 1 {
			rows = 1
		}
	}

	rampRunes := []rune(ramp)
	maxIdx := len(rampRunes) - 1

	density := densityFunc(mode)
	if density == nil {
		return fmt.Errorf("unknown mode %q (want: silhouette | sketch | lit)", mode)
	}

	var b strings.Builder
	for ry := 0; ry < rows; ry++ {
		y0 := bb.Min.Y + (ry*srcH)/rows
		y1 := bb.Min.Y + ((ry+1)*srcH)/rows
		for cx := 0; cx < cols; cx++ {
			x0 := bb.Min.X + (cx*srcW)/cols
			x1 := bb.Min.X + ((cx+1)*srcW)/cols

			var sumLum, sumAlpha, opaque, total uint64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, bl, a := img.At(x, y).RGBA()
					a8 := a >> 8
					total++
					if a8 < alphaT {
						continue
					}
					opaque++
					sumAlpha += uint64(a8)
					// Rec.601 luminance, 8-bit channels.
					lum := (uint64(r>>8)*299 + uint64(g>>8)*587 + uint64(bl>>8)*114) / 1000
					sumLum += lum
				}
			}

			if total == 0 || opaque == 0 {
				b.WriteRune(' ')
				continue
			}
			coverage := float64(opaque) / float64(total)
			avgLum := float64(sumLum) / float64(opaque) / 255.0 // 0..1
			avgAlpha := float64(sumAlpha) / float64(opaque) / 255.0

			d := density(coverage, avgLum, avgAlpha)
			if d < 0 {
				d = 0
			} else if d > 1 {
				d = 1
			}
			idx := int(d*float64(maxIdx) + 0.5)
			b.WriteRune(rampRunes[idx])
		}
		b.WriteByte('\n')
	}

	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

// densityFunc returns a function mapping (coverage, luminance, alpha) ∈ [0,1]
// to a ramp position ∈ [0,1]. Coverage is the fraction of opaque pixels in
// the cell; luminance is the average brightness of the opaque pixels.
func densityFunc(mode string) func(cov, lum, a float64) float64 {
	switch mode {
	case "silhouette":
		// Hard outline — coverage alone drives density.
		return func(cov, _, _ float64) float64 { return cov }
	case "sketch":
		// Negative-style: dark features (eyes, hair, outlines) are denser,
		// bright robe/skin areas are softer. Coverage gates the silhouette.
		return func(cov, lum, _ float64) float64 {
			return cov * (0.35 + 0.65*(1-lum))
		}
	case "lit":
		// Highlights are densest; dark areas fade.
		return func(cov, lum, _ float64) float64 {
			return cov * (0.35 + 0.65*lum)
		}
	default:
		return nil
	}
}
