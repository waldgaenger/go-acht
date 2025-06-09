package main

import (
	"flag"
	"fmt"
	"log/slog"
	_ "net/http/pprof"
	"os"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/waldgaenger/go-acht/src/chip8"
	"github.com/waldgaenger/go-acht/src/input"
	"github.com/waldgaenger/go-acht/src/renderer"
)

// const banner = `
//  ▗▄▄▖ ▗▄▖      ▗▄▖  ▗▄▄▖▗▖ ▗▖▗▄▄▄▖
// ▐▌   ▐▌ ▐▌    ▐▌ ▐▌▐▌   ▐▌ ▐▌  █
// ▐▌▝▜▌▐▌ ▐▌    ▐▛▀▜▌▐▌   ▐▛▀▜▌  █
// ▝▚▄▞▘▝▚▄▞▘    ▐▌ ▐▌▝▚▄▄▖▐▌ ▐▌  █
// `

var (
	flagRom          = flag.String("rom", "", "Set this flag to provide a path to a ROM file.")
	flagColorProfile = flag.String("colorprofile", "black-white", "Set this flag to provide a color hprofile.")
	flagScale        = flag.Int("scale", 20, "Set this flag to provide a screen scale factor.")
	// TODO: Implementing a debugging mode where we can constantly dump the registers to TTY
	// TODO: Visualization of the internal state on the right side of the window
)

func main() {
	flag.Parse()
	// window, renderer, err := sdl.CreateWindowAndRenderer(640, 320, sdl.WINDOW_SHOWN)
	// renderer.DrawRect()

	// renderer.DrawRect()
	if *flagRom == "" {
		fmt.Println("You have to provide a ROM file")
		flag.PrintDefaults()
		return
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		slog.Error("an error occurred while trying to initialize the SDL library")
		os.Exit(-1)
	}

	window, err := sdl.CreateWindow("CHIP8 EMULATOR", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		64*20, 32*20, sdl.WINDOW_SHOWN)

	if err != nil {
		slog.Error("an error occurred while trying to initialize the SDL library: " + err.Error())
		sdl.Quit()
		os.Exit(-1)
	}
	r, err := sdl.CreateRenderer(window, -1, 0)

	if err != nil {
		slog.Error("an error occurred while trying to obtain a renderer" + err.Error())
		sdl.Quit()
		os.Exit(-1)
	}

	c8 := chip8.Chip8{
		Input:    &input.SDLInput{},
		Renderer: &renderer.SDLRenderer{Renderer: r},
	}

	if err := c8.Run(*flagRom, int32(*flagScale), *flagColorProfile); err != nil {
		slog.Error("an error occurred while trying to run the emulator: " + err.Error())
		sdl.Quit()
		os.Exit(-1)
	}

	fmt.Println("End!")

	sdl.Quit()
}
