package clocks

import "log"

type VectorClock struct {
	this uint32
	clocks []uint32
};

func (this *VectorClock) ProgressTime() {
	this.clocks[this.this] += 1
}

func (this *VectorClock) MergeClocks(that *VectorClock) {

	if len(this.clocks) != len(that.clocks) {
		log.Fatalf("Clock vector D mismatch %d != %d\n", len(this.clocks), len(that.clocks))	
	}
	
	for i := range len(this.clocks) {
		this.clocks[i] = max(this.clocks[i], that.clocks[i]) + 1
	}
}

func (this *VectorClock) HappensBefore(that *VectorClock) bool {

	if len(this.clocks) != len(that.clocks) {
		log.Fatalf("Clock vector D mismatch %d != %d\n", len(this.clocks), len(that.clocks))	
	}
	
	for i := range len(this.clocks) {
		if this.clocks[i] > that.clocks[i] { return false }
	}

	return true
}

func (this *VectorClock) ThisTime() uint32 {
	return this.clocks[this.this];
}


