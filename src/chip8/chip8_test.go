package chip8

import (
	"fmt"
	"os"
	"testing"
)

func TestLoadRom(t *testing.T) {
	type testCase struct {
		name        string
		romData     []byte
		romTooLarge bool
		romMissing  bool
		wantErr     bool
		wantFirst   []byte
		wantLast    []byte
	}

	romFits := []byte{0xA2, 0xB4, 0x23, 0xE6, 0x00, 0xEE, 0x37, 0x23}
	romTooBig := make([]byte, 4096-startAddress+1)
	for i := range romTooBig {
		romTooBig[i] = 0xFF
	}

	tests := []testCase{
		{
			name:      "Valid ROM",
			romData:   romFits,
			wantErr:   false,
			wantFirst: romFits[:4],
			wantLast:  romFits[len(romFits)-4:],
		},
		{
			name:        "ROM too large",
			romData:     romTooBig,
			romTooLarge: true,
			wantErr:     true,
		},
		{
			name:       "Missing ROM file",
			romMissing: true,
			wantErr:    true,
		},
		{
			name:    "Empty ROM file",
			romData: []byte{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c8 := &Chip8{}

			var romPath string
			if tt.romMissing {
				romPath = "nonexistent.rom"
			} else {
				tmpFile, err := os.CreateTemp("", "chip8romtest_*.rom")
				if err != nil {
					t.Fatalf("could not create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				if len(tt.romData) > 0 {
					if _, err := tmpFile.Write(tt.romData); err != nil {
						t.Fatalf("could not write to temp file: %v", err)
					}
				}
				tmpFile.Close()
				romPath = tmpFile.Name()
			}

			err := c8.loadRom(romPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}

			if len(tt.romData) >= 4 {
				gotFirst := c8.memory[startAddress : startAddress+4]
				if !equalBytes(gotFirst, tt.wantFirst) {
					t.Errorf("first 4 bytes: got %v, want %v", gotFirst, tt.wantFirst)
				}
				gotLast := c8.memory[startAddress+len(tt.romData)-4 : startAddress+len(tt.romData)]
				if !equalBytes(gotLast, tt.wantLast) {
					t.Errorf("last 4 bytes: got %v, want %v", gotLast, tt.wantLast)
				}
			}
		})
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func TestCycle(t *testing.T) {
	tests := []struct {
		name           string
		memory         []byte
		cycles         int
		startPC        uint16
		wantPC         uint16
		wantRegisterV0 byte
	}{
		{
			name:           "5x CLS (0x00E0) - PC should advance +10",
			memory:         []byte{0x00, 0xE0, 0x00, 0xE0, 0x00, 0xE0, 0x00, 0xE0, 0x00, 0xE0},
			cycles:         5,
			startPC:        0x200,
			wantPC:         0x20A,
			wantRegisterV0: 0,
		},
		{
			name:           "1x LD V0, 0x42 (0x6042) - V0 should be set to 0x42",
			memory:         []byte{0x60, 0x42},
			cycles:         1,
			startPC:        0x200,
			wantPC:         0x202,
			wantRegisterV0: 0x42,
		},
		{
			name:           "2x LD V0, 0x12; LD V0, 0x34",
			memory:         []byte{0x60, 0x12, 0x60, 0x34},
			cycles:         2,
			startPC:        0x200,
			wantPC:         0x204,
			wantRegisterV0: 0x34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c8 := Chip8{}
			c8.programCounter = tt.startPC

			for i, b := range tt.memory {
				c8.memory[0x200+uint16(i)] = b
			}

			for i := 0; i < tt.cycles; i++ {
				c8.cycle()
			}

			if c8.programCounter != tt.wantPC {
				t.Errorf("PC: got 0x%X, want 0x%X", c8.programCounter, tt.wantPC)
			}
			if c8.registers[0] != tt.wantRegisterV0 {
				t.Errorf("V0: got 0x%X, want 0x%X", c8.registers[0], tt.wantRegisterV0)
			}
		})
	}
}

func TestOP00E0(t *testing.T) {
	c8 := Chip8{}

	for i, arr := range c8.display {
		for j := range arr {
			c8.display[i][j] = 1
		}
	}

	c8.op00E0()

	t.Run("OP00E0: Clearing the screen", func(t *testing.T) {
		for i, arr := range c8.display {
			for j := range arr {
				if c8.display[i][j] != 0 {
					t.Errorf("Expected all values to be zero but pixel [%d][%d] was not zero.", i, j)
				}
			}
		}
	})
}

func TestOP00EE(t *testing.T) {
	c8 := Chip8{}

	c8.callStack[0] = 0xFFF
	c8.callStack[1] = 0x200
	c8.callStack[2] = 0xF2F
	c8.stackPointer = 2
	c8.op00EE()

	t.Run("OP00EE: Returning from subroutine", func(t *testing.T) {
		if c8.programCounter != 0x200 {
			t.Errorf("Program counter -> got: %x, want: %x", c8.programCounter, 0x200)
		}
	})

	c8.op00EE()

	t.Run("OP00EE: Returning from subroutine", func(t *testing.T) {
		if c8.programCounter != 0xFFF {
			t.Errorf("Program counter -> got: %x, want: %x", c8.programCounter, 0xFFF)
		}
	})

}

func TestOP1NNN(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName string
		opcode   uint16
		want     uint16
	}{
		{
			"Jump to address 0xFFF",
			0x1FFF,
			0xFFF,
		},
		{
			"Jump to address 0x200",
			0x1200,
			0x200,
		},
		{
			"Jump to address 0x500",
			0x1500,
			0x500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.op1NNN()
			if c8.programCounter != tt.want {
				t.Errorf("Expected program counter to be %x but got %x", tt.want, c8.programCounter)
			}
		})
	}
}

