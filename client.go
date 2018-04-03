package miningmonitor

// Client interface to the mining software
type Client interface {
	// IP of the client
	IP() string
	// Stats returns information about the current state used to check thresholds
	Stats() (*Statistics, error)
	// Reboot the client if it is available and enabled
	Reboot() error
	// Restart the client if it is available and enabled
	Restart() error

	// PowerCycleEnabled bool to indicate if this client can be power cycled
	PowerCycleEnabled() bool
	// PowerCycle the client using an external API enabled power plug.
	PowerCycle() error

	// SetReadOnly to disable changing the state of the client
	SetReadOnly(readOnly, failOnWrites bool)
	// ReadOnly indicates if this client is readonly
	ReadOnly() bool
}

// Statistics of a client, used for determining thresholds.
type Statistics struct {
	Version         string
	RunningTime     int
	GpuTemperatures []float64
	GpuFanPercents  []float64

	MainMiningPool        string
	MainHashRate          float64
	MainShares            int
	MainRejectedShares    int
	MainGpuHashRate       []float64
	MainGpuShares         []int
	MainGpuRejectedShares []int
	MainGpuInvalidShares  []int
	MainPoolSwitches      int
	MainInvalidShares     int

	AltMiningPool        string
	AltHashRate          float64
	AltShares            int
	AltRejectedShares    int
	AltGpuHashRate       []float64
	AltGpuShares         []int
	AltGpuRejectedShares []int
	AltGpuInvalidShares  []int
	AltPoolSwitches      int
	AltInvalidShares     int

	PowerState *PowerState
}
