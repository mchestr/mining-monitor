package mining_monitor

import (
	"fmt"

	"strconv"

	"github.com/golang/glog"
)

type ThresholdFunc func(stats *Statistics) []error

type IntComparison func(a, b int) bool
type FloatComparison func(a, b float64) bool
type StringComparison func(a, b string) bool

func IntGreaterThan(a, b int) bool {
	return a > b
}

func IntLessThan(a, b int) bool {
	return a < b
}

func FloatGreaterThan(a, b float64) bool {
	return a > b
}

func FloatLessThan(a, b float64) bool {
	return a < b
}

func FloatComparatorFromstring(s string) FloatComparison {
	switch []rune(s)[0] {
	case []rune(">")[0]:
		return FloatGreaterThan
	case []rune("<")[0]:
		return FloatLessThan
	default:
		panic(fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number", s[0]))
	}
}

func IntComparatorFromstring(s string) IntComparison {
	switch []rune(s)[0] {
	case []rune(">")[0]:
		return IntGreaterThan
	case []rune("<")[0]:
		return IntLessThan
	default:
		panic(fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number", s[0]))
	}
}

type Threshold struct {
	Check       ThresholdFunc
	Threshold   string
	CauseReboot bool
	SendEmail   bool
	Name        string
}

func (t Threshold) String() string {
	return fmt.Sprintf("%s: %s", t.Name, t.Threshold)
}

func NewHashRateThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := IntComparatorFromstring(threshold)
	number, err := strconv.Atoi(threshold[1:])
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold[0], err)
	}
	return &Threshold{
		Check: func(stats *Statistics) []error {
			var errors []error
			for i, hash := range stats.MainGpuHashRate {
				glog.V(2).Infof("GPU %d hashrate %0.2f", i, hash)
				if comp(int(hash), number) {
					errors = append(errors, fmt.Errorf("GPU %d threshold exceeded %d%s", i, int(hash), threshold))
				}
			}
			return errors
		},
		Threshold:   threshold,
		CauseReboot: causeReboot,
		SendEmail:   sendEmail,
		Name:        "HashRate",
	}, nil
}

func NewPowerThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := FloatComparatorFromstring(threshold)
	number, err := strconv.ParseFloat(threshold[1:], 64)
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold[0], err)
	}
	return &Threshold{
		Check: func(stats *Statistics) []error {
			glog.V(2).Infof("rig power %0.2f", stats.PowerState.Power)
			if comp(stats.PowerState.Power, number) {
				return []error{fmt.Errorf("power threshold exceeded %0.2f%s", stats.PowerState.Power, threshold)}
			}
			return nil
		},
		Threshold:   threshold,
		CauseReboot: causeReboot,
		SendEmail:   sendEmail,
		Name:        "Power",
	}, nil
}

func NewTemperatureThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := FloatComparatorFromstring(threshold)
	number, err := strconv.ParseFloat(threshold[1:], 64)
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold[0], err)
	}
	return &Threshold{
		Check: func(stats *Statistics) []error {
			var errors []error
			for i, temp := range stats.GpuTemperatures {
				glog.V(2).Infof("GPU %d temperature %0.2f", i, temp)
				if comp(temp, number) {
					errors = append(errors, fmt.Errorf("GPU %d temperature threshold exceeded %0.2f%s", i, temp, threshold))
				}
			}
			return errors
		},
		Threshold:   threshold,
		CauseReboot: causeReboot,
		SendEmail:   sendEmail,
		Name:        "Temp",
	}, nil
}

func NewFanPercentThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := FloatComparatorFromstring(threshold)
	number, err := strconv.ParseFloat(threshold[1:], 64)
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold[0], err)
	}
	return &Threshold{
		Check: func(stats *Statistics) []error {
			var errors []error
			for i, fp := range stats.GpuFanPercents {
				glog.V(2).Infof("GPU %d fan percent %0.2f", i, fp)
				if comp(fp, number) {
					errors = append(errors, fmt.Errorf("GPU %d fan percent threshold exceeded %0.2f%s", i, fp, threshold))
				}
			}
			return errors
		},
		Threshold:   threshold,
		CauseReboot: causeReboot,
		SendEmail:   sendEmail,
		Name:        "FanPercent",
	}, nil
}
