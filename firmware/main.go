package main

import (
	"bufio"
	"fmt"
	"machine"
	"os"
	"time"

	"github.com/nornforge/firmware/channel"
	"github.com/nornforge/firmware/controller"
	"github.com/nornforge/norn/pkg/norn"
)

// The version to be injected by CI/CD
var Version string // This needs to be empty in order be overwritten by  ldflags

func readFromSerial(ch chan<- norn.Command) {
	reader := bufio.NewReader(os.Stdin)
	command := norn.Command{}
	for {
		err := command.Parse(reader)
		if err != nil {
			fmt.Print(norn.MarshalError(err))
			continue
		}
		ch <- command
	}
}

// output configures the pin as an output and returns it as the channel with the given index.
func output(pin machine.Pin, index uint) *channel.Channel {
	pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	return channel.New(pin, index)
}

func main() {
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})
	led.Set(true)
	ctrl := controller.New(Channels, Version, enterBootloader)
	commands := make(chan norn.Command)
	go readFromSerial(commands) // make it parallel

	// The watchdog is fed from the command loop: if the loop stops running,
	// the firmware restarts instead of staying unresponsive.
	startWatchdog()
	idle := time.NewTicker(watchdogInterval)
	for {
		select {
		case command := <-commands:
			if res, ok := ctrl.Handle(command); ok {
				fmt.Print(string(res.Marshal()))
			}
		case <-idle.C:
		}
		feedWatchdog()
	}
}