func TestOP2NNN(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName string
		opcode   uint16
		want     uint16
	}{
		{
			"Call subroutine on 0xF34",
			0x2F34,
			0xF34,
		},
		{
			"Call subroutine on 0x241",
			0x2241,
			0x241,
		},
		{
			"Call subroutine on 0x500",
			0x2500,
			0x500,
		},
	}

	for index, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.programCounter = uint16(0x200 + index)
			c8.opcode = tt.opcode
			c8.op2NNN()
			if c8.programCounter != tt.want {
				t.Errorf("Expected program counter to be %x but got %x", tt.want, c8.programCounter)
			}
			if c8.callStack[c8.stackPointer-1] != uint16((0x200 + index)) {
				t.Errorf("Expected the call stack value to be %x but got %x", (0x200 + index), c8.callStack[c8.stackPointer-1])
			}
		})
	}
}

func TestOP3XKK(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		register             uint8
		registerValue        uint8
		programCounterBefore uint16
		programCounterAfter  uint16
	}{
		{
			"Register V5 equal FF",
			0x35FF,
			0x5,
			0xFF,
			0x0000,
			0x0002,
		},
		{
			"Register V6 equal F4",
			0x36F4,
			0x6,
			0xF4,
			0x00F0,
			0x00F2,
		},
		{
			"Register V7 not equal FF",
			0x37FF,
			0x7,
			0x11,
			0x0000,
			0x0000,
		},
		{
			"Register V8 not equal 2F",
			0x382F,
			0x8,
			0x22,
			0x0031,
			0x0031,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.programCounter = tt.programCounterBefore
			c8.opcode = tt.opcode
			c8.registers[tt.register] = tt.registerValue
			c8.op3XKK()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected program counter to be %x but got %x", tt.programCounterAfter, c8.programCounter)
			}
		})
	}

}

func TestOP4XKK(t *testing.T) {
	// TODO: Fix OPCODE
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		register             uint8
		registerValue        uint8
		programCounterBefore uint16
		programCounterAfter  uint16
	}{
		{
			"Register V5 equal FF",
			0x45FF,
			0x5,
			0xFF,
			0x0000,
			0x0000,
		},
		{
			"Register V6 equal F4",
			0x46F4,
			0x6,
			0xF4,
			0x00F0,
			0x00F0,
		},
		{
			"Register V7 not equal FF",
			0x47FF,
			0x7,
			0x11,
			0x0000,
			0x0002,
		},
		{
			"Register V8 not equal 2F",
			0x482F,
			0x8,
			0x22,
			0x0031,
			0x0033,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.programCounter = tt.programCounterBefore
			c8.opcode = tt.opcode
			c8.registers[tt.register] = tt.registerValue
			c8.op4XKK()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected program counter to be %x but got %x", tt.programCounterAfter, c8.programCounter)
			}
		})
	}

}

