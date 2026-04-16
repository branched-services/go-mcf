// SPDX-License-Identifier: BSL-1.0

package mcf

import "github.com/holiman/uint256"

// Arc-state constants (LEMON: STATE_LOWER, STATE_TREE, STATE_UPPER).
const (
	stateLower int8 = -1
	stateTree  int8 = 0
	stateUpper int8 = 1
)

// Tree-arc direction relative to parent (LEMON: DIR_DOWN, DIR_UP).
const (
	dirDown int8 = -1
	dirUp   int8 = 1
)

// solverState holds the working data for the Network Simplex algorithm.
//
// Field mapping to LEMON 1.3.1 NetworkSimplex members:
//
//	n, mReal, m          – _node_num, _arc_num (real), _all_arc_num (real+artificial)
//	root                 – _root
//	source, target       – _source[i], _target[i]          (arc columns, length m)
//	cost                 – _cost[i]                         (arc column, length m)
//	capacity, flow       – _cap[i], _flow[i]                (arc columns, length m, uint256)
//	supply               – _supply[i]                       (node column, length n+1)
//	pi                   – _pi[i]                           (node potentials, length n+1)
//	parent               – _parent[i]                       (spanning-tree parent, length n+1)
//	predArc              – _pred[i]                         (predecessor arc, length n+1)
//	thread               – _thread[i]                       (DFS thread successor, length n+1)
//	revThread            – _rev_thread[i]                   (DFS thread predecessor, length n+1)
//	succNum              – _succ_num[i]                     (subtree size, length n+1)
//	lastSucc             – _last_succ[i]                    (last node in subtree, length n+1)
//	dirNode              – _pred_dir[i]                     (tree-arc direction, length n+1)
//	state                – _state[i]                        (arc state, length m)
//	nextArc              – _next_arc                        (block-search pricing cursor)
//	blockSize            – _block_size                      (block-search size)
//	bottleneck, delta    – scratch uint256 values reused per pivot (alloc-free hot path)
type solverState struct {
	// Graph dimensions.
	n     int // number of real nodes
	mReal int // number of real arcs
	m     int // total arcs: mReal + n (one artificial arc per real node)
	root  int // artificial root node index (== n)

	// Arc-indexed columns (length m). Indices [0, mReal) are real arcs;
	// [mReal, m) are artificial arcs.
	source   []int
	target   []int
	cost     []int64
	capacity []*uint256.Int
	flow     []*uint256.Int

	// Node-indexed columns (length n+1, index n is the artificial root).
	supply   []int64
	pi       []int64
	parent   []int
	predArc  []int
	thread   []int
	revThread []int
	succNum  []int
	lastSucc []int
	dirNode  []int8

	// Arc state (length m).
	state []int8

	// Block-search pricing state.
	nextArc   int
	blockSize int

	// Pre-allocated uint256 scratch buffers reused per pivot.
	bottleneck *uint256.Int
	delta      *uint256.Int
}

// newSolverState allocates a solverState for a graph with n nodes and mReal
// real arcs. It adds n artificial arcs (one per real node) so m = mReal + n.
func newSolverState(n, mReal int) *solverState {
	m := mReal + n
	nodes := n + 1

	s := &solverState{
		n:     n,
		mReal: mReal,
		m:     m,
		root:  n,

		source:   make([]int, m),
		target:   make([]int, m),
		cost:     make([]int64, m),
		capacity: make([]*uint256.Int, m),
		flow:     make([]*uint256.Int, m),

		supply:    make([]int64, nodes),
		pi:        make([]int64, nodes),
		parent:    make([]int, nodes),
		predArc:   make([]int, nodes),
		thread:    make([]int, nodes),
		revThread: make([]int, nodes),
		succNum:   make([]int, nodes),
		lastSucc:  make([]int, nodes),
		dirNode:   make([]int8, nodes),

		state: make([]int8, m),

		bottleneck: new(uint256.Int),
		delta:      new(uint256.Int),
	}

	// Pre-allocate every flow entry so the hot path can Add/Sub without nil checks.
	for i := range m {
		s.flow[i] = new(uint256.Int)
	}

	return s
}
