package utils

// Is it worth adding implementing a proper rope data structure if our inputs are always short?

import "fmt"

type FixedArray struct {
	cap uint
	len uint
	buffer []rune
}

func NewFixedArray(capacity uint) *FixedArray {

	arr := new(FixedArray)

	*arr = FixedArray { cap: capacity, len: 0, buffer: make([]rune, capacity) }

	return arr
}

func (arr *FixedArray) Append(ch rune) {
	Assert(arr.len < arr.cap, "Array is full")
	arr.buffer[arr.len] = ch
	arr.len++
}

func (arr *FixedArray) Last() rune {
	Assert(arr.len > 0, "Array is empty")
	return arr.buffer[arr.len]
}

func (arr *FixedArray) First() rune {
	Assert(arr.len > 0, "Array is empty")
	return arr.buffer[0]
}

func (arr *FixedArray) Pop() rune {
	last := arr.Last()
	arr.len--
	return last
}

func (arr *FixedArray) Delete(index uint) rune {
	Assert(index < arr.len, fmt.Sprintf("Index out of bounds: %d with len %d", index, arr.len))

	if index == arr.len - 1 {
		return arr.Pop()
	}

	ret := arr.buffer[index]
	for i := index; i < arr.len - 1; i++ {
		arr.buffer[i] = arr.buffer[i + 1]
	}
	arr.len--

	return ret
}

func (arr *FixedArray) Insert(index uint, ch rune) {
	Assert(index <= arr.len, "Index out of bounds")
	Assert(arr.len < arr.cap, "Array is full")

	if index == arr.len {
		arr.Append(ch)
		return
	}

	for i := arr.len; i > index; i-- {
		arr.buffer[i] = arr.buffer[i - 1]
	}
	arr.buffer[index] = ch
	arr.len++
}

func (arr *FixedArray) Reset() {
	arr.len = 0
}

func (arr *FixedArray) Len() uint {
	return arr.len
}

func (arr *FixedArray) Cap() uint {
	return arr.cap
}

func (arr *FixedArray) String() string {
	if arr.len == 0 { return "" }
	return string(arr.buffer[0:arr.len])
}

