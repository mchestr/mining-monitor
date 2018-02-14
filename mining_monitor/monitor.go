package mining_monitor

import (
	"fmt"
	"time"

	"github.com/golang/glog"
)

type ClientMonitorConfig struct {
	Thresholds                  []*Threshold
	CheckFailsBeforeReboot      int
	RebootFailsBeforePowerCycle int
	RebootInterval              time.Duration
	StatsInterval               time.Duration
	StateInterval               time.Duration
	PowerCycleOnly              bool
}

func NewClientMonitorConfig(thresholds []*Threshold, checkFailsBeforeReboot, rebootFailsBeforePowerCycle int,
	rebootInterval, statsInterval, stateInterval time.Duration, powerCycleOnly bool) *ClientMonitorConfig {
	return &ClientMonitorConfig{
		Thresholds:                  thresholds,
		CheckFailsBeforeReboot:      checkFailsBeforeReboot,
		RebootFailsBeforePowerCycle: rebootFailsBeforePowerCycle,
		RebootInterval:              rebootInterval,
		StatsInterval:               statsInterval,
		StateInterval:               stateInterval,
		PowerCycleOnly:              powerCycleOnly,
	}
}

type ClientMonitoring struct {
	C      Client
	Config *ClientMonitorConfig
}

type Monitor struct {
	c            []ClientMonitoring
	EventService *EventService

	stop     chan bool
	interval time.Duration
	state    int
}

func NewMonitor(eventService *EventService) *Monitor {
	return &Monitor{
		c:            []ClientMonitoring{},
		EventService: eventService,
	}
}

func (m *Monitor) AddClient(c Client, config *ClientMonitorConfig) {
	m.c = append(m.c, ClientMonitoring{C: c, Config: config})
}

func (m *Monitor) Start() error {
	if m.state == RUNNING {
		return fmt.Errorf("monitor already running")
	}
	m.stop = make(chan bool, len(m.c))
	m.state = RUNNING
	for _, c := range m.c {
		m.EventService.E <- NewLogEvent(c.C, "starting monitoring...")
		go m.monitorClient(m.stop, c.C, c.Config)
	}
	go m.EventService.Start()
	return nil
}

func (m *Monitor) Stop() error {
	if m.state == STOPPED {
		return fmt.Errorf("monitor already stopped")
	}
	for i := 0; i < len(m.c); i++ {
		m.stop <- true
	}
	m.EventService.Stop()
	m.state = STOPPED
	close(m.stop)
	return nil
}

