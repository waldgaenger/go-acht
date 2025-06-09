package input

import "github.com/veandco/go-sdl2/sdl"

var keyMap = map[sdl.Keycode]uint8{
	sdl.K_1: 0x1, sdl.K_2: 0x2, sdl.K_3: 0x3, sdl.K_4: 0xC,
	sdl.K_q: 0x4, sdl.K_w: 0x5, sdl.K_e: 0x6, sdl.K_r: 0xD,
	sdl.K_a: 0x7, sdl.K_s: 0x8, sdl.K_d: 0x9, sdl.K_f: 0xE,
	sdl.K_z: 0xA, sdl.K_x: 0x0, sdl.K_c: 0xB, sdl.K_v: 0xF,
}

type SDLInput struct{}

func (s *SDLInput) PollKeys() (keyPad [16]uint8, quit bool) {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			quit = true
		case *sdl.KeyboardEvent:
			if idx, ok := keyMap[e.Keysym.Sym]; ok {
				switch e.Type {
				case sdl.KEYDOWN:
					keyPad[idx] = 1
				case sdl.KEYUP:
					keyPad[idx] = 0
				}
			}
		}
	}
	return
}
