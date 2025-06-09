// Package renderer is intended to be used in order to decouple the specific graphics library
// from the Chip8 implementation.
package renderer

import "image/color"

type Renderer interface {
	Draw(display [32][64]bool, foreground, background color.RGBA)
}
