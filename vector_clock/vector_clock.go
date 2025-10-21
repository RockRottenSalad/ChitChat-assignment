package vector_clock

import (
	"fmt"
	"sync"
)

// protobuf:
/*
message TimestampIndex {
	string id   = 1;
	uint   tick = 2:
}

message Timestamp {
	repeated TimestampIndex = 2;
}
*/

type VectorClock struct {
	id string

	mu    sync.Mutex
	clock map[string]uint
}

func FromMap(id string, clock map[string]uint) *VectorClock {
	return &VectorClock{
		id:    id,
		clock: clock,
	}
}

func New(id string) *VectorClock {
	return &VectorClock{
		id:    id,
		clock: map[string]uint{id: 0},
	}
}

func (vc *VectorClock) Tick() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.clock[vc.id] += 1
}

func (vc *VectorClock) Sync(other *VectorClock) {
	vc.Tick()

	vc.mu.Lock()

	// todo: accessing other is not thread asfe atm
	for id := range other.clock {
		// vc[i] = max(vc[i], other[i])
		// Note: If vc.clock[id] doesn't exist, it returns 0.
		if vc.clock[id] < other.clock[id] {
			vc.clock[id] = other.clock[id]
		}
	}

	vc.mu.Unlock()
}

func (vc *VectorClock) ToString() string {
	return fmt.Sprintf("%v", vc.clock)
}

func (vc *VectorClock) Now() map[string]uint {
	return vc.clock
}
