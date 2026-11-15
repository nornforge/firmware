package channel_test

import (
	"testing"

	"github.com/nornforge/firmware/channel"
)

// fakePin records every level written to it.
type fakePin struct {
	levels []bool
}

func (p *fakePin) Set(high bool) {
	p.levels = append(p.levels, high)
}

func (p *fakePin) last(t *testing.T) bool {
	t.Helper()
	if len(p.levels) == 0 {
		t.Fatal("the pin was never written")
	}
	return p.levels[len(p.levels)-1]
}

func TestNewIsOffAndDoesNotTouchThePin(t *testing.T) {
	pin := &fakePin{}
	ch := channel.New(pin, 3)
	if ch.Get() {
		t.Error("a new channel must report off")
	}
	if ch.Index() != 3 {
		t.Errorf("Index() = %d, want 3", ch.Index())
	}
	if len(pin.levels) != 0 {
		t.Errorf("New wrote %v to the pin, want no writes", pin.levels)
	}
}

func TestSetDrivesThePinAndIsReportedByGet(t *testing.T) {
	pin := &fakePin{}
	ch := channel.New(pin, 1)

	for _, state := range []bool{true, true, false, true, false} {
		ch.Set(state)
		if got := pin.last(t); got != state {
			t.Errorf("after Set(%v) the pin is %v", state, got)
		}
		if got := ch.Get(); got != state {
			t.Errorf("after Set(%v) Get() = %v", state, got)
		}
	}
}

func TestToggleInvertsTheState(t *testing.T) {
	pin := &fakePin{}
	ch := channel.New(pin, 1)

	for _, want := range []bool{true, false, true} {
		ch.Toggle()
		if got := ch.Get(); got != want {
			t.Errorf("Get() = %v, want %v", got, want)
		}
		if got := pin.last(t); got != want {
			t.Errorf("pin is %v, want %v", got, want)
		}
	}
}
