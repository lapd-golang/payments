package main

import (
	"testing"
)

func TestOutgoing(t *testing.T) {
	payments := []struct {
		source   Payment
		expected Payment
	}{
		{
			source: Payment{
				AccountFromID: 5,
				AccountToID:   10,
				Amount:        100,
			},
			expected: Payment{
				AccountID:     5,
				AccountToID:   10,
				AccountFromID: 0,
				Amount:        100,
				Direction:     "outgoing",
			},
		},
	}

	for _, test := range payments {
		out := test.source.Outgoing()
		if out != test.expected {
			t.Errorf("Unexpected payment %s, expected %s", out, test.expected)
		}
	}
}
func TestIncoming(t *testing.T) {
	payments := []struct {
		source   Payment
		expected Payment
	}{
		{
			source: Payment{
				AccountFromID: 5,
				AccountToID:   10,
				Amount:        100,
			},
			expected: Payment{
				AccountID:     10,
				AccountToID:   0,
				AccountFromID: 5,
				Amount:        100,
				Direction:     "incoming",
			},
		},
	}

	for _, test := range payments {
		out := test.source.Incoming()
		if out != test.expected {
			t.Errorf("Unexpected payment %s, expected %s", out, test.expected)
		}
	}
}
