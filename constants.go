package miningmonitor

const (
	// POWERCYCLING state of the monitor, something went wrong and the monitor will attempt to power cycle the client
	POWERCYCLING = iota
	// RUNNING state of the monitor, this state will check the stats to determine if any actions are required
	RUNNING
	// REBOOTING state of the monitor, something went wrong and the montiro will attempt to restart the client
	REBOOTING
	// STOPPED state of the monitor, no longer checking on the clients state
	STOPPED
)
