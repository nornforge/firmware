package main

import (
	"bufio"
	"fmt"
	"machine"
	"os"

	"github.com/nornforge/firmware/channel"
	"github.com/nornforge/norn/pkg/norn"
)

const version = "v0.0.1"

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

func getChannelByIndex(index uint, channels []*channel.Channel) (*channel.Channel, error) {
	const minChannel = 1
	var maxChannel uint = uint(len(channels))
	if index > maxChannel || index < minChannel {
		return nil, fmt.Errorf("Invalid index provided: %d. Channel must be in range [%d,%d]",
			index,
			minChannel,
			maxChannel,
		)
	}
	return channels[index-1], nil
}

func main() {
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})
	led.Set(true)
	commands := make(chan norn.Command)
	go readFromSerial(commands) // make it parallel
	for {
		command := <-commands
		switch command.Type {
		case norn.Get:
			ch, err := getChannelByIndex(command.Channel, Channels)
			if err != nil {
				fmt.Print(norn.MarshalError(err))
			}
			res := norn.Response{
				Success: true,
				Channel: ch.Index(),
				Status:  ch.Get(),
			}
			fmt.Print(string(res.Marshal()))
		case norn.Set:
			ch, err := getChannelByIndex(command.Channel, Channels)
			if err != nil {
				fmt.Print(norn.MarshalError(err))
				continue
			}
			ch.Set(command.Status)
			res := norn.Response{
				Success: true,
				Channel: ch.Index(),
				Status:  ch.Get(),
			}
			fmt.Print(string(res.Marshal()))
		case norn.Version:
			res := norn.Response{
				Success: true,
				Message: fmt.Sprintf("%s", version),
			}
			fmt.Print(string(res.Marshal()))
		case norn.Bootloader:
			enterBootloader()
		case norn.MaxChannels:
			res := norn.Response{
				Success:    true,
				MaxChannel: len(Channels),
			}
			fmt.Print(string(res.Marshal()))
		}
	}
}
