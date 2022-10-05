package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/waldgaenger/go-acht/src/chip8"
)

var (
	flagRom          = flag.String("rom", "", "Set this flag to provide a path to a ROM file.")
	flagScale        = flag.Int("scale", 20, "Set this flag to provide a screen scale factor.")
	flagColorProfile = flag.String("colorprofile", "black-white", "Set this flag to provide a color hprofile.")
)

func main() {
	flag.Parse()

	if *flagRom != "" {
		c8 := chip8.Chip8{}

		c8.Init(int32(*flagScale), *flagColorProfile)
		c8.LoadRom(*flagRom)

		lastCycle := time.Now()
		cylcleDelay := 16 * time.Millisecond

		for c8.Running() {
			currentTime := time.Now()
			if time.Since(lastCycle) > time.Duration(cylcleDelay) {
				lastCycle = currentTime
				c8.Cycle()
			}
		}

		c8.ShutDown()

	} else {
		fmt.Println("You have to provide a ROM file")
		flag.PrintDefaults()
	}
}
