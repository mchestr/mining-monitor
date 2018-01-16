package ethos_monitor

type Client interface {
	IP() string
	Stats() (*Statistics, error)
	Reboot() error
	Restart() error
}

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
}
