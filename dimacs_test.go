// SPDX-License-Identifier: BSL-1.0

// Fixture expected costs are hand-verified:
//   tiny.min: 3 units from node 1 to node 4; cheapest path 1->2->4 at
//   cost 1+1 = 2 per unit, total = 6.

package mcf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/holiman/uint256"
)

type dimacsProblem struct {
	N            int
	Source, Sink int
	Demand       *uint256.Int
	Arcs         []Arc
}

func parseDIMACSMinCostFlow(r io.Reader) (*dimacsProblem, error) {
	scanner := bufio.NewScanner(r)

	var (
		nNodes     int
		nArcs      int
		havePLine  bool
		arcs       []Arc
		sourceID   = -1
		sinkID     = -1
		srcSupply  int64
		sinkSupply int64
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch line[0] {
		case 'c':
			continue
		case 'p':
			fields := strings.Fields(line)
			if len(fields) != 4 || fields[1] != "min" {
				return nil, fmt.Errorf("invalid problem line: %s", line)
			}
			n, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("invalid node count: %w", err)
			}
			a, err := strconv.Atoi(fields[3])
			if err != nil {
				return nil, fmt.Errorf("invalid arc count: %w", err)
			}
			nNodes = n
			nArcs = a
			havePLine = true
		case 'n':
			fields := strings.Fields(line)
			if len(fields) != 3 {
				return nil, fmt.Errorf("invalid node line: %s", line)
			}
			supply, err := strconv.ParseInt(fields[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid supply: %w", err)
			}
			id, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("invalid node id: %w", err)
			}
			if supply > 0 {
				if sourceID != -1 {
					return nil, fmt.Errorf("multiple positive-supply nodes: %d and %d", sourceID+1, id)
				}
				sourceID = id - 1
				srcSupply = supply
			} else if supply < 0 {
				if sinkID != -1 {
					return nil, fmt.Errorf("multiple negative-supply nodes: %d and %d", sinkID+1, id)
				}
				sinkID = id - 1
				sinkSupply = supply
			}
		case 'a':
			fields := strings.Fields(line)
			if len(fields) != 6 {
				return nil, fmt.Errorf("invalid arc line: %s", line)
			}
			u, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("invalid arc source: %w", err)
			}
			v, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("invalid arc target: %w", err)
			}
			low, err := strconv.ParseUint(fields[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid arc lower bound: %w", err)
			}
			if low != 0 {
				return nil, fmt.Errorf("nonzero lower bound %d on arc %d -> %d: lower bounds are not supported", low, u, v)
			}
			cap, err := strconv.ParseUint(fields[4], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid arc capacity: %w", err)
			}
			cost, err := strconv.ParseInt(fields[5], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid arc cost: %w", err)
			}
			arcs = append(arcs, Arc{
				From:     u - 1,
				To:       v - 1,
				Cost:     cost,
				Capacity: uint256.NewInt(cap),
			})
		default:
			return nil, fmt.Errorf("unrecognized line type %q: %s", line[0], line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	if !havePLine {
		return nil, fmt.Errorf("missing problem line")
	}
	if sourceID == -1 {
		return nil, fmt.Errorf("no positive-supply (source) node found")
	}
	if sinkID == -1 {
		return nil, fmt.Errorf("no negative-supply (sink) node found")
	}
	if srcSupply+sinkSupply != 0 {
		return nil, fmt.Errorf("supply/demand mismatch: source supplies %d but sink demands %d", srcSupply, -sinkSupply)
	}
	if len(arcs) != nArcs {
		return nil, fmt.Errorf("expected %d arcs, got %d", nArcs, len(arcs))
	}
	_ = nNodes // validated via problem line

	return &dimacsProblem{
		N:      nNodes,
		Source: sourceID,
		Sink:   sinkID,
		Demand: uint256.NewInt(uint64(srcSupply)),
		Arcs:   arcs,
	}, nil
}

func loadDIMACSFixture(t *testing.T, name string) *dimacsProblem {
	t.Helper()
	f, err := os.Open("testdata/dimacs/" + name)
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	defer f.Close()
	prob, err := parseDIMACSMinCostFlow(f)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return prob
}

func TestDIMACSParserTiny(t *testing.T) {
	prob := loadDIMACSFixture(t, "tiny.min")

	if prob.N != 4 {
		t.Fatalf("N: got %d, want 4", prob.N)
	}
	if prob.Source != 0 {
		t.Fatalf("Source: got %d, want 0", prob.Source)
	}
	if prob.Sink != 3 {
		t.Fatalf("Sink: got %d, want 3", prob.Sink)
	}
	if prob.Demand.Uint64() != 3 {
		t.Fatalf("Demand: got %s, want 3", prob.Demand)
	}
	if len(prob.Arcs) != 4 {
		t.Fatalf("len(Arcs): got %d, want 4", len(prob.Arcs))
	}

	a := prob.Arcs[0]
	if a.From != 0 {
		t.Fatalf("Arcs[0].From: got %d, want 0", a.From)
	}
	if a.To != 1 {
		t.Fatalf("Arcs[0].To: got %d, want 1", a.To)
	}
	if a.Cost != 1 {
		t.Fatalf("Arcs[0].Cost: got %d, want 1", a.Cost)
	}
	if a.Capacity.Cmp(uint256.NewInt(3)) != 0 {
		t.Fatalf("Arcs[0].Capacity: got %s, want 3", a.Capacity)
	}
}

func TestDIMACSParserRejectsLowerBound(t *testing.T) {
	input := "p min 2 1\na 1 2 1 10 5\n"
	_, err := parseDIMACSMinCostFlow(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for nonzero lower bound, got nil")
	}
	if !strings.Contains(err.Error(), "lower bound") {
		t.Fatalf("error should mention lower bound, got: %v", err)
	}
}
