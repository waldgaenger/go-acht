package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/waldgaenger/go-acht/internal/chip8"
	"github.com/waldgaenger/go-acht/internal/input"
	"github.com/waldgaenger/go-acht/internal/renderer"
)

var (
	flagRom          = flag.String("rom", "", "Set this flag to provide a path to a ROM file.")
	flagColorProfile = flag.String("colorprofile", "black-white", "Set this flag to provide a color hprofile.")
	flagScale        = flag.Int("scale", 20, "Set this flag to provide a screen scale factor.")
)

func main() {
	flag.Parse()

	if *flagRom == "" {
		fmt.Println("You have to provide a ROM file")
		flag.PrintDefaults()
		return
	}

	if *flagColorProfile != "" {
		profile, found := renderer.Profiles[*flagColorProfile]

		if !found {
			profile = renderer.Profiles["black-white"]
			fmt.Printf("no such color profile: %s - fallback: default profile black-white will be used \n", *flagColorProfile)
		}
		renderer.Profile = profile
	}

	r, err := renderer.NewSDLRenderer()

	if err != nil {
		fmt.Println("an error occurred while trying to create a new SDLRenderer: ", err)
		os.Exit(-1)
	}

	c8 := chip8.Chip8{Input: &input.SDLInput{}, Renderer: r}

	if err := c8.Run(*flagRom); err != nil {
		slog.Error("an error occurred while trying to run the emulator: " + err.Error())
		r.Cleanup()
		os.Exit(-1)
	}

	sdl.Quit()
}
