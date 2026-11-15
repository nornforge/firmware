package status_test

import (
	"image/color"
	"io"
	"testing"
	"time"

	"github.com/nornforge/firmware/status"
)

type fakeLED struct {
	writes []color.RGBA
	err    error
}

func (f *fakeLED) WriteColors(colors []color.RGBA) error {
	f.writes = append(f.writes, colors...)
	return f.err
}

func (f *fakeLED) last() color.RGBA {
	return f.writes[len(f.writes)-1]
}

var start = time.Unix(1000, 0)

// shown returns the color the LED shows for the state at the default brightness.
func shown(s status.State) color.RGBA {
	return status.Scale(s.Color(), status.DefaultBrightness)
}

func TestStatesHaveDistinctColors(t *testing.T) {
	seen := map[color.RGBA]status.State{}
	for _, s := range []status.State{status.Idle, status.Busy, status.Failed, status.Updating} {
		c := s.Color()
		if c == (color.RGBA{}) {
			t.Errorf("state %d is dark", s)
		}
		if other, ok := seen[c]; ok {
			t.Errorf("states %d and %d share the color %v", other, s, c)
		}
		seen[c] = s
	}
}

func TestANewLightShowsIdle(t *testing.T) {
	led := &fakeLED{}
	light := status.NewLight(led, start)
	if light.State() != status.Idle || led.last() != shown(status.Idle) {
		t.Errorf("state %d, color %v", light.State(), led.last())
	}
}

func TestTransientStatesReturnToIdleAfterTheirHoldTime(t *testing.T) {
	for _, s := range []status.State{status.Busy, status.Failed} {
		led := &fakeLED{}
		light := status.NewLight(led, start)

		light.Show(s, start)
		if led.last() != shown(s) {
			t.Errorf("state %d: color %v, want %v", s, led.last(), shown(s))
		}

		light.Tick(start.Add(s.Hold() - time.Millisecond))
		if light.State() != s {
			t.Errorf("state %d: left before the hold time was over", s)
		}

		light.Tick(start.Add(s.Hold()))
		if light.State() != status.Idle || led.last() != shown(status.Idle) {
			t.Errorf("state %d: did not return to idle", s)
		}
	}
}

func TestPermanentStatesStay(t *testing.T) {
	led := &fakeLED{}
	light := status.NewLight(led, start)

	light.Show(status.Updating, start)
	light.Tick(start.Add(time.Hour))

	if light.State() != status.Updating || led.last() != shown(status.Updating) {
		t.Errorf("state %d, color %v", light.State(), led.last())
	}
}

func TestANewStateRestartsTheHoldTime(t *testing.T) {
	led := &fakeLED{}
	light := status.NewLight(led, start)

	light.Show(status.Failed, start)
	// A successful command shortly before the error would have expired
	later := start.Add(status.Failed.Hold() - time.Millisecond)
	light.Show(status.Busy, later)

	light.Tick(start.Add(status.Failed.Hold()))
	if light.State() != status.Busy {
		t.Error("the hold time of the replaced state ended the new one")
	}
	light.Tick(later.Add(status.Busy.Hold()))
	if light.State() != status.Idle {
		t.Error("did not return to idle")
	}
}

func TestTickDoesNotRewriteAnUnchangedLED(t *testing.T) {
	led := &fakeLED{}
	light := status.NewLight(led, start)
	writes := len(led.writes)

	for i := 1; i <= 10; i++ {
		light.Tick(start.Add(time.Duration(i) * time.Second))
	}
	if len(led.writes) != writes {
		t.Errorf("the LED was written %d times while idle", len(led.writes)-writes)
	}
}

func TestScale(t *testing.T) {
	full := color.RGBA{R: 255, G: 255, B: 255}
	for brightness, want := range map[uint8]uint8{0: 0, 1: 1, 10: 10, 128: 128, 255: 255} {
		got := status.Scale(full, brightness)
		if got != (color.RGBA{R: want, G: want, B: want}) {
			t.Errorf("brightness %d: %v, want components of %d", brightness, got, want)
		}
	}
	if got := status.Scale(color.RGBA{R: 255}, 40); got != (color.RGBA{R: 40}) {
		t.Errorf("a dark component must stay dark: %v", got)
	}
}

func TestStatesStayDistinctAtTheLowestBrightness(t *testing.T) {
	seen := map[color.RGBA]bool{}
	for _, s := range []status.State{status.Idle, status.Busy, status.Failed, status.Updating} {
		c := status.Scale(s.Color(), 1)
		if c == (color.RGBA{}) || seen[c] {
			t.Errorf("state %d is dark or not distinct at brightness 1: %v", s, c)
		}
		seen[c] = true
	}
}

func TestSetBrightnessRedrawsWithoutChangingStateOrHoldTime(t *testing.T) {
	led := &fakeLED{}
	light := status.NewLight(led, start)
	light.Show(status.Failed, start)

	light.SetBrightness(100)

	if light.Brightness() != 100 || light.State() != status.Failed {
		t.Errorf("brightness %d, state %d", light.Brightness(), light.State())
	}
	if want := status.Scale(status.Failed.Color(), 100); led.last() != want {
		t.Errorf("color %v, want %v", led.last(), want)
	}

	light.Tick(start.Add(status.Failed.Hold()))
	if want := status.Scale(status.Idle.Color(), 100); light.State() != status.Idle || led.last() != want {
		t.Errorf("after the hold time: state %d, color %v, want idle in %v", light.State(), led.last(), want)
	}
}

func TestBrightnessZeroTurnsTheLEDOff(t *testing.T) {
	led := &fakeLED{}
	light := status.NewLight(led, start)

	light.SetBrightness(0)

	if led.last() != (color.RGBA{}) {
		t.Errorf("color %v, want dark", led.last())
	}
}

func TestWriteErrorsAreIgnored(t *testing.T) {
	led := &fakeLED{err: io.ErrClosedPipe}
	light := status.NewLight(led, start)

	light.Show(status.Busy, start)
	light.SetBrightness(100)
	light.Tick(start.Add(status.Busy.Hold()))

	if light.State() != status.Idle {
		t.Errorf("state %d, want idle despite a write error", light.State())
	}
	if len(led.writes) != 4 {
		t.Errorf("writes %d, want 4 writes even when the driver reports an error", len(led.writes))
	}
}
