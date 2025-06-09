package chip8

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

const startAddress int = 0x200
const fontStartAddress int = 0x50
const fontSize int = 80

var fontSet = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type colorProfile struct {
	background uint32
	foreground uint32
}

var profiles = map[string]colorProfile{
	"black-white": {0x000000, 0xFFFFFF},
	"night-sky":   {0x000044, 0xFFFFCC},
	"console":     {0x000000, 0x22EE22},
	"honey":       {0x996600, 0xFFCC00},
}

var keyMap = map[sdl.Keycode]uint8{
	sdl.K_1: 0x1, sdl.K_2: 0x2, sdl.K_3: 0x3, sdl.K_4: 0xC,
	sdl.K_q: 0x4, sdl.K_w: 0x5, sdl.K_e: 0x6, sdl.K_r: 0xD,
	sdl.K_a: 0x7, sdl.K_s: 0x8, sdl.K_d: 0x9, sdl.K_f: 0xE,
	sdl.K_z: 0xA, sdl.K_x: 0x0, sdl.K_c: 0xB, sdl.K_v: 0xF,
}

type opcodeHandler func(*Chip8)

// The dispatch table points to all the supported operations of the Chip8-Emulator.
var dispatchTable = map[uint16]opcodeHandler{
	0x00E0: (*Chip8).op00E0,
	0x00EE: (*Chip8).op00EE,
	0x1000: (*Chip8).op1NNN,
	0x2000: (*Chip8).op2NNN,
	0x3000: (*Chip8).op3XKK,
	0x4000: (*Chip8).op4XKK,
	0x5000: (*Chip8).op5XY0,
	0x6000: (*Chip8).op6XKK,
	0x7000: (*Chip8).op7XKK,
	0x8000: (*Chip8).op8XY0,
	0x8001: (*Chip8).op8XY1,
	0x8002: (*Chip8).op8XY2,
	0x8003: (*Chip8).op8XY3,
	0x8004: (*Chip8).op8XY4,
	0x8005: (*Chip8).op8XY5,
	0x8006: (*Chip8).op8XY6,
	0x8007: (*Chip8).op8XY7,
	0x800E: (*Chip8).op8XYE,
	0x9000: (*Chip8).op9XY0,
	0xA000: (*Chip8).opANNN,
	0xB000: (*Chip8).opBNNN,
	0xC000: (*Chip8).opCXKK,
	0xD000: (*Chip8).opDXYN,
	0xE09E: (*Chip8).opEX9E,
	0xE0A1: (*Chip8).opEXA1,
	0xF007: (*Chip8).opFX07,
	0xF00A: (*Chip8).opFX0A,
	0xF015: (*Chip8).opFX15,
	0xF018: (*Chip8).opFX18,
	0xF01E: (*Chip8).opFX1E,
	0xF029: (*Chip8).opFX29,
	0xF033: (*Chip8).opFX33,
	0xF055: (*Chip8).opFX55,
	0xF065: (*Chip8).opFX65,
}

type Chip8 struct {
	registers           [16]uint8 // All 16 registers of the emulator
	memory              [4096]uint8
	programCounter      uint16
	indexRegister       uint16
	callStack           [16]uint16
	stackPointer        uint8
	opcode              uint16
	keyPad              [16]uint8
	delayTimer          uint8         // The delay timer is decremented at a rate of 60 Hz according to the specification.
	soundTimer          uint8         // The sound timer is decremented at a rate of 60 Hz according to the specification.
	display             [32][64]uint8 // 64x32 monochrome display
	sdlWindow           *sdl.Window   // Stores the corresponding SDL main window
	scaleFactor         int32
	running             bool // Indicates whether the emulator is running
	colorCodeForeground uint32
	colorCodeBackground uint32
}

