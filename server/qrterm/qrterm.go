// Package qrterm renders QR codes for terminals with half-block characters
// (two QR rows per text line), dark modules on a light background so phones
// can scan it from dark and light terminal themes alike.
package qrterm

import (
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Render returns the QR code for text as lines of block characters.
func Render(text string) (string, error) {
	q, err := qrcode.New(text, qrcode.Low)
	if err != nil {
		return "", err
	}
	q.DisableBorder = false
	bits := q.Bitmap() // true = dark module; includes the quiet zone
	var b strings.Builder
	for y := 0; y < len(bits); y += 2 {
		for x := range bits[y] {
			top := bits[y][x]
			bottom := y+1 < len(bits) && bits[y+1][x]
			// Light is the drawn "ink" (white blocks), dark is background.
			switch {
			case !top && !bottom:
				b.WriteRune('█')
			case !top && bottom:
				b.WriteRune('▀')
			case top && !bottom:
				b.WriteRune('▄')
			default:
				b.WriteRune(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}
