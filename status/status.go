// Package status defines what the RGB status LED shows. It does not depend on
// the machine package, so it can be tested without hardware.
package status

import (
	"image/color"
	"time"
)

// State is a condition of the firmware that is shown on the status LED.
type State uint8

const (
	// Idle: the firmware is running and waiting for commands.
	Idle State = iota
	// Busy: a command was executed successfully.
	Busy
	// Failed: a command was answered with an error.
	Failed
	// Updating: the bootloader is about to be entered.
	Updating
)

// DefaultBrightness is the brightness of a new Light. The LED is very bright
// at full scale, so the default is a small fraction of it.
const DefaultBrightness = 16

// Color returns the color of the state at full brightness.
func (s State) Color() color.RGBA {
	switch s {
	case Idle:
		return color.RGBA{G: 255}
	case Busy:
		return color.RGBA{B: 255}
	case Failed:
		return color.RGBA{R: 255}
	case Updating:
		return color.RGBA{R: 255, B: 255}
	}
	return color.RGBA{}
}

// Scale returns the color dimmed to the brightness, where 255 is full scale
// and 0 is dark.
func Scale(c color.RGBA, brightness uint8) color.RGBA {
	scale := func(v uint8) uint8 { return uint8(uint16(v) * uint16(brightness) / 255) }
	return color.RGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B)}
}

// Hold returns how long the state is shown before the LED returns to Idle.
// Zero means the state stays until another one replaces it.
func (s State) Hold() time.Duration {
	switch s {
	case Busy:
		return 150 * time.Millisecond
	case Failed:
		return time.Second
	}
	return 0
}

// Writer is the part of an LED driver a Light needs.
type Writer interface {
	WriteColors(colors []color.RGBA) error
}

// Light shows states on a single RGB LED and returns to Idle after a
// transient state was shown for its hold time.
type Light struct {
	led        Writer
	state      State
	brightness uint8
	revertAt   time.Time
}

// NewLight returns a Light showing Idle at DefaultBrightness.
func NewLight(led Writer, now time.Time) *Light {
	l := &Light{led: led, brightness: DefaultBrightness}
	l.Show(Idle, now)
	return l
}

// Show displays the state.
func (l *Light) Show(state State, now time.Time) {
	l.state = state
	l.revertAt = time.Time{}
	if hold := state.Hold(); hold > 0 {
		l.revertAt = now.Add(hold)
	}
	l.write()
}

// SetBrightness changes the brightness, where 255 is full scale and 0 turns
// the LED off. The state shown and its hold time are not affected.
func (l *Light) SetBrightness(brightness uint8) {
	l.brightness = brightness
	l.write()
}

// Brightness returns the current brightness.
func (l *Light) Brightness() uint8 {
	return l.brightness
}

func (l *Light) write() {
	// LED write failures are best-effort only: they do not change the firmware
	// state, and the status package cannot report them to the caller.
	_ = l.led.WriteColors([]color.RGBA{Scale(l.state.Color(), l.brightness)})
}

// Tick has to be called regularly. It returns to Idle once the hold time of
// a transient state is over.
func (l *Light) Tick(now time.Time) {
	if !l.revertAt.IsZero() && !now.Before(l.revertAt) {
		l.Show(Idle, now)
	}
}

// State returns the state currently shown.
func (l *Light) State() State {
	return l.state
}