// Runs the Chip8 emulator with the given ROM and configuration.
// Note that Run never returns unless there is an error.
func (c8 *Chip8) Run(romPath string, scaleFactor int32, colorProfile string) error {

	if err := c8.loadRom(romPath); err != nil {
		return fmt.Errorf("failed to load ROM: %w", err)
	}
	// Fail early and try to load the rom first.
	c8.Init(scaleFactor, colorProfile)

	clock := time.NewTicker(time.Millisecond)
	video := time.NewTicker(time.Second / 60)
	sound := time.NewTicker(time.Second / 60)
	delay := time.NewTicker(time.Second / 60)

	for c8.Running() {
		select {
		case <-sound.C:
			if c8.soundTimer > 0 {
				c8.soundTimer--
				c8.updateSound()
			}
		case <-delay.C:
			if c8.delayTimer > 0 {
				c8.delayTimer--
			}
		case <-video.C:
			c8.draw()
		case <-clock.C:
			c8.cycle()
		}
	}

	return nil

}

// Initializes the values of the Chip8 structure.
// Sets the program counter to the start address.
func (c8 *Chip8) Init(scaleFactor int32, colorProfile string) {
	c8.programCounter = uint16(startAddress)
	c8.scaleFactor = scaleFactor
	// Loading the set of fonts into the specified memory area
	for i := 0; i < fontSize; i++ {
		c8.memory[fontStartAddress+i] = fontSet[i]
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Printf("An error occurred while trying to initialize the SDL components: %v", err)
		sdl.Quit()
		os.Exit(-1)
	}

	c8.SetColorProfile(colorProfile)

	window, err := sdl.CreateWindow("CHIP8 EMULATOR", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		64*c8.scaleFactor, 32*c8.scaleFactor, sdl.WINDOW_SHOWN)

	if err != nil {
		fmt.Printf("An error occurred while trying to create the SDL window: %v", err)
		sdl.Quit()
		os.Exit(-1)
	}
	c8.sdlWindow = window
	c8.running = true
}

func (c8 *Chip8) Running() bool {
	return c8.running
}

// Sets the color profile specified by the user.
func (c8 *Chip8) SetColorProfile(profileName string) {
	if profile, ok := profiles[profileName]; ok {
		c8.colorCodeBackground = profile.background
		c8.colorCodeForeground = profile.foreground
	} else {
		// Default profile which is black-white
		c8.colorCodeBackground = 0x000000 // black
		c8.colorCodeForeground = 0xFFFFFF // white
	}
}

var AudioDevice sdl.AudioDeviceID

// ObtainedSpec is the spec opened for the device.
var ObtainedSpec *sdl.AudioSpec

func initAudio() {
	var err error

	// the desired audio specification
	desiredSpec := &sdl.AudioSpec{
		Freq:     64 * 60,
		Format:   sdl.AUDIO_F32LSB,
		Channels: 1,
		Samples:  64,
	}

	ObtainedSpec = &sdl.AudioSpec{}

	// open the device and start playing it
	if sdl.GetNumAudioDevices(false) > 0 {
		if AudioDevice, err = sdl.OpenAudioDevice("", false, desiredSpec, ObtainedSpec, sdl.AUDIO_ALLOW_ANY_CHANGE); err != nil {
			panic(err)
		}

		sdl.PauseAudioDevice(AudioDevice, false)
	}

	fmt.Println(AudioDevice)
	fmt.Println(ObtainedSpec)
}

func (c8 *Chip8) updateSound() {
	if AudioDevice != 0 {
		sample := make([]byte, 4)

		binary.LittleEndian.PutUint32(sample, math.Float32bits(1.0))

		// N channels, each channel has S samples (4 bytes each)
		n := int(ObtainedSpec.Channels) * int(ObtainedSpec.Samples) * 4
		data := make([]byte, n)

		// 128 samples per 1/60 of a second
		for i := 0; i < n; i += 4 {
			copy(data[i:], sample)
		}

		if err := sdl.QueueAudio(AudioDevice, data); err != nil {
			println(err)
		}
	}
}

