// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"testing"

	"github.com/holiman/uint256"
)

// failRecorder implements testing.TB and records whether Errorf was called.
type failRecorder struct {
	testing.TB
	failed bool
}

func (r *failRecorder) Errorf(string, ...any) { r.failed = true }
func (r *failRecorder) Helper()               {}

func TestCheckSolutionHelperAcceptsValidInstance(t *testing.T) {
	arcs := []Arc{
		{
			From:     0,
			To:       1,
			Cost:     3,
			Capacity: uint256.NewInt(10),
			Flow:     uint256.NewInt(5),
		},
	}
	demand := uint256.NewInt(5)
	res := Result{
		TotalFlow: uint256.NewInt(5),
		TotalCost: 15,
	}

	checkSolution(t, arcs, 2, 0, 1, demand, res)
}

func TestCheckSolutionHelperFlagsCapacityViolation(t *testing.T) {
	arcs := []Arc{
		{
			From:     0,
			To:       1,
			Cost:     3,
			Capacity: uint256.NewInt(10),
			Flow:     uint256.NewInt(20),
		},
	}
	demand := uint256.NewInt(20)
	res := Result{
		TotalFlow: uint256.NewInt(20),
		TotalCost: 60,
	}

	rec := &failRecorder{TB: t}
	checkSolution(rec, arcs, 2, 0, 1, demand, res)
	if !rec.failed {
		t.Error("expected checkSolution to report capacity violation")
	}
}
