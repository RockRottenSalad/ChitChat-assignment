package clocks

type Clock interface {
	ProgressTime()
	MergeClocks(that *Clock)
	HappensBefore(that *Clock) bool
	ThisTime() uint32
}

