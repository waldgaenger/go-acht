package chip8

import (
	"fmt"
	"log"
	"math/rand"
	"os"

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

type Chip8 struct {
	registers      [16]uint8
	memory         [4096]uint8
	programCounter uint16
	indexRegister  uint16
	callStack      [16]uint16
	stackPointer   uint8
	opcode         uint16
	keyPad         [16]uint8
	// The delay timer is decremented at a rate of 60 Hz according to the specification.
	delayTimer uint8
	// The sound timer is decremented at a rate of 60 Hz according to the specification.
	soundTimer uint8
	// 64x32 monochrome display.
	display             [32][64]uint8
	sdlWindow           *sdl.Window
	scaleFactor         int32
	running             bool
	colorCodeForeground uint32
	colorCodeBackground uint32
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

	window, err := sdl.CreateWindow("CHIP8 EMULATOR", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		64*c8.scaleFactor, 32*c8.scaleFactor, sdl.WINDOW_SHOWN)

	if err != nil {
		fmt.Printf("An error occurred  while trying to create the SDL window: %v", err)
		sdl.Quit()
		os.Exit(-1)
	}
	if colorProfile == "black-white" {
		c8.colorCodeBackground = 0x0000000
		c8.colorCodeForeground = 0xFFFFFFF
	}
	if colorProfile == "night-sky" {
		c8.colorCodeBackground = 0x000044
		c8.colorCodeForeground = 0xFFFFCC
	}
	if colorProfile == "console" {
		c8.colorCodeBackground = 0x000000
		c8.colorCodeForeground = 0x22EE22
	}
	if colorProfile == "honey" {
		c8.colorCodeBackground = 0x996600
		c8.colorCodeForeground = 0xFFCC00
	}
	c8.sdlWindow = window
	c8.running = true
}

func (c8 *Chip8) Running() bool {
	return c8.running
}

func (c8 *Chip8) draw() {
	surface, err := c8.sdlWindow.GetSurface()
	if err != nil {
		panic(err)
	}

	for row, rows := range c8.display {
		for column := range rows {
			rect := sdl.Rect{X: (int32(column * int(c8.scaleFactor))), Y: (int32(row * int(c8.scaleFactor))), H: c8.scaleFactor, W: c8.scaleFactor}
			if c8.display[row][column] == 0 {
				surface.FillRect(&rect, c8.colorCodeBackground)
			} else {
				surface.FillRect(&rect, c8.colorCodeForeground)
			}
		}
	}
	c8.sdlWindow.UpdateSurface()
}

func (c8 *Chip8) DisplayDebugger() {
	surface, err := c8.sdlWindow.GetSurface()

	if err != nil {
		panic(err)
	}
	for row, rows := range c8.display {
		for column := range rows {
			rect := sdl.Rect{X: (int32(column * 20)), Y: (int32(row * 20)), H: 20, W: 20}
			if c8.display[row][column] == 0 {
				if row%2 == 0 && column%2 == 0 {
					surface.FillRect(&rect, 0xd9d9d9)
				}
				if row%2 == 0 && column%2 != 0 {
					surface.FillRect(&rect, 0xFFFFFFFF)
				}
				if row%2 != 0 && column%2 == 0 {
					surface.FillRect(&rect, 0xFFFFFFFF)
				}
				if row%2 != 0 && column%2 != 0 {
					surface.FillRect(&rect, 0xd9d9d9)
				}
			} else {
				surface.FillRect(&rect, 0x37FE65)
			}
		}
	}
}
func (c8 *Chip8) keyHandler() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch et := event.(type) {
		case *sdl.QuitEvent:
			c8.running = false
		case *sdl.KeyboardEvent:
			if et.Type == sdl.KEYUP {
				switch et.Keysym.Sym {
				case sdl.K_1:
					c8.keyPad[0x1] = 0
				case sdl.K_2:
					c8.keyPad[0x2] = 0
				case sdl.K_3:
					c8.keyPad[0x3] = 0
				case sdl.K_4:
					c8.keyPad[0xC] = 0
				case sdl.K_q:
					c8.keyPad[0x4] = 0
				case sdl.K_w:
					c8.keyPad[0x5] = 0
				case sdl.K_e:
					c8.keyPad[0x6] = 0
				case sdl.K_r:
					c8.keyPad[0xD] = 0
				case sdl.K_a:
					c8.keyPad[0x7] = 0
				case sdl.K_s:
					c8.keyPad[0x8] = 0
				case sdl.K_d:
					c8.keyPad[0x9] = 0
				case sdl.K_f:
					c8.keyPad[0xE] = 0
				case sdl.K_z:
					c8.keyPad[0xA] = 0
				case sdl.K_x:
					c8.keyPad[0x0] = 0
				case sdl.K_c:
					c8.keyPad[0xB] = 0
				case sdl.K_v:
					c8.keyPad[0xF] = 0
				}
			} else if et.Type == sdl.KEYDOWN {
				switch et.Keysym.Sym {
				case sdl.K_1:
					c8.keyPad[0x1] = 1
				case sdl.K_2:
					c8.keyPad[0x2] = 1
				case sdl.K_3:
					c8.keyPad[0x3] = 1
				case sdl.K_4:
					c8.keyPad[0xC] = 1
				case sdl.K_q:
					c8.keyPad[0x4] = 1
				case sdl.K_w:
					c8.keyPad[0x5] = 1
				case sdl.K_e:
					c8.keyPad[0x6] = 1
				case sdl.K_r:
					c8.keyPad[0xD] = 1
				case sdl.K_a:
					c8.keyPad[0x7] = 1
				case sdl.K_s:
					c8.keyPad[0x8] = 1
				case sdl.K_d:
					c8.keyPad[0x9] = 1
				case sdl.K_f:
					c8.keyPad[0xE] = 1
				case sdl.K_z:
					c8.keyPad[0xA] = 1
				case sdl.K_x:
					c8.keyPad[0x0] = 1
				case sdl.K_c:
					c8.keyPad[0xB] = 1
				case sdl.K_v:
					c8.keyPad[0xF] = 1
				}
			}
		}
	}
}