func (m *Monitor) monitorClient(stop chan bool, c Client, config *ClientMonitorConfig) {
	m.EventService.E <- NewLogEvent(c,
		fmt.Sprintf("Monitor Starting on %s\nPower Cycle Only: %t\nThresholds: %s\nPowerCycle: %t\nReadOnly: %t\nCheckFailsBeforeReboot: %d\nRebootFailsBeforePowercycle: %d\nRebootInterval: %v\nStatsInterval: %v\nStateInterval: %v",
			c.IP(), config.PowerCycleOnly, config.Thresholds, c.PowerCycleEnabled(), c.ReadOnly(), config.CheckFailsBeforeReboot, config.RebootFailsBeforePowerCycle, config.RebootInterval, config.StatsInterval, config.StateInterval),
	)
	stateTicker := time.NewTicker(config.StateInterval)
	statsTicker := time.NewTicker(config.StatsInterval)

	failedReboots := 0
	failedChecks := 0
	lastReboot := time.Now().Add(-config.RebootInterval)
	var errors []error
	reset := false
	state := RUNNING

	for {
		select {
		case <-stateTicker.C:
			glog.V(1).Infof("State: {failedReboots: %d, failedChecks: %d}", failedReboots, failedChecks)
			if reset {
				failedReboots = 0
				failedChecks = 0
				errors = []error{}
				reset = false
			}
			// If client has power cycling enabled and number of failed reboots is greater than threshold OR power cycle only enabled and failed checks greater than threshold and last reboot is longer than threshold
			if c.PowerCycleEnabled() && (failedReboots >= config.RebootFailsBeforePowerCycle || config.PowerCycleOnly && failedChecks >= config.CheckFailsBeforeReboot && time.Now().Sub(lastReboot) > config.RebootInterval) {
				if state != POWERCYCLING {
					m.EventService.E <- NewLogEvent(c, "transitioning to POWERCYCLING state...")
				}
				state = POWERCYCLING
			} else if !config.PowerCycleOnly && failedChecks >= config.CheckFailsBeforeReboot && time.Now().Sub(lastReboot) > config.RebootInterval {
				if state != REBOOTING {
					m.EventService.E <- NewLogEvent(c, "transitioning to REBOOTING state...")
				}
				state = REBOOTING
			} else {
				if state != RUNNING {
					m.EventService.E <- NewLogEvent(c, "transitioning to RUNNING state...")
				}
				state = RUNNING
			}
		case <-statsTicker.C:
			switch state {
			case RUNNING:
				stats, err := c.Stats()
				if err != nil {
					m.EventService.E <- NewErrorEvent(c, err)
				} else {
					var rebootErrors []error
					var emailErrors []error
					for _, t := range config.Thresholds {
						thresholdErrors := t.Check(stats)
						if thresholdErrors != nil && len(thresholdErrors) > 0 {
							if t.SendEmail {
								emailErrors = append(emailErrors, thresholdErrors...)
							}
							if t.CauseReboot {
								rebootErrors = append(rebootErrors, thresholdErrors...)
							}
						}
					}
					if len(rebootErrors) > 0 {
						for _, err := range rebootErrors {
							m.EventService.E <- NewErrorEvent(c, err)
							errors = append(errors, err)
						}
						failedChecks++
					}
					if len(emailErrors) > 0 {
						body := ""
						for _, err := range emailErrors {
							m.EventService.E <- NewErrorEvent(c, err)
							body += err.Error() + "\n\r"
						}
						m.EventService.E <- NewEmailEvent(c, "Thresholds Exceeded!", body)
					}
					if len(rebootErrors) == 0 && len(emailErrors) == 0 {
						reset = true
					}
				}
			case REBOOTING:
				m.EventService.E <- NewLogEvent(c, "Attempting to reboot client...")
				if err := c.Reboot(); err != nil {
					m.EventService.E <- NewErrorEvent(c, fmt.Errorf("failed to reboot: %s", err))
					m.EventService.E <- NewEmailEvent(c, "FAILED to Reboot", fmt.Sprintf("Client was unable to be restarted due to error: %s", err))
					failedReboots++
				} else {
					m.EventService.E <- NewLogEvent(c, "rebooted successfully")
					m.EventService.E <- NewEmailEvent(c, "SUCCESSFULLY rebooted", fmt.Sprintf("Client was restarted due to events: %s", fmtErrors(errors)))
					reset = true
					lastReboot = time.Now()
				}
			case POWERCYCLING:
				m.EventService.E <- NewLogEvent(c, fmt.Sprintf("Attempting to power cycle..."))
				if err := c.PowerCycle(); err != nil {
					m.EventService.E <- NewErrorEvent(c, err)
					m.EventService.E <- NewEmailEvent(c, "FAILED to Power Cycle", fmt.Sprintf("Client was unable to power cycle due to error: %s", err))
				} else {
					m.EventService.E <- NewLogEvent(c, "power cycled successfully")
					m.EventService.E <- NewEmailEvent(c, "SUCCESSFULLY Power Cycled", fmt.Sprintf("Client was power cycled due to errors: %s", fmtErrors(errors)))
					reset = true
					lastReboot = time.Now()
				}
			}
		case <-stop:
			m.EventService.E <- NewLogEvent(c, "Client monitoring stopped")
			return
		}
	}
}
