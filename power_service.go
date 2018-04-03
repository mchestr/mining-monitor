package miningmonitor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/oliveagle/jsonpath"
	"github.com/sausheong/hs1xxplug"
)

const (
	hs110plugRelayStateJSONPath = "$.system.get_sysinfo.relay_state"
	hs110plugPowerJSONPath      = "$.emeter.get_realtime.power"
)

// PowerState of the PowerService
type PowerState struct {
	On    bool
	Power float64
}

// String returns a human readable state of the PowerState
func (p PowerState) String() string {
	return fmt.Sprintf("{On: %t, Power: %0.2f}", p.On, p.Power)
}

// PowerService interface to implement a power service for power cycling the miner
type PowerService interface {
	Off() error
	On() error
	PowerCycle() error
	State() (*PowerState, error)
}

// HS110PowerService implements PowerService for the HS110 Smart Plug
type HS110PowerService struct {
	IP string

	c hs1xxplug.Hs1xxPlug
}

// NewHS110PowerService returns a new PowerService for the HS110 Smart Plug
func NewHS110PowerService(ip string) PowerService {
	return &HS110PowerService{
		IP: ip,
		c:  hs1xxplug.Hs1xxPlug{IPAddress: ip},
	}
}

// Off turns off the smart plug
func (h *HS110PowerService) Off() error {
	return h.c.TurnOff()
}

// On turns on the smart plug
func (h *HS110PowerService) On() error {
	return h.c.TurnOn()
}

// PowerCycle the HS110 smart plug
func (h *HS110PowerService) PowerCycle() error {
	state, err := h.State()
	if err != nil {
		return err
	}

	if state.On {
		if err := h.Off(); err != nil {
			return fmt.Errorf("failed to turn power off: %s", err)
		}
		// Wait 10 seconds before turning on again
		time.Sleep(10 * time.Second)
	}

	if err := h.On(); err != nil {
		return fmt.Errorf("failed to turn power on: %s", err)
	}
	return nil
}

// State returns the current state of the smart plug
func (h *HS110PowerService) State() (*PowerState, error) {
	info, err := h.c.MeterInfo()
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(info), &data); err != nil {
		return nil, fmt.Errorf("unable to unmarshal %s", info)
	}
	res, err := jsonpath.JsonPathLookup(data, hs110plugRelayStateJSONPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get relay_state from %s", info)
	}
	state := int(res.(float64))
	res, err = jsonpath.JsonPathLookup(data, hs110plugPowerJSONPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get power from %s", info)
	}
	power := res.(float64)
	return &PowerState{On: state == 1, Power: power}, nil
}