func (c8 *Chip8) draw() {
	surface, err := c8.sdlWindow.GetSurface()
	if err != nil {
		// TODO: Should not panic? Can we recover from the error or not?
		panic(err)
	}

	bg := c8.colorCodeBackground
	fg := c8.colorCodeForeground
	scale := c8.scaleFactor

	for row, rowVals := range c8.display {
		y := int32(row) * scale
		for col, pixel := range rowVals {
			x := int32(col) * scale
			color := bg
			if pixel != 0 {
				color = fg
			}
			rect := sdl.Rect{X: x, Y: y, W: scale, H: scale}
			surface.FillRect(&rect, color)
		}
	}
	c8.sdlWindow.UpdateSurface()
}

func (c8 *Chip8) keyHandler() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			c8.running = false
		case *sdl.KeyboardEvent:
			if idx, ok := keyMap[e.Keysym.Sym]; ok {
				switch e.Type {
				case sdl.KEYDOWN:
					c8.keyPad[idx] = 1
				case sdl.KEYUP:
					c8.keyPad[idx] = 0
				}
			}
		}
	}
}

// cycle carries out one full CPU cycle: fetches the next opcode, decodes it using the dispatch table, and executes the matching instruction handler.
func (c8 *Chip8) cycle() {
	c8.fetch()
	c8.keyHandler()

	if handler := dispatchTable[c8.decodeOpcode()]; handler != nil {
		handler(c8)
	} else {
		fmt.Printf("Invalid opcode: %#X\n", c8.opcode)
	}

}

func (c8 *Chip8) fetch() {
	hi := uint16(c8.memory[c8.programCounter])
	lo := uint16(c8.memory[c8.programCounter+1])
	c8.opcode = (hi << 8) | lo
}

func (c8 *Chip8) ShutDown() {
	c8.running = false
	c8.sdlWindow.Destroy()
	sdl.Quit()
}

func (c8 *Chip8) loadRom(pathToRom string) error {
	f, err := os.Open(pathToRom)

	if err != nil {
		f.Close()
		return err
	}

	defer f.Close()

	bytesRead, err := f.Read(c8.memory[startAddress:])

	// TODO: Check the number of bytes read and tell the user when the ROM is too big? Etc.

	if err != nil {
		f.Close()
		return fmt.Errorf("an error occurred while trying to read the ROM into memory: %w", err)
	}

	fmt.Println("[+] ROM successfully read into the memory.")
	fmt.Println("[+] ROM size: ", bytesRead)

	return nil
}

// decodeOpcode decodes the the current opcode and returns the value.
// Intended to be used for the dispatch map to find the corresponding functions which realizes the instruction.
func (c8 *Chip8) decodeOpcode() uint16 {
	switch c8.opcode & 0xF000 {
	case 0x0000:
		return c8.opcode & 0x00FF // 00E0, 00EE
	case 0x8000:
		return c8.opcode & 0xF00F // z.B. 8XY0
	case 0xE000, 0xF000:
		return c8.opcode & 0xF0FF // z.B. EX9E, FX07
	default:
		return c8.opcode & 0xF000 // z.B. 1000, 2000, etc.
	}
}

// Clears the display by resetting all pixels to 0.
func (c8 *Chip8) op00E0() {
	c8.display = [32][64]uint8{} // TODO: Should be a bool.
}

// Returns from a subroutine.
// Decrements the stack pointer by one and sets the program counter to the address of the call stack on index of the stack pointer.
func (c8 *Chip8) op00EE() {
	c8.stackPointer -= 1
	c8.programCounter = c8.callStack[c8.stackPointer]
}

// Jumps to location NNN.
// Internally the program counter will be set to the address NNN.
func (c8 *Chip8) op1NNN() {
	var address uint16 = c8.opcode & 0x0FFF

	c8.programCounter = address
}

// Calls the subroutine at address NNN.
func (c8 *Chip8) op2NNN() {
	var address uint16 = c8.opcode & 0x0FFF

	c8.callStack[c8.stackPointer] = c8.programCounter

	c8.stackPointer += 1

	c8.programCounter = address
}

