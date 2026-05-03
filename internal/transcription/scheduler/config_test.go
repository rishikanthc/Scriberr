package scheduler

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchedulerDefaultConfigUsesPriorityPolicy(t *testing.T) {
	config := DefaultConfig()

	require.Equal(t, PolicyPriority, config.Policy)
	require.NoError(t, config.Validate())
}

func TestSchedulerConfigValidationAcceptsPlannedPolicies(t *testing.T) {
	for _, policy := range []Policy{PolicyPriority, PolicyFIFO, PolicyWeightedDuration} {
		t.Run(string(policy), func(t *testing.T) {
			require.NoError(t, Config{Policy: policy}.Validate())
		})
	}
	require.NoError(t, Config{Policy: PolicyFairShare, MaxConcurrentPerUser: 1}.Validate())
}

func TestSchedulerParseJSONRejectsInvalidOrLooseConfig(t *testing.T) {
	cases := map[string]string{
		"unknown policy":  `{"policy":"random"}`,
		"unknown field":   `{"policy":"priority","extra":true}`,
		"missing policy":  `{}`,
		"fair share cap":  `{"policy":"fair_share"}`,
		"negative cap":    `{"policy":"priority","max_concurrent_per_user":-1}`,
		"malformed json":  `{"policy":`,
		"trailing object": `{"policy":"priority"} {"policy":"fifo"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseJSON(raw)
			require.Error(t, err)
			require.True(t, errors.Is(err, ErrInvalidConfig))
		})
	}
}

func TestSchedulerMarshalRejectsInvalidConfig(t *testing.T) {
	_, err := Marshal(Config{Policy: "random"})

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidConfig))
}
