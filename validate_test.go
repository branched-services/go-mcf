// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/holiman/uint256"
)

func TestValidate(t *testing.T) {
	one := uint256.NewInt(1)
	cap10 := uint256.NewInt(10)

	goodArcs := func() []Arc {
		return []Arc{
			{From: 0, To: 1, Cost: 5, Capacity: cap10},
		}
	}

	tests := []struct {
		name    string
		arcs    []Arc
		n       int
		source  int
		sink    int
		demand  *uint256.Int
		wantSub string
	}{
		{
			name:    "n less than 2",
			arcs:    goodArcs(),
			n:       1,
			source:  0,
			sink:    0,
			demand:  one,
			wantSub: "n (1) must be >= 2",
		},
		{
			name:    "source negative",
			arcs:    goodArcs(),
			n:       2,
			source:  -1,
			sink:    1,
			demand:  one,
			wantSub: "source (-1) out of range",
		},
		{
			name:    "source too large",
			arcs:    goodArcs(),
			n:       2,
			source:  2,
			sink:    1,
			demand:  one,
			wantSub: "source (2) out of range",
		},
		{
			name:    "sink negative",
			arcs:    goodArcs(),
			n:       2,
			source:  0,
			sink:    -1,
			demand:  one,
			wantSub: "sink (-1) out of range",
		},
		{
			name:    "sink too large",
			arcs:    goodArcs(),
			n:       2,
			source:  0,
			sink:    5,
			demand:  one,
			wantSub: "sink (5) out of range",
		},
		{
			name:    "source equals sink",
			arcs:    goodArcs(),
			n:       2,
			source:  1,
			sink:    1,
			demand:  one,
			wantSub: "source (1) == sink (1)",
		},
		{
			name:    "demand nil",
			arcs:    goodArcs(),
			n:       2,
			source:  0,
			sink:    1,
			demand:  nil,
			wantSub: "demand is nil",
		},
		{
			name:    "demand zero",
			arcs:    goodArcs(),
			n:       2,
			source:  0,
			sink:    1,
			demand:  uint256.NewInt(0),
			wantSub: "demand is zero",
		},
		{
			name: "arc From out of range",
			arcs: []Arc{
				{From: 7, To: 1, Cost: 1, Capacity: cap10},
			},
			n:       4,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "arcs[0].From (7) out of range [0,4)",
		},
		{
			name: "arc To out of range",
			arcs: []Arc{
				{From: 0, To: 9, Cost: 1, Capacity: cap10},
			},
			n:       4,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "arcs[0].To (9) out of range [0,4)",
		},
		{
			name: "self-loop",
			arcs: []Arc{
				{From: 2, To: 2, Cost: 1, Capacity: cap10},
			},
			n:       4,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "self-loop (2 -> 2)",
		},
		{
			name: "nil Capacity",
			arcs: []Arc{
				{From: 0, To: 1, Cost: 1, Capacity: nil},
			},
			n:       2,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "arcs[0].Capacity is nil",
		},
		{
			name: "cost overflow boundary fail",
			arcs: []Arc{
				{From: 0, To: 1, Cost: math.MaxInt64/8/3 + 1, Capacity: cap10},
			},
			n:       2,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "overflows guard",
		},
		{
			name: "cost overflow negative",
			arcs: []Arc{
				{From: 0, To: 1, Cost: -(math.MaxInt64/8/3 + 1), Capacity: cap10},
			},
			n:       2,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "overflows guard",
		},
		{
			name: "cost overflow MinInt64",
			arcs: []Arc{
				{From: 0, To: 1, Cost: math.MinInt64, Capacity: cap10},
			},
			n:       2,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "overflows guard",
		},
		{
			name: "cost overflow interior pass",
			arcs: []Arc{
				{From: 0, To: 1, Cost: math.MaxInt64/8/3 - 1, Capacity: cap10},
			},
			n:       2,
			source:  0,
			sink:    1,
			demand:  one,
			wantSub: "", // should pass validation
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Solve(context.Background(), tc.arcs, tc.n, tc.source, tc.sink, tc.demand)
			if tc.wantSub == "" {
				if err != nil && errors.Is(err, ErrInvalidInput) {
					t.Fatalf("expected validation to pass, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected errors.Is(err, ErrInvalidInput), got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidateValidInput(t *testing.T) {
	arcs := []Arc{
		{From: 0, To: 1, Cost: 5, Capacity: uint256.NewInt(10)},
		{From: 1, To: 2, Cost: 3, Capacity: uint256.NewInt(20)},
	}
	err := validate(arcs, 3, 0, 2, uint256.NewInt(5))
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}