// Skips the next instruction if register VX - whereby X is a placeholder for the number of the register - is equal to KK.
func (c8 *Chip8) op3XKK() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	var value byte = byte((c8.opcode & 0x00FF))

	if c8.registers[vx] == value {
		c8.programCounter += 2
	}
}

// Skips the next instruction if register VX - whereby X is a placeholder for the number of the register - is not equal to KK.
func (c8 *Chip8) op4XKK() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	var value byte = byte((c8.opcode & 0x00FF))

	if c8.registers[vx] != value {
		c8.programCounter += 2
	}
}

// Skips the next instruction if register VX holds the same value as register VY whereby X, Y are placeholder for the number of the register.
func (c8 *Chip8) op5XY0() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	if c8.registers[vx] == c8.registers[vy] {
		c8.programCounter += 2
	}

}

// Loads the value KK into the register X.
func (c8 *Chip8) op6XKK() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var value byte = byte((c8.opcode & 0x00FF))

	c8.registers[vx] = value
}

// Adds KK to the register VX
func (c8 *Chip8) op7XKK() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var value byte = byte((c8.opcode & 0x00FF))

	c8.registers[vx] += value
}

// Load VY into VX
func (c8 *Chip8) op8XY0() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	c8.registers[vx] = c8.registers[vy]
}

// Sets VX to (VX OR VY)
func (c8 *Chip8) op8XY1() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	c8.registers[vx] = (c8.registers[vx] | c8.registers[vy])
}

// Sets VX to (VX AND VY)
func (c8 *Chip8) op8XY2() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	c8.registers[vx] = (c8.registers[vx] & c8.registers[vy])
}

// Sets VX to (VX XOR VY)
func (c8 *Chip8) op8XY3() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	c8.registers[vx] = (c8.registers[vx] ^ c8.registers[vy])
}

// Add VY to VX, sets the carry flag if necessary.
func (c8 *Chip8) op8XY4() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	var result uint16 = uint16(c8.registers[vx]) + uint16(c8.registers[vy])
	if result > 255 {
		c8.registers[0xF] = 1
	} else {
		c8.registers[0xF] = 0
	}

	c8.registers[vx] = uint8((result) & 0xFF)

}

// Subtracts VY to VX, sets flag in VF if VX > VY.
func (c8 *Chip8) op8XY5() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	if c8.registers[vx] > c8.registers[vy] {
		c8.registers[0xF] = 1
	} else {
		c8.registers[0xF] = 0
	}

	c8.registers[vx] -= c8.registers[vy]

}

// Shift the register VX to the right. Sets the VF to one if the least significant bit is one.
func (c8 *Chip8) op8XY6() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	c8.registers[0xF] = (c8.registers[vx] & 0x1)

	c8.registers[vx] = c8.registers[vx] >> 1
}

// Subtracts VY from VX and stores the result in VX. If VY is greater than VX the VF register will be set to one.
func (c8 *Chip8) op8XY7() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	if c8.registers[vy] > c8.registers[vx] {
		c8.registers[0xF] = 1
	} else {
		c8.registers[0xF] = 0
	}

	c8.registers[vx] = c8.registers[vy] - c8.registers[vx]
}

// Shift the register VX to the left. Sets the VF to one if the most significant bit is one.
func (c8 *Chip8) op8XYE() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	c8.registers[0xF] = (c8.registers[vx] & uint8(0b10000000) >> 7)
	c8.registers[vx] = c8.registers[vx] << 1
}

// Skips the next instruction if register VX is not equal VY.
func (c8 *Chip8) op9XY0() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var vy uint8 = uint8((c8.opcode & 0x00F0) >> 4)

	if c8.registers[vx] != c8.registers[vy] {
		c8.programCounter += 2
	}
}

// Sets the index register to NNN.
func (c8 *Chip8) opANNN() {
	var address uint16 = c8.opcode & 0x0FFF

	c8.indexRegister = address
}

// Jumps to address NNN + register V0.
func (c8 *Chip8) opBNNN() {
	var address uint16 = c8.opcode & 0x0FFF

	c8.programCounter = uint16(c8.registers[0x0]) + uint16((address))

}

