package chip8

import (
	"fmt"
	"image/color"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/waldgaenger/go-acht/internal/input"
	"github.com/waldgaenger/go-acht/internal/renderer"
)

const startAddress int = 0x200
const fontStartAddress int = 0x50
const displayWidth = 64
const displayHeight = 32

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
	foreground color.RGBA
	background color.RGBA
}

var profiles = map[string]colorProfile{
	"black-white": {color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}},
	"night-sky":   {color.RGBA{0, 0, 68, 255}, color.RGBA{255, 255, 204, 255}},
	"console":     {color.RGBA{0, 0, 0, 255}, color.RGBA{34, 238, 34, 255}},
	"honey":       {color.RGBA{153, 102, 0, 255}, color.RGBA{255, 204, 0, 255}},
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

// TODO: Decouple the Chip8 from the SDL library by abstracting the window and display away.
type Chip8 struct {
	registers      [16]uint8   // All 16 registers of the emulator
	memory         [4096]uint8 // 4096 Bytes of RAM
	programCounter uint16      // Holds the next instruction
	indexRegister  uint16
	callStack      [16]uint16
	stackPointer   uint8
	opcode         uint16
	keyPad         [16]uint8
	delayTimer     uint8        // The delay timer is decremented at a rate of 60 Hz according to the specification.
	soundTimer     uint8        // The sound timer is decremented at a rate of 60 Hz according to the specification.
	display        [32][64]bool // 64x32 monochrome display
	scaleFactor    int32        // Holds the scaling factor of the display
	running        bool         // Indicates whether the emulator is running
	colorProfile   colorProfile // Holds the color of the foreground and the background color
	Input          input.InputHandler
	Renderer       renderer.Renderer
}

// Runs the Chip8 emulator with the given ROM and configuration.
// Note that Run never returns unless there the renderer indicates a quit or there is an error.
func (c8 *Chip8) Run(romPath string, scaleFactor int32, colorProfile string) error {

	if err := c8.loadRom(romPath); err != nil {
		return fmt.Errorf("failed to load ROM: %w", err)
	}
	// TODO: Error should be checked and not discarded.
	c8.init(scaleFactor, colorProfile)

	clock := time.NewTicker(time.Millisecond)
	video := time.NewTicker(time.Second / 60)
	sound := time.NewTicker(time.Second / 60)
	delay := time.NewTicker(time.Second / 60)

	for c8.Running() {
		select {
		case <-sound.C:
			if c8.soundTimer > 0 {
				c8.soundTimer--
			}
		case <-delay.C:
			if c8.delayTimer > 0 {
				c8.delayTimer--
			}
		case <-video.C:
			c8.draw()
		case <-clock.C:
			c8.updateInput()
			c8.cycle()
		}
	}

	return nil

}

// Initializes the values of the Chip8 structure.
func (c8 *Chip8) init(scaleFactor int32, colorProfile string) {
	c8.programCounter = uint16(startAddress)
	c8.scaleFactor = scaleFactor
	c8.SetColorProfile(colorProfile)
	// Loading the set of fonts into the specified memory area
	copy(c8.memory[fontStartAddress:], fontSet[:])

	c8.running = true
}

func (c8 *Chip8) Running() bool {
	return c8.running
}

// Sets the color profile specified by the user.
func (c8 *Chip8) SetColorProfile(profileName string) {
	if profile, ok := profiles[profileName]; ok {
		c8.colorProfile = profile
	} else {
		// Default profile which is black-white
		c8.colorProfile = profiles["black-white"]
	}
}

func (c8 *Chip8) draw() {
	c8.Renderer.Draw(c8.display, c8.colorProfile.foreground, c8.colorProfile.background)
}

func (c8 *Chip8) updateInput() {
	quit := c8.Input.PollKeys(&c8.keyPad)
	if quit {
		c8.running = false
	}
}

// fetch fetches the next instruction from the memory and sets the opcode accordingly.
func (c8 *Chip8) fetch() {
	hi := uint16(c8.memory[c8.programCounter])
	lo := uint16(c8.memory[c8.programCounter+1])
	c8.opcode = (hi << 8) | lo
}

// cycle carries out one full CPU cycle: fetches the next opcode, decodes it using the dispatch table, and executes the matching instruction handler.
func (c8 *Chip8) cycle() {
	c8.fetch()
	c8.programCounter += 2

	if handler := dispatchTable[c8.decodeOpcode()]; handler != nil {
		handler(c8)
	} else {
		fmt.Printf("Invalid opcode: %#X\n", c8.opcode)
	}

}

