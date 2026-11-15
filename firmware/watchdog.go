//go:build rp2040

package main

import (
	"device/rp"
	"machine"
	"time"
)

const (
	// watchdogTimeout is the time after which the firmware is restarted if the
	// command loop stops running, for example after a panic.
	watchdogTimeout = 2 * time.Second
	// watchdogInterval is how often an idle command loop feeds the watchdog.
	watchdogInterval = watchdogTimeout / 4
)

func startWatchdog() {
	machine.Watchdog.Configure(machine.WatchdogConfig{
		TimeoutMillis: uint32(watchdogTimeout / time.Millisecond),
	})
	machine.Watchdog.Start()
}

func feedWatchdog() {
	machine.Watchdog.Update()
}

// stopWatchdog disables the watchdog. The machine package offers no way to do
// that, so the register is written directly.
func stopWatchdog() {
	rp.WATCHDOG.CTRL.ClearBits(rp.WATCHDOG_CTRL_ENABLE)
}

func enterBootloader() {
	// The bootloader does not feed the watchdog.
	stopWatchdog()
	machine.EnterBootloader()
}