func TestOP5XY0(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		registerX            uint8
		registerValueX       uint8
		registerY            uint8
		registerValueY       uint8
		programCounterBefore uint16
		programCounterAfter  uint16
	}{
		{
			"Register V0 equal V8",
			0x5080,
			0x0,
			0x43,
			0x8,
			0x43,
			0x0F00,
			0x0F02,
		},

		{
			"Register V5 equal VE",
			0x5080,
			0x5,
			0xDF,
			0xE,
			0xDF,
			0xD000,
			0xD002,
		},

		{
			"Register V3 not equal VA",
			0x53A0,
			0x3,
			0x43,
			0xA,
			0x44,
			0x0F00,
			0x0F00,
		},

		{
			"Register VC not equal VD",
			0x5CD0,
			0xC,
			0x13,
			0xD,
			0xA4,
			0x0200,
			0x0200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.programCounter = tt.programCounterBefore
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op5XY0()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected program counter to be %#X but got %#X", tt.programCounterAfter, c8.programCounter)
			}
		})
	}
}

func TestOP6XKK(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName      string
		opcode        uint16
		register      uint8
		registerValue uint8
	}{
		{
			"Load 0xFF into V0",
			0x60FF,
			0x0,
			0xFF,
		},
		{
			"Load 0xCC into VD",
			0x6DCC,
			0xD,
			0xCC,
		},
		{
			"Load 0xA1 into V8",
			0x68A1,
			0x8,
			0xA1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.op6XKK()

			if c8.registers[tt.register] != tt.registerValue {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.registerValue, c8.registers[tt.register])
			}
		})
	}

}

func TestOP7XKK(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName      string
		opcode        uint16
		register      uint8
		registerValue uint8
	}{
		{
			"Add 0xFF to V0",
			0x70FF,
			0x0,
			0xFF,
		},
		{
			"Add 0xCC to VD",
			0x7DCC,
			0xD,
			0xCC,
		},
		{
			"Add 0xA1 to V8",
			0x78A1,
			0x8,
			0xA1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.op7XKK()

			if c8.registers[tt.register] != tt.registerValue {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.registerValue, c8.registers[tt.register])
			}
		})
	}

}

func TestOP8XY0(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName      string
		opcode        uint16
		registerX     uint8
		registerY     uint8
		registerValue uint8
	}{
		{
			"Load V2 into V0",
			0x8020,
			0x0,
			0x2,
			0xD0,
		},
		{
			"Load VF into V9",
			0x89F0,
			0x9,
			0xF,
			0xAB,
		},
		{
			"Load V3 into V5",
			0x8530,
			0x5,
			0x3,
			0x34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerY] = tt.registerValue
			c8.op8XY0()

			if c8.registers[tt.registerX] != tt.registerValue {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.registerValue, c8.registers[tt.registerX])
			}
		})
	}

}

func TestOP8XY1(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName       string
		opcode         uint16
		registerX      uint8
		registerY      uint8
		registerValueX uint8
		registerValueY uint8
		expectedResult uint8
	}{
		{
			"Sets V0 to V0 OR V2",
			0x8021,
			0x0,
			0x2,
			0xA0,
			0xB2,
			0xB2,
		},
		{
			"Sets V6 to V6 OR VB",
			0x86B1,
			0x6,
			0xB,
			0xA9,
			0xD2,
			0xFB,
		},
		{
			"Sets VD to VD OR V3",
			0x8D31,
			0xD,
			0x3,
			0x9A,
			0x3C,
			0xBE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op8XY1()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}
		})
	}

}

func TestOP8XY2(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName       string
		opcode         uint16
		registerX      uint8
		registerY      uint8
		registerValueX uint8
		registerValueY uint8
		expectedResult uint8
	}{
		{
			"Sets V0 to V0 OR V2",
			0x8022,
			0x0,
			0x2,
			0xA0,
			0xB2,
			0xA0,
		},
		{
			"Sets V6 to V6 OR VB",
			0x86B2,
			0x6,
			0xB,
			0xA9,
			0xD2,
			0x80,
		},
		{
			"Sets VD to VD OR V3",
			0x8D32,
			0xD,
			0x3,
			0x9A,
			0x3C,
			0x18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op8XY2()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}
		})
	}

}

