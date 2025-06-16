package renderer

import (
	"image/color"
)

// Renderer defines an abstraction for the graphics output of a CHIP-8 emulator.
// This interface decouples rendering logic from any specific graphics library,
// allowing flexible backends such as OpenGL, SDL, or headless testing environments.
//
// Each Renderer implementation is responsible for:
//   - Drawing the CHIP-8 display buffer using the configured foreground and background colors.
//   - Clearing the screen.
//
// The colorProfile (Profile) should be initialized at program startup and used by all Renderer implementations,
// ensuring consistent color handling across different rendering backends.
//
// The Draw method renders the provided CHIP-8 display buffer ([32][64]bool), where each boolean value
// represents the on/off state of a pixel
type Renderer interface {
	Draw(display [32][64]bool)
}

type colorProfile struct {
	Foreground color.RGBA
	Background color.RGBA
}

var Profiles = map[string]colorProfile{
	"black-white": {color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}},
	"night-sky":   {color.RGBA{255, 255, 204, 255}, color.RGBA{0, 0, 68, 255}},
	"console":     {color.RGBA{0, 0, 0, 255}, color.RGBA{34, 238, 34, 255}},
	"honey":       {color.RGBA{153, 102, 0, 255}, color.RGBA{255, 204, 0, 255}},
	"paper":       {color.RGBA{34, 34, 34, 255}, color.RGBA{255, 250, 240, 255}},
}

var Profile colorProfile
