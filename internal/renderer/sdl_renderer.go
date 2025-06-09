package renderer

import (
	"fmt"
	"image/color"

	"github.com/veandco/go-sdl2/sdl"
)

type SDLRenderer struct {
	Renderer *sdl.Renderer
}

func (r SDLRenderer) Draw(display [32][64]bool, foreground, background color.RGBA) {

	fmt.Println("Called")
	err := r.Renderer.SetDrawColor(background.R, background.G, background.B, background.A) // Hintergrundfarbe (schwarz)

	if err != nil {
		fmt.Println(err)
	}

	err = r.Renderer.Clear()

	if err != nil {
		fmt.Println(err)
	}

	r.Renderer.SetDrawColor(foreground.R, foreground.G, foreground.B, foreground.A) // Pixel-Farbe (weiß)
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			if display[y][x] {
				rect := sdl.Rect{int32(x * 20), int32(y * 20), int32(20), int32(20)}
				err = r.Renderer.FillRect(&rect)

				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	r.Renderer.Present() // Fenster aktualisieren
}
