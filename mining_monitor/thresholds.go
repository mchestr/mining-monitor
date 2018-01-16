package mining_monitor

import (
	"fmt"

	"strconv"
)

type ThresholdFunc func(stats *Statistics) []error

type IntComparison func(a, b int) bool
type StringComparison func(a, b string) bool

func IntGreaterThan(a, b int) bool {
	return a > b
}

func IntLessThan(a, b int) bool {
	return a < b
}

type Threshold struct {
	Check     ThresholdFunc
	Threshold string
}

func NewHashRateThreshold(threshold string) (*Threshold, error) {
	var comp IntComparison
	switch []rune(threshold)[0] {
	case []rune(">")[0]:
		comp = IntGreaterThan
	case []rune("<")[0]:
		comp = IntLessThan
	default:
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number", threshold[0])
	}
	number, err := strconv.Atoi(threshold[1:])
	if err != nil {
		return nil, fmt.Errorf("unknown threshold found %s, a threshold must have a first character of '>|<' followed by a number: %s", threshold[0], err)
	}
	return &Threshold{
		Check: func(stats *Statistics) []error {
			var errors []error
			for i, hash := range stats.MainGpuHashRate {
				if comp(int(hash), number) {
					errors = append(errors, fmt.Errorf("GPU %d threshold exceeded %d%s", i, int(hash), threshold))
				}
			}
			return errors
		},
		Threshold: threshold,
	}, nil
}