func TestOP8XY3(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName       string
		opcode         uint16
		registerX      uint8
		registerY      uint8
		registerValueX uint8
		registerValueY uint8
		expectedResult uint8
	}{
		{
			"Sets V0 to V0 XOR V2",
			0x8023,
			0x0,
			0x2,
			0xA0,
			0xB2,
			0x12,
		},
		{
			"Sets V6 to V6 XOR VB",
			0x86B3,
			0x6,
			0xB,
			0xA9,
			0xD2,
			0x7B,
		},
		{
			"Sets VD to VD XOR V3",
			0x8D33,
			0xD,
			0x3,
			0x9A,
			0x3C,
			0xA6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op8XY3()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}
		})
	}

}

func TestOP8XY4(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerY        uint8
		registerValueX   uint8
		registerValueY   uint8
		expectedResult   uint8
		expectedCarryBit bool
	}{
		{
			"Adds V2 to V0",
			0x8024,
			0x0,
			0x2,
			0xA0,
			0xB2,
			0x52,
			true,
		},
		{
			"Adds V5 to VE",
			0x85E4,
			0x5,
			0xE,
			0x0F,
			0xAF,
			0xBE,
			false,
		},
		{
			"Adds V9 to VA",
			0x89A4,
			0x9,
			0xA,
			0xAF,
			0xAF,
			0x5E,
			true,
		},
		{
			"Adds VD to V1",
			0x8D14,
			0xD,
			0x1,
			0x0F,
			0x01,
			0x10,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op8XY4()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}

			if tt.expectedCarryBit {
				if c8.registers[0xF] != 1 {
					t.Errorf("Expected the value of VF to be 0x01 but got %#X", c8.registers[0xF])
				}
			}
		})
	}

}

func TestOP8XY5(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerY        uint8
		registerValueX   uint8
		registerValueY   uint8
		expectedResult   uint8
		expectedCarryBit bool
	}{
		{
			"Subtracts V2 from V0",
			0x8025,
			0x0,
			0x2,
			0xA0,
			0xB2,
			0xEE,
			false,
		},
		{
			"Subtracts VE from V5",
			0x85E5,
			0x5,
			0xE,
			0xF0,
			0xAF,
			0x41,
			true,
		},
		{
			"Subtracts V8 from VA",
			0x8A85,
			0xA,
			0x8,
			0xD7,
			0xF3,
			0xE4,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op8XY5()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}

			if tt.expectedCarryBit {
				if c8.registers[0xF] != 1 {
					t.Errorf("Expected the value of VF to be 0x01 but got %#X", c8.registers[0xF])
				}
			}
		})
	}

}

func TestOP8XY6(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerValueX   uint8
		expectedResult   uint8
		expectedCarryBit bool
	}{
		{
			"Shift right on register V0",
			0x8006,
			0x0,
			0x01,
			0x0,
			true,
		},
		{
			"Shift right on register VE",
			0x8E06,
			0xE,
			0xF0,
			0x78,
			false,
		},
		{
			"Shift right on register VA",
			0x8A06,
			0xA,
			0xFF,
			0x7F,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.op8XY6()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}

			if tt.expectedCarryBit {
				if c8.registers[0xF] != 1 {
					t.Errorf("Expected the value of VF to be 0x01 but got %#X", c8.registers[0xF])
				}
			}
		})
	}
}

func TestOP8XY7(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerY        uint8
		registerValueX   uint8
		registerValueY   uint8
		expectedResult   uint8
		expectedCarryBit bool
	}{
		{
			"Subtracts V0 from V2 and stores the result in V0",
			0x8027,
			0x0,
			0x2,
			0xA0,
			0xB2,
			0x12,
			true,
		},
		{
			"Subtracts VE from V5 and stores the result in V5",
			0x85E7,
			0x5,
			0xE,
			0xF0,
			0xAF,
			0xBF,
			false,
		},
		{
			"Subtracts VA from V8 and stores the result in VA",
			0x8A87,
			0xA,
			0x8,
			0xD7,
			0xF3,
			0x1C,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.op8XY7()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}

			if tt.expectedCarryBit {
				if c8.registers[0xF] != 1 {
					t.Errorf("Expected the value of VF to be 0x01 but got %#X", c8.registers[0xF])
				}
			}
		})
	}

}

