package logicalclocks

import "sync/atomic"

type LamportClock struct {
	ticks atomic.Uint64
}

func NewLamport() *LamportClock {
	return &LamportClock{ticks: atomic.Uint64{}}
}

func From(ticks uint64) *LamportClock {
	clock := NewLamport()
	clock.Set(ticks)

	return clock
}

func (lc *LamportClock) Tick() {
	lc.ticks.Add(1)
}

func (lc *LamportClock) Set(ticks uint64) {
	lc.ticks.Store(ticks)
}

func (lc *LamportClock) Sync(other *LamportClock) uint64 {
	// Uses a CAS loop to sync the clocks in a safe manner
	// If the atomic counter sees that its been changed during the computation of the new value it simply retires
	for {
		old := lc.ticks.Load()
		synced := max(old, other.ticks.Load()) + 1

		if lc.ticks.CompareAndSwap(old, synced) {
			return synced
		}
	}
}

func (lc *LamportClock) Now() uint64 {
	return lc.ticks.Load()
}

/*func (lc *LamportClock) Compare(other *LamportClock) Ordering {
	this := lc.ticks.Load()
	that := other.ticks.Load()

	// If a -> b, then T(a) < T(b)
	// But from T(a) < T(b) we can't infer a -> b

	// In our implementation the ticks of the server on the recieval of a message from a client is the one we use
	// This means that the server defines the real ordering, and thus if T(a) < T(b) then we can in fact infer that a -> b
	// Obviously, this is not how real lamport clocks work
	if this < that {
		return Less
	} else if that > this {
		return Greater
	}

	return Equal
}*/

