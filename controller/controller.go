// Package controller executes norn commands on a set of relay channels.
// It does not depend on the machine package, so it can be tested without hardware.
package controller

import (
	"fmt"

	"github.com/nornforge/firmware/channel"
	"github.com/nornforge/norn/pkg/norn"
)

// Brightness is the type of the command that sets the brightness of the
// status LED. The brightness (0 to 255, 0 turns the LED off) is passed in the
// channel field of the command and echoed in the channel field of the response.
//
// The norn package does not define this command yet.
const Brightness norn.CommandType = 6

// DefaultVersion is reported when no version was injected at build time.
const DefaultVersion = "v0.0.0"

// Controller executes commands on its channels.
type Controller struct {
	channels        []*channel.Channel
	version         string
	enterBootloader func()
	setBrightness   func(brightness uint8)
}

// New returns a Controller for the given channels. The channel at position i
// is addressed with index i+1. An empty version is replaced by DefaultVersion.
// enterBootloader is called when a bootloader command is received, after all
// channels have been switched off. setBrightness is called when a brightness
// command with a valid value is received.
func New(channels []*channel.Channel, version string, enterBootloader func(), setBrightness func(brightness uint8)) *Controller {
	if version == "" {
		version = DefaultVersion
	}
	return &Controller{
		channels:        channels,
		version:         version,
		enterBootloader: enterBootloader,
		setBrightness:   setBrightness,
	}
}

// Handle executes the command. It returns the response to send to the host
// and whether there is a response to send at all.
func (c *Controller) Handle(command norn.Command) (norn.Response, bool) {
	switch command.Type {
	case norn.Get:
		ch, err := c.channel(command.Channel)
		if err != nil {
			return failure(err), true
		}
		return norn.Response{Success: true, Channel: ch.Index(), Status: ch.Get()}, true
	case norn.Set:
		ch, err := c.channel(command.Channel)
		if err != nil {
			return failure(err), true
		}
		ch.Set(command.Status)
		return norn.Response{Success: true, Channel: ch.Index(), Status: ch.Get()}, true
	case norn.Version:
		return norn.Response{Success: true, Message: c.version}, true
	case norn.Bootloader:
		// Nothing controls the channels while the bootloader is running.
		c.AllOff()
		c.enterBootloader()
		return norn.Response{}, false
	case norn.MaxChannels:
		return norn.Response{Success: true, MaxChannel: len(c.channels)}, true
	case Brightness:
		const maxBrightness = 255
		if command.Channel > maxBrightness {
			return failure(fmt.Errorf("Invalid brightness provided: %d. Brightness must be in range [0,%d]",
				command.Channel,
				maxBrightness,
			)), true
		}
		c.setBrightness(uint8(command.Channel))
		return norn.Response{Success: true, Channel: command.Channel}, true
	}
	return failure(fmt.Errorf("Unknown command type: %d", command.Type)), true
}

// AllOff switches every channel off.
func (c *Controller) AllOff() {
	for _, ch := range c.channels {
		ch.Set(false)
	}
}

func (c *Controller) channel(index uint) (*channel.Channel, error) {
	const minChannel = 1
	maxChannel := uint(len(c.channels))
	if index > maxChannel || index < minChannel {
		return nil, fmt.Errorf("Invalid index provided: %d. Channel must be in range [%d,%d]",
			index,
			minChannel,
			maxChannel,
		)
	}
	return c.channels[index-1], nil
}

func failure(err error) norn.Response {
	return norn.Response{Success: false, Message: err.Error()}
}
