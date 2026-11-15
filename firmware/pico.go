//go:build pico

package main

import (
	"machine"

	"github.com/nornforge/firmware/channel"
)

var (
	led      = machine.LED
	Channels = []*channel.Channel{
		output(machine.GP21, 1),
		output(machine.GP20, 2),
		output(machine.GP19, 3),
		output(machine.GP18, 4),
		output(machine.GP17, 5),
		output(machine.GP16, 6),
		output(machine.GP15, 7),
		output(machine.GP14, 8),
	}
)