func TestOP8XYE(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerValueX   uint8
		expectedResult   uint8
		expectedCarryBit bool
	}{
		{
			"Shift left on register V0",
			0x800E,
			0x0,
			0x01,
			0x02,
			false,
		},
		{
			"Shift left on register V7",
			0x870E,
			0x7,
			0x69,
			0xD2,
			false,
		},
		{
			"Shift left on register VE",
			0x8E0E,
			0xE,
			0xF0,
			0xE0,
			true,
		},
		{
			"Shift left on register VA",
			0x8A0E,
			0xA,
			0xFF,
			0xFE,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.op8XYE()

			if c8.registers[tt.registerX] != tt.expectedResult {
				t.Errorf("Expected the register value to be %#X but got %#X.", tt.expectedResult, c8.registers[tt.registerX])
			}

			if tt.expectedCarryBit {
				if c8.registers[0xF] != 1 {
					t.Errorf("Expected the value of VF to be 0x01 but got %#X", c8.registers[0xF])
				}
			}
		})
	}
}

func TestOP9XY0(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		registerX            uint8
		registerY            uint8
		registerValueX       uint8
		registerValueY       uint8
		programCounterBefore uint16
		programCounterAfter  uint16
	}{
		{
			"V5 is not equal to V8",
			0x9580,
			0x5,
			0x8,
			0xF0,
			0xA3,
			0x0200,
			0x0202,
		},
		{
			"VA is equal to V0",
			0x90A0,
			0x0,
			0xA,
			0x83,
			0x83,
			0x0200,
			0x0200,
		},
		{
			"V1 is not equal to V2",
			0x9120,
			0x1,
			0x2,
			0xFF,
			0xAA,
			0x0200,
			0x0202,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.registers[tt.registerY] = tt.registerValueY
			c8.programCounter = tt.programCounterBefore

			c8.op9XY0()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected the program counter to be %#X but got %#X.", tt.programCounterAfter, c8.programCounter)
			}
		})
	}
}

func TestOPANNN(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName        string
		opcode          uint16
		expectedAddress uint16
	}{
		{
			"Set the index register to 0x200",
			0xA200,
			0x200,
		},
		{
			"Set the index register to 0x541",
			0xA541,
			0x541,
		},
		{
			"Set the index register to 0xAB9",
			0xAAB9,
			0xAB9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.opANNN()

			if c8.indexRegister != tt.expectedAddress {
				t.Errorf("Expected the index register to be %#X but got %#X.", tt.expectedAddress, c8.indexRegister)
			}
		})
	}
}

func TestOPBNNN(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName        string
		opcode          uint16
		registerValue   uint8
		expectedAddress uint16
	}{
		{
			"Jump to address 0x2B7",
			0xB200,
			0xB7,
			0x2B7,
		},
		{
			"Jump to address 0x49C",
			0xB400,
			0x9C,
			0x49C,
		},
		{
			"Jump to address 0xF94",
			0xBF0F,
			0x85,
			0xF94,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[0x0] = tt.registerValue
			c8.opBNNN()

			if c8.programCounter != tt.expectedAddress {
				t.Errorf("Expected the program counter to be %#X but got %#X.", tt.expectedAddress, c8.programCounter)
			}
		})
	}
}

