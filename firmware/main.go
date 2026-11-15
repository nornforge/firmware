package main

import (
	"bufio"
	"fmt"
	"machine"
	"os"

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
	for {
		command := <-commands
		if res, ok := ctrl.Handle(command); ok {
			fmt.Print(string(res.Marshal()))
		}
	}
}
