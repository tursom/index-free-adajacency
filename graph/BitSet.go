package graph

import (
	"encoding/hex"
)

type BitSet []uint8

func (b BitSet) BitLength() int {
	return len(b) * 8
}

func (b BitSet) SetBit(bit int, up bool) (old bool) {
	arrIndex := bit / 8

	location := uint8(1) << (bit % 8)
	oldValue := b[arrIndex]
	if up {
		b[arrIndex] = oldValue | location
	} else {
		b[arrIndex] = oldValue & ^location
	}
	return oldValue&location != 0
}

func (b BitSet) Get(bit int) (up bool) {
	arrIndex := bit / 8
	location := uint8(1) << (bit % 8)

	return b[arrIndex]&location != 0
}

func (b BitSet) String() string {
	return hex.EncodeToString(b)
}

func (b BitSet) NextUp(from int) int {
	if from < 0 {
		from = 0
	} else {
		from++
	}

	if from >= len(b)*8 {
		return -1
	}

	for !b.Get(from) {
		from++
		if from >= len(b)*8 {
			return -1
		}
	}
	return from
}

func (b BitSet) NextDown(from int) int {
	if from < 0 {
		from = 0
	} else {
		from++
	}

	if from >= len(b)*8 {
		return -1
	}

	for b.Get(from) {
		from++
		if from >= len(b)*8 {
			return -1
		}
	}
	return from
}
