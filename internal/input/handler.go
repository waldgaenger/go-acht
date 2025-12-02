package input

// InputHandler abstracts the input mechanism for the CHIP-8 emulator.
// Implementations should update the provided keyPad array to reflect the current
// state of each key (pressed or not). PollKeys returns true if a quit event was detected.
type InputHandler interface {
	PollKeys(keyPad *[16]bool) (quit bool)
}
