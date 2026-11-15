package controller_test

import (
	"testing"

	"github.com/nornforge/firmware/channel"
	"github.com/nornforge/firmware/controller"
	"github.com/nornforge/norn/pkg/norn"
)

type fakePin struct {
	level  bool
	writes int
}

func (p *fakePin) Set(high bool) {
	p.level = high
	p.writes++
}

type fixture struct {
	ctrl        *controller.Controller
	pins        []*fakePin
	bootloaders int
}

func newFixture(numChannels int, version string) *fixture {
	f := &fixture{}
	channels := make([]*channel.Channel, 0, numChannels)
	for i := 0; i < numChannels; i++ {
		pin := &fakePin{}
		f.pins = append(f.pins, pin)
		channels = append(channels, channel.New(pin, uint(i+1)))
	}
	f.ctrl = controller.New(channels, version, func() { f.bootloaders++ })
	return f
}

// handle executes the command and fails the test if there is no response.
func (f *fixture) handle(t *testing.T, command norn.Command) norn.Response {
	t.Helper()
	res, ok := f.ctrl.Handle(command)
	if !ok {
		t.Fatalf("no response to %+v", command)
	}
	return res
}

func TestSetSwitchesOnlyTheAddressedChannel(t *testing.T) {
	f := newFixture(8, "")

	for index := uint(1); index <= 8; index++ {
		res := f.handle(t, norn.Command{Type: norn.Set, Channel: index, Status: true})
		if !res.Success || res.Channel != index || !res.Status {
			t.Errorf("set %d on: unexpected response %+v", index, res)
		}
		for i, pin := range f.pins {
			if want := uint(i+1) == index; pin.level != want {
				t.Errorf("set %d on: pin of channel %d is %v, want %v", index, i+1, pin.level, want)
			}
		}

		res = f.handle(t, norn.Command{Type: norn.Set, Channel: index, Status: false})
		if !res.Success || res.Channel != index || res.Status {
			t.Errorf("set %d off: unexpected response %+v", index, res)
		}
		if f.pins[index-1].level {
			t.Errorf("set %d off: the pin is still high", index)
		}
	}
}

func TestGetReportsTheStateWithoutWritingThePin(t *testing.T) {
	f := newFixture(8, "")

	res := f.handle(t, norn.Command{Type: norn.Get, Channel: 4})
	if !res.Success || res.Channel != 4 || res.Status {
		t.Errorf("get on a fresh channel: unexpected response %+v", res)
	}

	f.handle(t, norn.Command{Type: norn.Set, Channel: 4, Status: true})
	writes := f.pins[3].writes

	// The status field of a get command must be ignored.
	res = f.handle(t, norn.Command{Type: norn.Get, Channel: 4, Status: false})
	if !res.Success || res.Channel != 4 || !res.Status {
		t.Errorf("get after set on: unexpected response %+v", res)
	}
	if f.pins[3].writes != writes {
		t.Error("get wrote to the pin")
	}
}

func TestInvalidChannelIsRejected(t *testing.T) {
	messages := map[uint]string{
		0:   "Invalid index provided: 0. Channel must be in range [1,8]",
		9:   "Invalid index provided: 9. Channel must be in range [1,8]",
		255: "Invalid index provided: 255. Channel must be in range [1,8]",
	}

	for _, commandType := range []norn.CommandType{norn.Get, norn.Set} {
		for index, message := range messages {
			f := newFixture(8, "")
			res := f.handle(t, norn.Command{Type: commandType, Channel: index, Status: true})
			if res.Success {
				t.Errorf("type %d, channel %d: reported success", commandType, index)
			}
			if res.Message != message {
				t.Errorf("type %d, channel %d: message %q, want %q", commandType, index, res.Message, message)
			}
			for i, pin := range f.pins {
				if pin.writes != 0 {
					t.Errorf("type %d, channel %d: pin of channel %d was written", commandType, index, i+1)
				}
			}
		}
	}
}

func TestTheControllerKeepsWorkingAfterAnError(t *testing.T) {
	f := newFixture(8, "v1.2.3")

	f.handle(t, norn.Command{Type: norn.Get, Channel: 9})
	res := f.handle(t, norn.Command{Type: norn.Version})
	if !res.Success || res.Message != "v1.2.3" {
		t.Errorf("version after an error: unexpected response %+v", res)
	}
}

func TestVersion(t *testing.T) {
	for version, want := range map[string]string{
		"v1.2.3": "v1.2.3",
		"":       controller.DefaultVersion,
	} {
		f := newFixture(8, version)
		res := f.handle(t, norn.Command{Type: norn.Version})
		if !res.Success || res.Message != want {
			t.Errorf("version %q: unexpected response %+v, want message %q", version, res, want)
		}
	}
}

func TestMaxChannels(t *testing.T) {
	for _, numChannels := range []int{4, 8} {
		f := newFixture(numChannels, "")
		res := f.handle(t, norn.Command{Type: norn.MaxChannels})
		if !res.Success || res.MaxChannel != numChannels {
			t.Errorf("%d channels: unexpected response %+v", numChannels, res)
		}
	}
}

func TestBootloaderIsEnteredWithoutResponse(t *testing.T) {
	f := newFixture(8, "")

	if res, ok := f.ctrl.Handle(norn.Command{Type: norn.Bootloader}); ok {
		t.Errorf("unexpected response %+v", res)
	}
	if f.bootloaders != 1 {
		t.Errorf("the bootloader was entered %d times, want once", f.bootloaders)
	}
}

func TestOtherCommandsDoNotEnterTheBootloader(t *testing.T) {
	f := newFixture(8, "")

	for _, commandType := range []norn.CommandType{norn.Get, norn.Set, norn.Version, norn.MaxChannels} {
		f.ctrl.Handle(norn.Command{Type: commandType, Channel: 1})
	}
	if f.bootloaders != 0 {
		t.Errorf("the bootloader was entered %d times, want never", f.bootloaders)
	}
}