// loadRom loads the ROM from a given path into the CHIP8 memory.
func (c8 *Chip8) loadRom(pathToRom string) error {
	data, err := os.ReadFile(pathToRom)
	if err != nil {
		return fmt.Errorf("could not open ROM file: %w", err)
	}

	if len(data) > len(c8.memory)-startAddress {
		return fmt.Errorf("ROM (%d bytes) is too large for memory (%d bytes available)", len(data), len(c8.memory)-startAddress)
	}

	copy(c8.memory[startAddress:], data)

	slog.Info("ROM successfully loaded into memory", "size", len(data), "path", pathToRom)
	return nil
}

// decodeOpcode decodes the the current opcode and returns the value.
// Intended to be used for the dispatch map to find the corresponding functions which realizes the instruction.
func (c8 *Chip8) decodeOpcode() uint16 {
	switch c8.opcode & 0xF000 {
	case 0x0000:
		return c8.opcode & 0x00FF // e.g. 00E0, 00EE
	case 0x8000:
		return c8.opcode & 0xF00F // e.g. 8XY0
	case 0xE000, 0xF000:
		return c8.opcode & 0xF0FF // e.g. EX9E, FX07
	default:
		return c8.opcode & 0xF000 // e.g. 1000, 2000, etc.
	}
}

// Clears the display by resetting all pixels to 0.
func (c8 *Chip8) op00E0() {
	c8.display = [32][64]bool{}
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

	c8.registers[vx] = uint8(rand.Intn(256)) & value
}

// Draws the next n bytes from the position of the index register at position (VX, VY).
func (c8 *Chip8) opDXYN() {
	// TODO: Refactor and beautify; make it more readable.
	xPos := c8.registers[(c8.opcode&0x0F00)>>8]
	yPos := c8.registers[(c8.opcode&0x00F0)>>4]
	height := c8.opcode & 0x000F

	// Resets the collision register
	c8.registers[0xF] = 0

	for j := uint16(0); j < height; j++ {
		pixel := c8.memory[c8.indexRegister+j]
		for i := uint16(0); i < 8; i++ {
			if (pixel & (0x80 >> i)) != 0 {
				// TODO: Should be simplied!
				// Checks for collision
				if c8.display[(yPos+uint8(j))%displayHeight][(xPos+uint8(i))%displayWidth] == true { // TODO: Remove magic numbers
					c8.registers[0xF] = 1
				}
				c8.display[(yPos+uint8(j))%displayHeight][(xPos+uint8(i))%displayWidth] = !c8.display[(yPos+uint8(j))%displayHeight][(xPos+uint8(i))%displayWidth] // TODO: Remove magic numbers
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

// Sets the value of register VX to the value of the dalay timer.
func (c8 *Chip8) opFX07() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	c8.registers[vx] = c8.delayTimer
}

// Waits for a key press and store the value of the key in VX
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

// Sets the index register to the digit that is stored in VX.
func (c8 *Chip8) opFX29() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var digit uint8 = c8.registers[vx]

	c8.indexRegister = uint16(fontStartAddress + int((5 * digit)))

}

// Stores the BCD representation of VX in memory locations I, I+1 and I+2.
// Takes the decimal value of VX, and places the hundreds digit in memory at location in I,
// the tens digit at location I+1, and the ones digit at location I+2.
func (c8 *Chip8) opFX33() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)
	var value uint8 = c8.registers[vx]

	c8.memory[c8.indexRegister+2] = value % 10
	value /= 10

	c8.memory[c8.indexRegister+1] = value % 10
	value /= 10

	c8.memory[c8.indexRegister] = value % 10

}

// Fills V0 to VX (including VX) with values from memory starting at address I. I is increased by 1
func (c8 *Chip8) opFX55() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	for i := 0; uint8(i) <= vx; i++ {
		c8.memory[c8.indexRegister+uint16(i)] = c8.registers[i]
	}

	// Reference manual dependent
	c8.indexRegister = ((c8.opcode & 0x0F00) >> 8) + 1
}

// Reads into registers V0 - VX from memory starting at location I.
func (c8 *Chip8) opFX65() {
	var vx uint8 = uint8((c8.opcode & 0x0F00) >> 8)

	for i := 0; uint8(i) <= vx; i++ {
		c8.registers[i] = c8.memory[c8.indexRegister+uint16(i)]
	}

	c8.indexRegister = ((c8.opcode & 0x0F00) >> 8) + 1
}
