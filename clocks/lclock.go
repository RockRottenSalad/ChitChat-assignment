package clocks

type LamportClock struct {
	timestamp uint32
};

func NewClock(t0 uint32) LamportClock {
	return LamportClock {timestamp: t0}
}

func (this *LamportClock) ProgressTime() {
	this.timestamp += 1
}

func (this *LamportClock) MergeClocks(that *LamportClock) {
	this.timestamp = max(this.timestamp, that.timestamp) + 1
}

func (this *LamportClock) HappensBefore(that *LamportClock) bool {
	return this.timestamp < that.timestamp
}

func (this *LamportClock) ThisTime() uint32 {
	return this.timestamp
}