func TestOPEX9E(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		registerX            uint8
		registerValueX       uint8
		programCounterBefore uint16
		programCounterAfter  uint16
		pressed              bool
	}{
		{
			"Pressed the key on register 0x8",
			0xE89E,
			0x8,
			0x8,
			0x200,
			0x202,
			true,
		},
		{
			"Not pressed the key on register 0x9",
			0xE99E,
			0x9,
			0x9,
			0x200,
			0x200,
			false,
		},
		{
			"Not pressed the key on register 0xA",
			0xEA9E,
			0xA,
			0xA,
			0x200,
			0x200,
			false,
		},
		{
			"Pressed the key on register 0xA",
			0xEA9E,
			0xA,
			0xA,
			0x200,
			0x202,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.programCounter = tt.programCounterBefore

			if tt.pressed {
				c8.keyPad[tt.registerValueX] = 0x1
			}

			c8.opEX9E()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected the program counter to be %#X but got %#X.", tt.programCounterAfter, c8.programCounter)
			}

		})
	}

}
func TestOPEXA1(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		registerX            uint8
		registerValueX       uint8
		programCounterBefore uint16
		programCounterAfter  uint16
		pressed              bool
	}{
		{
			"Pressed the key on register 0x8",
			0xE89E,
			0x8,
			0x8,
			0x200,
			0x200,
			true,
		},
		{
			"Not pressed the key on register 0x9",
			0xE99E,
			0x9,
			0x9,
			0x200,
			0x202,
			false,
		},
		{
			"Not pressed the key on register 0xA",
			0xEA9E,
			0xA,
			0xA,
			0x200,
			0x202,
			false,
		},
		{
			"Pressed the key on register 0xA",
			0xEA9E,
			0xA,
			0xA,
			0x200,
			0x200,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.registers[tt.registerX] = tt.registerValueX
			c8.programCounter = tt.programCounterBefore

			if tt.pressed {
				c8.keyPad[tt.registerValueX] = 0x1
			}

			c8.opEXA1()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected the program counter to be %#X but got %#X.", tt.programCounterAfter, c8.programCounter)
			}

		})
	}

}

func TestOPFX07(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName   string
		opcode     uint16
		registerX  uint8
		delayTimer uint8
	}{
		{
			"Set the register 0xB to the value of the delay timer -> 0x0A",
			0xFB07,
			0xB,
			0x0A,
		},
		{
			"Set the register 0x7 to the value of the delay timer -> 0xFF",
			0xF707,
			0x7,
			0xFF,
		},
		{
			"Set the register 0x3 to the value of the delay timer -> 0xC9",
			0xF307,
			0x3,
			0xC9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.delayTimer = tt.delayTimer

			c8.opFX07()

			if c8.registers[tt.registerX] != tt.delayTimer {
				t.Errorf("Expected the register %#X to be %#X but got %#X.", tt.registerX, tt.delayTimer, c8.registers[tt.registerX])
			}
		})
	}
}

func TestOPFX0A(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName             string
		opcode               uint16
		registerX            uint8
		registerValue        uint8
		key                  uint8
		keyValue             uint8
		programCounterBefore uint16
		programCounterAfter  uint16
	}{

		{
			"Pressed key 0xF the value should be stored in register 0xA",
			0xFA0A,
			0xA,
			0xF,
			0xF,
			0x1,
			0x6C0,
			0x6C0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.keyPad[tt.key] = tt.keyValue
			c8.programCounter = tt.programCounterBefore

			c8.opFX0A()

			if c8.programCounter != tt.programCounterAfter {
				t.Errorf("Expected the program counter to be %#X but got %#X.", tt.programCounterAfter, c8.programCounter)
			}

			if c8.registers[tt.registerX] != tt.key {
				t.Errorf("Expected the register %#X to be %#X but got %#X.", tt.registerX, tt.key, c8.registers[tt.registerX])
			}
		})
	}
}

func TestOPFX15(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerValueX   uint8
		delayTimerBefore uint8
		delayTimerAfter  uint8
	}{
		{
			"Set the delay timer to register 0x4 with value 0x0F",
			0xF415,
			0x4,
			0x0F,
			0xAA,
			0x0F,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.delayTimer = tt.delayTimerBefore
			c8.registers[tt.registerX] = tt.registerValueX

			c8.opFX15()

			if c8.delayTimer != tt.delayTimerAfter {
				t.Errorf("Expected the delay timer to be %#X but got %#X.", tt.delayTimerAfter, c8.delayTimer)
			}
		})
	}
}

func TestOPFX18(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName         string
		opcode           uint16
		registerX        uint8
		registerValueX   uint8
		soundTimerBefore uint8
		soundTimerAfter  uint8
	}{
		{
			"Set the sound timer to register 0x9 with value 0xA0",
			0xF918,
			0x9,
			0xA0,
			0xAA,
			0xA0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.soundTimer = tt.soundTimerBefore
			c8.registers[tt.registerX] = tt.registerValueX

			c8.opFX18()

			if c8.soundTimer != tt.soundTimerAfter {
				t.Errorf("Expected the sound timer to be %#X but got %#X.", tt.soundTimerAfter, c8.soundTimer)
			}
		})
	}
}

