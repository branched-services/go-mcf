// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"math"
	"strings"
	"testing"

	"github.com/holiman/uint256"
)

func TestValidateSolveInputs(t *testing.T) {
	cap10 := uint256.NewInt(10)
	demand := uint256.NewInt(5)

	validArcs := []Arc{
		{From: 0, To: 1, Cost: 1, Capacity: cap10},
		{From: 1, To: 2, Cost: 2, Capacity: cap10},
	}

	tests := []struct {
		name    string
		arcs    []Arc
		n       int
		source  int
		sink    int
		demand  *uint256.Int
		wantErr string // substring expected in error; empty means no error
	}{
		{
			name:   "valid 3-node graph",
			arcs:   validArcs,
			n:      3,
			source: 0,
			sink:   2,
			demand: demand,
		},
		{
			name:    "n too small",
			arcs:    nil,
			n:       1,
			source:  0,
			sink:    0,
			demand:  demand,
			wantErr: "node count",
		},
		{
			name:    "source out of range",
			arcs:    validArcs,
			n:       3,
			source:  5,
			sink:    2,
			demand:  demand,
			wantErr: "source",
		},
		{
			name:    "sink out of range",
			arcs:    validArcs,
			n:       3,
			source:  0,
			sink:    -1,
			demand:  demand,
			wantErr: "sink",
		},
		{
			name:    "source equals sink",
			arcs:    validArcs,
			n:       3,
			source:  1,
			sink:    1,
			demand:  demand,
			wantErr: "source and sink",
		},
		{
			name:    "nil demand",
			arcs:    validArcs,
			n:       3,
			source:  0,
			sink:    2,
			demand:  nil,
			wantErr: "demand",
		},
		{
			name:    "zero demand",
			arcs:    validArcs,
			n:       3,
			source:  0,
			sink:    2,
			demand:  uint256.NewInt(0),
			wantErr: "demand",
		},
		{
			name: "arc From out of range",
			arcs: []Arc{
				{From: 9, To: 1, Cost: 1, Capacity: cap10},
			},
			n:       3,
			source:  0,
			sink:    2,
			demand:  demand,
			wantErr: "From",
		},
		{
			name: "arc To out of range",
			arcs: []Arc{
				{From: 0, To: 99, Cost: 1, Capacity: cap10},
			},
			n:       3,
			source:  0,
			sink:    2,
			demand:  demand,
			wantErr: "To",
		},
		{
			name: "self-loop",
			arcs: []Arc{
				{From: 1, To: 1, Cost: 1, Capacity: cap10},
			},
			n:       3,
			source:  0,
			sink:    2,
			demand:  demand,
			wantErr: "self-loop",
		},
		{
			name: "nil capacity",
			arcs: []Arc{
				{From: 0, To: 1, Cost: 1, Capacity: nil},
			},
			n:       3,
			source:  0,
			sink:    2,
			demand:  demand,
			wantErr: "capacity",
		},
		{
			name: "cost exceeds bound",
			arcs: []Arc{
				{From: 0, To: 1, Cost: math.MinInt64, Capacity: cap10},
			},
			n:       3,
			source:  0,
			sink:    2,
			demand:  demand,
			wantErr: "cost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSolveInputs(tt.arcs, tt.n, tt.source, tt.sink, tt.demand)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
