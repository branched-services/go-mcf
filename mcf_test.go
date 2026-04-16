// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"errors"
	"testing"

	"github.com/holiman/uint256"
)

func TestArcResultRoundTrip(t *testing.T) {
	cap := uint256.NewInt(1000)
	flow := uint256.NewInt(42)

	arc := Arc{
		From:     0,
		To:       1,
		Cost:     99,
		Capacity: cap,
		Flow:     flow,
	}

	if arc.From != 0 {
		t.Fatalf("From: got %d, want 0", arc.From)
	}
	if arc.To != 1 {
		t.Fatalf("To: got %d, want 1", arc.To)
	}
	if arc.Cost != 99 {
		t.Fatalf("Cost: got %d, want 99", arc.Cost)
	}
	if arc.Capacity.Cmp(uint256.NewInt(1000)) != 0 {
		t.Fatalf("Capacity: got %s, want 1000", arc.Capacity)
	}
	if arc.Flow.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("Flow: got %s, want 42", arc.Flow)
	}

	totalFlow := uint256.NewInt(500)
	res := Result{
		TotalFlow: totalFlow,
		TotalCost: 12345,
	}

	if res.TotalFlow.Cmp(uint256.NewInt(500)) != 0 {
		t.Fatalf("TotalFlow: got %s, want 500", res.TotalFlow)
	}
	if res.TotalCost != 12345 {
		t.Fatalf("TotalCost: got %d, want 12345", res.TotalCost)
	}

	if !errors.Is(ErrInfeasible, ErrInfeasible) {
		t.Fatal("ErrInfeasible should match itself via errors.Is")
	}
	if msg := ErrInfeasible.Error(); len(msg) < 4 || msg[:4] != "mcf:" {
		t.Fatalf("ErrInfeasible message should start with \"mcf:\", got %q", msg)
	}
}
