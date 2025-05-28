package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/waldgaenger/go-acht/src/chip8"
)

// const banner = `
//  ▗▄▄▖ ▗▄▖      ▗▄▖  ▗▄▄▖▗▖ ▗▖▗▄▄▄▖
// ▐▌   ▐▌ ▐▌    ▐▌ ▐▌▐▌   ▐▌ ▐▌  █
// ▐▌▝▜▌▐▌ ▐▌    ▐▛▀▜▌▐▌   ▐▛▀▜▌  █
// ▝▚▄▞▘▝▚▄▞▘    ▐▌ ▐▌▝▚▄▄▖▐▌ ▐▌  █
// `

var (
	flagRom          = flag.String("rom", "", "Set this flag to provide a path to a ROM file.")
	flagScale        = flag.Int("scale", 20, "Set this flag to provide a screen scale factor.")
	flagColorProfile = flag.String("colorprofile", "black-white", "Set this flag to provide a color hprofile.")
	// TODO: Implementing a debugging mode where we can constantly dump the registers to TTY
	// TODO: Visualization of the internal state on the right side of the window
)

func main() {
	flag.Parse()

	if *flagRom == "" {
		fmt.Println("You have to provide a ROM file")
		flag.PrintDefaults()
		return
	}

	c8 := chip8.Chip8{}

	if err := c8.Run(*flagRom, int32(*flagScale), *flagColorProfile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
