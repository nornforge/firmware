package channel

import (
	"machine"
)

type Channel struct {
	pin   machine.Pin
	state bool
	index uint
}

func New(pin machine.Pin, index uint) *Channel {
	asOutput := machine.PinConfig{
		Mode: machine.PinOutput,
	}
	ch := Channel{pin: pin, state: false, index: index}
	ch.configure(asOutput)
	return &ch
}

func (c *Channel) Toggle() {
	c.state = !c.state
	c.pin.Set(c.state)
}

func (c *Channel) Set(state bool) {
	c.state = state
	c.pin.Set(c.state)
}

func (c *Channel) Get() bool {
	return c.state
}

func (c *Channel) Index() uint {
	return c.index
}

func (c *Channel) configure(config machine.PinConfig) {
	c.pin.Configure(config)
}