// Stores the result of a random byte & KK in VX.
func (c8 *Chip8) opCXKK() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var value uint8 = uint8(c8.opcode & 0x00FF)

	c8.registers[vx] = uint8(rand.Intn(255)) & value

}

// Draws the next n bytes from the position of the index register at position (VX, VY).
func (c8 *Chip8) opDXYN() {
	xPos := c8.registers[(c8.opcode&0x0F00)>>8]
	yPos := c8.registers[(c8.opcode&0x00F0)>>4]
	height := c8.opcode & 0x000F

	// Resets the collision register
	c8.registers[0xF] = 0

	for j := uint16(0); j < height; j++ {
		pixel := c8.memory[c8.indexRegister+j]
		for i := uint16(0); i < 8; i++ {
			if (pixel & (0x80 >> i)) != 0 {

				// Checks for collision
				if c8.display[(yPos+uint8(j))%32][(xPos+uint8(i))%64] == 1 {
					c8.registers[0xF] = 1
				}
				c8.display[(yPos+uint8(j))%32][(xPos+uint8(i))%64] ^= 1
			}
		}
	}
}

// Skips the next instruction if the key stored in register VX is pressed.
func (c8 *Chip8) opEX9E() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	var key uint8 = c8.registers[vx]

	if c8.keyPad[key] == 0x1 {
		c8.programCounter += 2
	}
}

// Skips the next instruction if the key stored in register VX is not pressed.
func (c8 *Chip8) opEXA1() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	var key uint8 = c8.registers[vx]

	if c8.keyPad[key] == 0x0 {
		c8.programCounter += 2
	}
}

// Set the value of register VX to the value of the dalay timer.
func (c8 *Chip8) opFX07() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	c8.registers[vx] = c8.delayTimer
}

// Wait for a key press and store the value of the key in VX
func (c8 *Chip8) opFX0A() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	pressed := false
	for key := range c8.keyPad {
		if c8.keyPad[key] != 0 {
			c8.registers[vx] = uint8(key)
			pressed = true
		}
	}

	if !pressed {
		c8.programCounter -= 2
	}
}

// Sets the delay timer to the value that is stored in register VX.
func (c8 *Chip8) opFX15() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	c8.delayTimer = c8.registers[vx]
}

// Sets the sound timer to the value that is stored in register VX.
func (c8 *Chip8) opFX18() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	c8.soundTimer = c8.registers[vx]
}

// Adds the value of register VX to the index register.
func (c8 *Chip8) opFX1E() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	c8.indexRegister += uint16(c8.registers[vx])
}

// Set the index register to the digit that is stored in VX.
func (c8 *Chip8) opFX29() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var digit uint8 = c8.registers[vx]

	c8.indexRegister = uint16(fontStartAddress + int((5 * digit)))

}

// Stores the BCD representation of VX in memory locations I, I+1 and I+2.
func (c8 *Chip8) opFX33() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var value uint8 = c8.registers[vx]

	c8.memory[c8.indexRegister+2] = value % 10
	value /= 10

	c8.memory[c8.indexRegister+1] = value % 10
	value /= 10

	c8.memory[c8.indexRegister] = value % 10

}

// 0xFX65 Fills V0 to VX (including VX) with values from memory starting at address I. I is increased by 1
func (c8 *Chip8) opFX55() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	for i := 0; uint8(i) <= vx; i++ {
		c8.memory[c8.indexRegister+uint16(i)] = c8.registers[i]
	}

	// Reference manual dependent
	c8.indexRegister = ((c8.opcode & 0x0F00) >> 8) + 1
}

// Read into registers V0 - VX from memory starting at location I.
func (c8 *Chip8) opFX65() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	for i := 0; uint8(i) <= vx; i++ {
		c8.registers[i] = c8.memory[c8.indexRegister+uint16(i)]
	}

	c8.indexRegister = ((c8.opcode & 0x0F00) >> 8) + 1
}
