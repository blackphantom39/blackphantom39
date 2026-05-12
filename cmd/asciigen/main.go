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

// defaultRamp is the sparse → dense ramp used by silhouette/sketch/lit when
// the caller does not override it via -ramp.
const defaultRamp = " ░▒▓█"

func main() {
	var (
		in          = flag.String("in", "", "input image path (PNG or JPEG; convert WebP via sips first)")
		out         = flag.String("out", "ascii.txt", "output ASCII path")
		cols        = flag.Int("cols", 40, "output width in characters")
		rows        = flag.Int("rows", 0, "output height in characters (0 = auto from aspect)")
		mode        = flag.String("mode", "sketch", "density mode: silhouette | sketch | lit | quadrant | braille")
		alphaT      = flag.Int("alpha-threshold", 32, "alpha below this value is treated as transparent (0–255)")
		lumT        = flag.Float64("lum-threshold", 0.5, "quadrant/braille: luminance threshold within opaque area (0–1)")
		ramp        = flag.String("ramp", defaultRamp, "density ramp (sparse → dense) for silhouette/sketch/lit")
		rampReverse = flag.Bool("ramp-reverse", false, "treat -ramp as dense → sparse and reverse it")
		colorsOut   = flag.String("colors", "", "if set, also write a per-character hex-color sidecar; braille mode then uses alpha-only so light regions are included")
	)
	flag.Parse()

	if *in == "" {
		log.Fatal("missing -in")
	}
	rampRunes := []rune(*ramp)
	if *rampReverse {
		for i, j := 0, len(rampRunes)-1; i < j; i, j = i+1, j-1 {
			rampRunes[i], rampRunes[j] = rampRunes[j], rampRunes[i]
		}
	}
	if len(rampRunes) < 2 {
		log.Fatalf("-ramp must have at least 2 runes (got %q)", *ramp)
	}

	switch *mode {
	case "quadrant":
		if err := runQuadrant(*in, *out, *cols, *rows, uint32(*alphaT), *lumT); err != nil {
			log.Fatal(err)
		}
	case "braille":
		if err := runBraille(*in, *out, *colorsOut, *cols, *rows, uint32(*alphaT), *lumT); err != nil {
			log.Fatal(err)
		}
	default:
		if err := run(*in, *out, *cols, *rows, *mode, uint32(*alphaT), rampRunes); err != nil {
			log.Fatal(err)
		}
	}
}

func run(inPath, outPath string, cols, rows int, mode string, alphaT uint32, ramp []rune) error {
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

	maxIdx := len(ramp) - 1

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
			b.WriteRune(ramp[idx])
		}
		b.WriteByte('\n')
	}

	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

// quadrantChars maps a 4-bit sub-pixel mask to a Unicode block quadrant rune.
// Bit layout: bit 0 = upper-left, bit 1 = upper-right, bit 2 = lower-left,
// bit 3 = lower-right.
var quadrantChars = []rune{
	' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛',
	'▗', '▚', '▐', '▜', '▄', '▙', '▟', '█',
}

// runQuadrant renders the image using 2×2 sub-pixel quadrant block characters,
// quadrupling spatial resolution at the same column count. Each character cell
// is divided into four sub-cells; a sub-cell is "on" when it is sufficiently
// opaque AND its luminance is below lumT (so dark features like eyes, hair
// and outlines drive the shape).
func runQuadrant(inPath, outPath string, cols, rows int, alphaT uint32, lumT float64) error {
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

	// Quadrant sub-pixels are roughly square inside a monospace cell
	// (~4×8 px each), so source aspect needs no halving here.
	if rows == 0 {
		rows = cols * srcH / srcW
		if rows < 1 {
			rows = 1
		}
	}

	subW, subH := cols*2, rows*2

	// Sub-pixel mask: true when the sub-cell is "on".
	mask := make([]bool, subW*subH)
	for sy := 0; sy < subH; sy++ {
		y0 := bb.Min.Y + (sy*srcH)/subH
		y1 := bb.Min.Y + ((sy+1)*srcH)/subH
		for sx := 0; sx < subW; sx++ {
			x0 := bb.Min.X + (sx*srcW)/subW
			x1 := bb.Min.X + ((sx+1)*srcW)/subW

			var sumLum, opaque, total uint64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, bl, a := img.At(x, y).RGBA()
					a8 := a >> 8
					total++
					if a8 < alphaT {
						continue
					}
					opaque++
					sumLum += (uint64(r>>8)*299 + uint64(g>>8)*587 + uint64(bl>>8)*114) / 1000
				}
			}
			if total == 0 || opaque == 0 {
				continue
			}
			coverage := float64(opaque) / float64(total)
			avgLum := float64(sumLum) / float64(opaque) / 255.0
			if coverage >= 0.5 && avgLum < lumT {
				mask[sy*subW+sx] = true
			}
		}
	}

	var b strings.Builder
	for ry := 0; ry < rows; ry++ {
		for cx := 0; cx < cols; cx++ {
			ul := mask[(ry*2+0)*subW+(cx*2+0)]
			ur := mask[(ry*2+0)*subW+(cx*2+1)]
			ll := mask[(ry*2+1)*subW+(cx*2+0)]
			lr := mask[(ry*2+1)*subW+(cx*2+1)]
			var idx int
			if ul {
				idx |= 1
			}
			if ur {
				idx |= 2
			}
			if ll {
				idx |= 4
			}
			if lr {
				idx |= 8
			}
			b.WriteRune(quadrantChars[idx])
		}
		b.WriteByte('\n')
	}

	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

