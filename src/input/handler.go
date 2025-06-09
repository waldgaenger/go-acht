package input

type InputHandler interface {
	PollKeys() (keyPad [16]uint8, quit bool)
}
