//go:build xiao_rp2040

/*
 * The https://wiki.seeedstudio.com/XIAO-RP2040/ is not PIN compatible with the Pico
 * Do not use this in the WaveShare Pico-Relay-B baseboard!
 */

package main

import (
	"machine"

	"github.com/nornforge/firmware/channel"
)

var (
	led = machine.LED
	// The on-board RGB LED. It is not powered here, so it stays dark.
	rgbPin   = machine.NEOPIXEL
	Channels = []*channel.Channel{
		output(machine.D10, 1),
		output(machine.D9, 2),
		output(machine.D8, 3),
		output(machine.D7, 4),
	}
)