// brailleBit maps a sub-dot at (col, row) inside the 2×4 grid to its Unicode
// Braille bit position relative to U+2800. Order follows the Unicode standard:
//
//	(0,0)=dot1 bit0   (1,0)=dot4 bit3
//	(0,1)=dot2 bit1   (1,1)=dot5 bit4
//	(0,2)=dot3 bit2   (1,2)=dot6 bit5
//	(0,3)=dot7 bit6   (1,3)=dot8 bit7
var brailleBit = [4][2]int{
	{0, 3},
	{1, 4},
	{2, 5},
	{6, 7},
}

// runBraille renders the image using Braille patterns (U+2800–U+28FF). Each
// character cell encodes 2×4 binary sub-dots, giving 8× spatial resolution at
// the same column count and the squarest sub-pixel aspect of any block mode.
//
// Default sub-dot inclusion: opaque AND luminance < lumT (dark features like
// eyes/outlines drive the dot mask).
//
// When colorsOut is non-empty, the mode is switched to alpha-only inclusion
// (any sufficiently opaque sub-dot counts) so that light regions — towel
// arms, skin, highlights — are also part of the silhouette. A parallel hex
// color grid is written to colorsOut, one space-separated token per char.
// Cells with no opaque dots get the sentinel "-" so the renderer can fall
// back to the theme accent.
func runBraille(inPath, outPath, colorsOut string, cols, rows int, alphaT uint32, lumT float64) error {
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

	// Braille sub-dots are roughly square inside a monospace cell
	// (~4×4 px each), so source aspect needs no halving here.
	if rows == 0 {
		rows = cols * srcH / srcW
		if rows < 1 {
			rows = 1
		}
	}

	subW, subH := cols*2, rows*4
	wantColors := colorsOut != ""

	mask := make([]bool, subW*subH)
	// Per-sub-dot color sums, populated only when -colors is requested.
	var rSum, gSum, bSum []uint64
	var nPx []uint64
	if wantColors {
		rSum = make([]uint64, subW*subH)
		gSum = make([]uint64, subW*subH)
		bSum = make([]uint64, subW*subH)
		nPx = make([]uint64, subW*subH)
	}

	for sy := 0; sy < subH; sy++ {
		y0 := bb.Min.Y + (sy*srcH)/subH
		y1 := bb.Min.Y + ((sy+1)*srcH)/subH
		for sx := 0; sx < subW; sx++ {
			x0 := bb.Min.X + (sx*srcW)/subW
			x1 := bb.Min.X + ((sx+1)*srcW)/subW

			var sumLum, opaque, total, sR, sG, sB uint64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, bl, a := img.At(x, y).RGBA()
					a8 := a >> 8
					total++
					if a8 < alphaT {
						continue
					}
					opaque++
					r8, g8, b8 := uint64(r>>8), uint64(g>>8), uint64(bl>>8)
					sumLum += (r8*299 + g8*587 + b8*114) / 1000
					sR += r8
					sG += g8
					sB += b8
				}
			}
			if total == 0 || opaque == 0 {
				continue
			}
			coverage := float64(opaque) / float64(total)
			avgLum := float64(sumLum) / float64(opaque) / 255.0

			on := coverage >= 0.5
			if !wantColors {
				on = on && avgLum < lumT
			}
			if on {
				idx := sy*subW + sx
				mask[idx] = true
				if wantColors {
					rSum[idx] = sR
					gSum[idx] = sG
					bSum[idx] = sB
					nPx[idx] = opaque
				}
			}
		}
	}

	var chars strings.Builder
	var cols2 strings.Builder
	for ry := 0; ry < rows; ry++ {
		for cx := 0; cx < cols; cx++ {
			var bits int
			var cR, cG, cB, cN uint64
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					idx := (ry*4+dy)*subW + (cx*2 + dx)
					if !mask[idx] {
						continue
					}
					bits |= 1 << brailleBit[dy][dx]
					if wantColors {
						cR += rSum[idx]
						cG += gSum[idx]
						cB += bSum[idx]
						cN += nPx[idx]
					}
				}
			}
			chars.WriteRune(rune(0x2800 + bits))
			if wantColors {
				if cx > 0 {
					cols2.WriteByte(' ')
				}
				if cN == 0 {
					cols2.WriteByte('-')
				} else {
					fmt.Fprintf(&cols2, "#%02x%02x%02x", cR/cN, cG/cN, cB/cN)
				}
			}
		}
		chars.WriteByte('\n')
		if wantColors {
			cols2.WriteByte('\n')
		}
	}

	if err := os.WriteFile(outPath, []byte(chars.String()), 0o644); err != nil {
		return err
	}
	if wantColors {
		if err := os.WriteFile(colorsOut, []byte(cols2.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
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
