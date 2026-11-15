// Package channel models a single relay output.
package channel

// Pin is the part of machine.Pin a Channel needs. Depending on this interface
// instead of the machine package allows the code to be tested without hardware.
type Pin interface {
	Set(high bool)
}

// Channel is a relay output which remembers the state it was switched to.
type Channel struct {
	pin   Pin
	state bool
	index uint
}

// New returns the channel with the given one-based index driving pin.
// The pin has to be configured as an output by the caller.
func New(pin Pin, index uint) *Channel {
	return &Channel{pin: pin, state: false, index: index}
}

// Toggle inverts the state of the channel.
func (c *Channel) Toggle() {
	c.Set(!c.state)
}

// Set switches the channel on (true) or off (false).
func (c *Channel) Set(state bool) {
	c.state = state
	c.pin.Set(c.state)
}

// Get returns the state the channel was last switched to.
func (c *Channel) Get() bool {
	return c.state
}

// Index returns the one-based index of the channel.
func (c *Channel) Index() uint {
	return c.index
}
