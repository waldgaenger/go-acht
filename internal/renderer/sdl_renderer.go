package renderer

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

type SDLRenderer struct {
	Renderer *sdl.Renderer // TODO: Should not be exported
	Window   *sdl.Window   // TODO: Should not be exported
}

// Draw renders the CHIP-8 display buffer to the window.
//
// Each 'true' value in the display buffer ([32][64]bool) is drawn as a filled rectangle
// using the current foreground color; all other pixels use the background color.
// The display is cleared and redrawn on every call.
func (r SDLRenderer) Draw(display [32][64]bool) {
	// PERF: Could be optimized. No redraw neccessary
	r.Renderer.SetDrawColor(Profile.Background.R, Profile.Background.G, Profile.Background.B, Profile.Background.A)
	r.Renderer.Clear()
	r.Renderer.SetDrawColor(Profile.Foreground.R, Profile.Foreground.G, Profile.Foreground.B, Profile.Foreground.A)
	for y := range 32 {
		for x := range 64 {
			if display[y][x] {
				// TODO: Provide a scaling factor
				rect := sdl.Rect{X: int32(x * 10), Y: int32(y * 10), W: int32(10), H: int32(10)}
				r.Renderer.FillRect(&rect)
			}
		}
	}
	r.Renderer.Present()
}

// NewSDLRenderer initializes SDL, creates a window and renderer, and returns an SDLRenderer.
func NewSDLRenderer() (*SDLRenderer, error) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return nil, fmt.Errorf("failed to initialize SDL: %w", err)
	}

	window, err := sdl.CreateWindow(
		"CHIP8 EMULATOR",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		640, 320, sdl.WINDOW_SHOWN,
	)
	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("failed to create window: %w", err)
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		window.Destroy()
		sdl.Quit()
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	return &SDLRenderer{Renderer: renderer, Window: window}, nil
}

// Cleanup releases SDL resources.
func (r *SDLRenderer) Cleanup() {
	if r.Renderer != nil {
		r.Renderer.Destroy()
	}
	if r.Window != nil {
		r.Window.Destroy()
	}
	sdl.Quit()
}