func TestOPFX1E(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName            string
		opcode              uint16
		registerX           uint8
		registerValueX      uint8
		indexRegisterBefore uint16
		indexRegisterAfter  uint16
	}{
		{
			"Add register 0x2 to index register",
			0xF21E,
			0x2,
			0x02,
			0xAA,
			0xAC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.indexRegister = tt.indexRegisterBefore
			c8.registers[tt.registerX] = tt.registerValueX

			c8.opFX1E()

			if c8.indexRegister != tt.indexRegisterAfter {
				t.Errorf("Expected the index register to be %#X but got %#X.", tt.indexRegisterAfter, c8.indexRegister)
			}
		})
	}
}

func TestOPFX29(t *testing.T) {
	c8 := Chip8{}

	tests := []struct {
		testName            string
		opcode              uint16
		registerX           uint8
		registerValueX      uint8
		indexRegisterBefore uint16
		indexRegisterAfter  uint16
	}{
		{
			"Sets the index register to the location of the sprite for digit that is stored in 0xB",
			0xFB29,
			0xB,
			0x9,
			0xAA,
			0x7D,
		},
		{
			"Sets the index register to the location of the sprite for digit that is stored in 0x0",
			0xF029,
			0x0,
			0xF,
			0x3C,
			0x9B,
		},
		{
			"Sets the index register to the location of the sprite for digit that is stored in 0x0",
			0xF829,
			0x8,
			0x3,
			0xD9,
			0x5F,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c8.opcode = tt.opcode
			c8.indexRegister = tt.indexRegisterBefore
			c8.registers[tt.registerX] = tt.registerValueX

			c8.opFX29()
			fmt.Println(tt.registerValueX)
			fmt.Println(c8.indexRegister)
			if c8.indexRegister != tt.indexRegisterAfter {
				t.Errorf("Expected the index register to be %#X but got %#X.", tt.indexRegisterAfter, c8.indexRegister)
			}
		})
	}
}
func TestOpFX33(t *testing.T) {
	type fields struct {
		registers     [16]uint8
		memory        [4096]uint8
		indexRegister uint16
		opcode        uint16
	}
	tests := []struct {
		name           string
		fields         fields
		vx             uint8
		expectedMemory [3]uint8 // [hundreds, tens, ones]
	}{
		{
			name: "BCD of 123 at I = 100",
			fields: fields{
				registers:     [16]uint8{123},
				indexRegister: 100,
				opcode:        0xF033, // VX = 0
			},
			vx:             0,
			expectedMemory: [3]uint8{1, 2, 3},
		},
		{
			name: "BCD of 0 at I = 200",
			fields: fields{
				registers:     [16]uint8{0, 0},
				indexRegister: 200,
				opcode:        0xF133, // VX = 1
			},
			vx:             1,
			expectedMemory: [3]uint8{0, 0, 0},
		},
		{
			name: "BCD of 255 at I = 300",
			fields: fields{
				registers:     [16]uint8{0, 255},
				indexRegister: 300,
				opcode:        0xF133, // VX = 1
			},
			vx:             1,
			expectedMemory: [3]uint8{2, 5, 5},
		},
		{
			name: "BCD of 42 at I = 0",
			fields: fields{
				registers:     [16]uint8{0, 0, 42},
				indexRegister: 0,
				opcode:        0xF233, // VX = 2
			},
			vx:             2,
			expectedMemory: [3]uint8{0, 4, 2},
		},
		{
			name: "BCD of 99 at I = 4093 (near end of memory)",
			fields: fields{
				registers:     [16]uint8{99},
				indexRegister: 4093,
				opcode:        0xF033, // VX = 0
			},
			vx:             0,
			expectedMemory: [3]uint8{0, 9, 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c8 := &Chip8{
				registers:     tt.fields.registers,
				memory:        tt.fields.memory,
				indexRegister: tt.fields.indexRegister,
				opcode:        tt.fields.opcode | (uint16(tt.vx) << 8),
			}
			c8.opFX33()

			base := c8.indexRegister
			got := [3]uint8{
				c8.memory[base],
				c8.memory[base+1],
				c8.memory[base+2],
			}
			if got != tt.expectedMemory {
				t.Errorf("BCD memory = %v, want %v", got, tt.expectedMemory)
			}
		})
	}
}
