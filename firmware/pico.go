//go:build pico

package main

import (
	"machine"

	"github.com/nornforge/firmware/channel"
)

var (
	led      = machine.LED
	Channels = []*channel.Channel{
		channel.New(machine.GP21, 1),
		channel.New(machine.GP20, 2),
		channel.New(machine.GP19, 3),
		channel.New(machine.GP18, 4),
		channel.New(machine.GP17, 5),
		channel.New(machine.GP16, 6),
		channel.New(machine.GP15, 7),
		channel.New(machine.GP14, 8),
	}
)

func enterBootloader() {
	machine.EnterBootloader()
}
