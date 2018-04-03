package miningmonitor

import (
	"fmt"

	"strconv"

	"github.com/golang/glog"
)

// ThresholdFunc contains logic to determine if any actions need to be taken on the client
type ThresholdFunc func(stats *Statistics) []error

// IntComparison will comapre the two integers provided
type IntComparison func(a, b int) bool

// FloatComparison will compare the two floats provided
type FloatComparison func(a, b float64) bool

// StringComparison will compare the two strings provided
type StringComparison func(a, b string) bool

// IntGreaterThan returns true is a > b
func IntGreaterThan(a, b int) bool {
	return a > b
}

// IntLessThan returns true if a < b
func IntLessThan(a, b int) bool {
	return a < b
}

// FloatGreaterThan returns true is a > b
func FloatGreaterThan(a, b float64) bool {
	return a > b
}

// FloatLessThan returns true if a < b
func FloatLessThan(a, b float64) bool {
	return a < b
}

// FloatComparatorFromString will parse a string of the format "<10.0" to use the proper comparator
func FloatComparatorFromString(s string) FloatComparison {
	switch []rune(s)[0] {
	case []rune(">")[0]:
		return FloatGreaterThan
	case []rune("<")[0]:
		return FloatLessThan
	default:
		panic(fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number", s))
	}
}

// IntComparatorFromString will parse a string of the format "<10.0" to use the proper comparator
func IntComparatorFromString(s string) IntComparison {
	switch []rune(s)[0] {
	case []rune(">")[0]:
		return IntGreaterThan
	case []rune("<")[0]:
		return IntLessThan
	default:
		panic(fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number", s))
	}
}

// Threshold used to take action on a client if exceeded
type Threshold struct {
	Check       ThresholdFunc
	Threshold   string
	CauseReboot bool
	SendEmail   bool
	Name        string
}

// String human readable format ofa threshold
func (t Threshold) String() string {
	return fmt.Sprintf("%s: %s - [Cause Reboot? %t, Send Email? %t]", t.Name, t.Threshold, t.CauseReboot, t.SendEmail)
}

// NewHashRateThreshold returns a Threshold that will check if a client has exceeded the given hash rate.
// threshold should be of the format "<20000" or ">20000".
func NewHashRateThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := IntComparatorFromString(threshold)
	number, err := strconv.Atoi(threshold[1:])
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold, err)
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

// NewPowerThreshold returns a Threshold that will check if a client has exceeded the given power wattage.
// threshold should be of the format "<200" or ">200".
func NewPowerThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := FloatComparatorFromString(threshold)
	number, err := strconv.ParseFloat(threshold[1:], 64)
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold, err)
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

// NewTemperatureThreshold returns a Threshold that will check if a client has exceeded the given temperature.
// threshold should be of the format "<20" or ">20".
func NewTemperatureThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := FloatComparatorFromString(threshold)
	number, err := strconv.ParseFloat(threshold[1:], 64)
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold, err)
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

// NewFanPercentThreshold returns a Threshold that will check if a client has exceeded the given fan percent.
// threshold should be of the format "<20" or ">20".
func NewFanPercentThreshold(threshold string, causeReboot, sendEmail bool) (*Threshold, error) {
	comp := FloatComparatorFromString(threshold)
	number, err := strconv.ParseFloat(threshold[1:], 64)
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold, err)
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
