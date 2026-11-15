package main

import (
	"bufio"
	"fmt"
	"machine"
	"os"
	"time"

	"github.com/nornforge/firmware/channel"
	"github.com/nornforge/firmware/controller"
	"github.com/nornforge/firmware/status"
	"github.com/nornforge/norn/pkg/norn"
	"tinygo.org/x/drivers/ws2812"
)

// tickInterval is how often an idle command loop updates the status LED and
// feeds the watchdog.
const tickInterval = 50 * time.Millisecond

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
// The level is set before the pin becomes an output, so the relay does not see a pulse.
func output(pin machine.Pin, index uint) *channel.Channel {
	pin.Low()
	pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	return channel.New(pin, index)
}

func main() {
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})
	led.Set(true)

	rgb := ws2812.NewWS2812(rgbPin)
	light := status.NewLight(&rgb, time.Now())

	ctrl := controller.New(Channels, Version, func() {
		// The LED keeps this color while the bootloader is running.
		light.Show(status.Updating, time.Now())
		enterBootloader()
	}, light.SetBrightness)
	commands := make(chan norn.Command)
	go readFromSerial(commands) // make it parallel

	// The watchdog is fed from the command loop: if the loop stops running,
	// the firmware restarts instead of staying unresponsive.
	startWatchdog()
	idle := time.NewTicker(tickInterval)
	for {
		select {
		case command := <-commands:
			if res, ok := ctrl.Handle(command); ok {
				fmt.Print(string(res.Marshal()))
				if res.Success {
					light.Show(status.Busy, time.Now())
				} else {
					light.Show(status.Failed, time.Now())
				}
			}
		case <-idle.C:
		}
		light.Tick(time.Now())
		feedWatchdog()
	}
}
