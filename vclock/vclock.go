package vclock

import "log"

type VectorClock struct {
	this int
	clocks []int
};

func (c *VectorClock) ProgressTime() {
	c.clocks[c.this] += 1
}

func (c *VectorClock) MergeClocks(other *VectorClock) {

	if len(c.clocks) != len(other.clocks) {
		log.Fatalf("Clock vector D mismatch %d != %d\n", len(c.clocks), len(other.clocks))	
	}
	
	for i := range len(c.clocks) {
		c.clocks[i] = max(c.clocks[i], other.clocks[i]) + 1
	}
}

func (c *VectorClock) HappensBefore(other *VectorClock) bool {

	if len(c.clocks) != len(other.clocks) {
		log.Fatalf("Clock vector D mismatch %d != %d\n", len(c.clocks), len(other.clocks))	
	}
	
	for i := range len(c.clocks) {
		if c.clocks[i] > other.clocks[i] { return false; }
	}

	return true;
}


