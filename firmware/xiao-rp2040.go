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
	led      = machine.LED
	Channels = []*channel.Channel{
		channel.New(machine.D10, 1),
		channel.New(machine.D9, 2),
		channel.New(machine.D8, 3),
		channel.New(machine.D7, 4),
	}
)