func (c8 *Chip8) Cycle() {

	// Fetch
	c8.opcode = ((uint16(c8.memory[c8.programCounter]) << 8) | uint16(c8.memory[c8.programCounter+1]))

	c8.programCounter += 2

	// Key handling
	c8.keyHandler()
	// Decode and execute
	switch c8.opcode & 0xF000 {
	case 0x0000:
		switch c8.opcode & 0x00FF {
		case 0x00E0:
			c8.op00E0()
		case 0x00EE:
			c8.op00EE()
		}
	case 0x1000:
		c8.op1NNN()
	case 0x2000:
		c8.op2NNN()
	case 0x3000:
		c8.op3XKK()
	case 0x4000:
		c8.op4XKK()
	case 0x5000:
		c8.op5XY0()
	case 0x6000:
		c8.op6XKK()
	case 0x7000:
		c8.op7XKK()
	case 0x8000:
		switch c8.opcode & 0x000F {
		case 0x0000:
			c8.op8XY0()
		case 0x0001:
			c8.op8XY1()
		case 0x0002:
			c8.op8XY2()
		case 0x0003:
			c8.op8XY3()
		case 0x0004:
			c8.op8XY4()
		case 0x0005:
			c8.op8XY5()
		case 0x0006:
			c8.op8XY6()
		case 0x0007:
			c8.op8XY7()
		case 0x000E:
			c8.op8XYE()
		}
	case 0x9000:
		c8.op9XY0()
	case 0xA000:
		c8.opANNN()
	case 0xB000:
		c8.opBNNN()
	case 0xC000:
		c8.opCXKK()
	case 0xD000:
		c8.opDXYN()
		c8.draw()
	case 0xE000:
		switch c8.opcode & 0x00FF {
		case 0x009E:
			c8.opEX9E()
		case 0x00A1:
			c8.opEXA1()
		}
	case 0xF000:
		switch c8.opcode & 0x00FF {
		case 0x0007:
			c8.opFX07()
		case 0x000A:
			c8.opFX0A()
		case 0x0015:
			c8.opFX15()
		case 0x0018:
			c8.opFX18()
		case 0x001E:
			c8.opFX1E()
		case 0x0029:
			c8.opFX29()
		case 0x0033:
			c8.opFX33()
		case 0x0055:
			c8.opFX55()
		case 0x0065:
			c8.opFX65()
		}
	default:
		fmt.Printf("Invalid opcode: %#X\n", c8.opcode)

	}
	if c8.delayTimer > 0 {
		c8.delayTimer--
	}

	if c8.soundTimer > 0 {
		c8.soundTimer--
	}
}

func (c8 *Chip8) ShutDown() {
	c8.running = false
	c8.sdlWindow.Destroy()
	sdl.Quit()
}

func (c8 *Chip8) LoadRom(pathToRom string) {
	f, err := os.Open(pathToRom)

	if err != nil {
		f.Close()
		log.Fatalf("An error occurred while trying to open the ROM: %v\n", err)
	}

	defer f.Close()

	bytesRead, err := f.Read(c8.memory[startAddress:])

	if err != nil {
		f.Close()
		log.Fatalf("An error occurred while trying to read the ROM into memory: %v\n", err)
	}

	fmt.Println("[+] ROM successfully read into the memory.")
	fmt.Println("[+] ROM size: ", bytesRead)

}

func (c8 *Chip8) PrintRegisters() {
	for regIndex, regValue := range c8.registers {
		fmt.Printf("%#X: %08b\t", regIndex, regValue)
	}

	fmt.Printf("\n\nIndex register: %#X\n", c8.indexRegister)
}

func (c8 *Chip8) PrintStatus() {
	c8.PrintRegisters()

	fmt.Printf("\nProgram counter: %#X\n", c8.programCounter)
	fmt.Println("----------------------")
	for stackIndex, stackValue := range c8.callStack {
		if stackIndex == int(c8.stackPointer) {
			fmt.Printf("|%#X: %#X   <---SP   |\n", stackIndex, stackValue)
		} else {
			fmt.Printf("|%#X: %#X            |\n", stackIndex, stackValue)
		}
	}
	fmt.Println("----------------------")
}

func (c8 *Chip8) PrintMemoryStatus() {
	var i uint16 = 0x200
	for ; i < 612; i++ {
		fmt.Printf("%#X: %#X\n", i, c8.memory[i])
	}
}

func (c8 *Chip8) GetDisplay() [32][64]uint8 {
	return c8.display
}

// Clears the screen.
// Sets all values of the display to 0.
func (c8 *Chip8) op00E0() {
	for i, arr := range c8.display {
		for j := range arr {
			c8.display[i][j] = 0
		}
	}
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

	fmt.Printf("Result: %#X\n", (c8.registers[vx] & 0b10000000))

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
