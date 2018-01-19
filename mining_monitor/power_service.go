package mining_monitor

import (
	"encoding/json"
	"fmt"

	"github.com/oliveagle/jsonpath"
	"github.com/sausheong/hs1xxplug"
)

const (
	hs110plugRelayStateJsonPath = "$.system.get_sysinfo.relay_state"
	hs110plugPowerJsonPath      = "$.emeter.get_realtime.power"
)

type PowerState struct {
	On    bool
	Power float64
}

func (p PowerState) String() string {
	return fmt.Sprintf("{On: %t, Power: %0.2f}", p.On, p.Power)
}

type PowerService interface {
	Off() error
	On() error
	State() (*PowerState, error)
}

type HS110PowerService struct {
	IP string

	c hs1xxplug.Hs1xxPlug
}

func NewHS110PowerService(ip string) PowerService {
	return &HS110PowerService{
		IP: ip,
		c:  hs1xxplug.Hs1xxPlug{IPAddress: ip},
	}
}

func (h *HS110PowerService) Off() error {
	return h.c.TurnOff()
}

func (h *HS110PowerService) On() error {
	return h.c.TurnOn()
}

func (h *HS110PowerService) State() (*PowerState, error) {
	info, err := h.c.MeterInfo()
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(info), &data); err != nil {
		return nil, fmt.Errorf("unable to unmarshal %s", info)
	}
	res, err := jsonpath.JsonPathLookup(data, hs110plugRelayStateJsonPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get relay_state from %s", info)
	}
	state := int(res.(float64))
	res, err = jsonpath.JsonPathLookup(data, hs110plugPowerJsonPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get power from %s", info)
	}
	power := res.(float64)
	return &PowerState{On: state == 1, Power: power}, nil
}
