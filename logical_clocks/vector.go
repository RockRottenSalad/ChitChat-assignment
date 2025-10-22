package clocks

import (
	"maps"
	"sync"
)

/*
protobuf definition if needed:
message TimestampEntry {
	string id   = 1;
	uint64   tick = 2:
}

message Timestamp {
	repeated TimestampEntry timestamp = 1;
}
*/

// An ordering
type Ordering int

const (
	Before Ordering = iota
	Equal
	After
	Concurrent
)

type VectorClock struct {
	id string

	mu    sync.Mutex
	clock map[string]uint64
}

func NewVector(id string) *VectorClock {
	return &VectorClock{
		clock: map[string]uint64{id: 0},
	}
}

// Ticks this clock, i.e. adds one tick to the entry corresponding to this clock
func (vc *VectorClock) Tick() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.clock[vc.id]++
}

// Syncs this clock (T) with another clock (T')
//
// That is T[j] = max(T[j], T'[j]) for every node j in T', and then T[id]++
//
// Please note: If T' contains more nodes that T, the missing nodes are added/merged into T
func (vc *VectorClock) Sync(other *VectorClock) {
	other.mu.Lock()
	cp := maps.Clone(other.clock)
	other.mu.Unlock()

	vc.mu.Lock()

	for node, ticks := range cp {
		if vc.clock[node] < ticks {
			vc.clock[node] = ticks
		}
	}
	vc.mu.Unlock()

	vc.Tick()

}

// Compares two VectorClocks by using the following order:
//
// If T[vc] < T[other], then vc -> other, i.e. vc is Before other
//
// If T[vc] > T[other], then other -> vc, i.e. vc is After other
//
// If T[vc] = T[other], then vc = other,  i.e. vc is Equal to other
//
// If none of the above is true, then vc || other, i.e. vs is Concurrent with Other
func (vc *VectorClock) Compare(other *VectorClock) Ordering {
	other.mu.Lock()
	cp := maps.Clone(other.clock)
	other.mu.Unlock()

	vc.mu.Lock()
	defer vc.mu.Unlock()

	cmpResult := Equal
	for node, ticks := range vc.clock {
		if ticks < cp[node] {
			switch cmpResult {
			case Equal:
				cmpResult = Before
			case After:
				return Concurrent
			}
		} else if ticks > cp[node] {
			switch cmpResult {
			case Equal:
				cmpResult = After
			case Before:
				return Concurrent

			}
		}
	}

	return cmpResult
}

// Returns the current timestamp of the clock
//
// The returned timestamp is a shallow copy of the internal state of the clock
func (vc *VectorClock) Now() map[string]uint64 {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	return maps.Clone(vc.clock)
}
