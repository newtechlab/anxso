package main

import "math/rand"

type UniformSampler struct {
	low, high uint32
}

func (u UniformSampler) TransactionSize(r *rand.Rand) uint32 {
	// Consider 1000 to be $1
	return uint32((r).Int31n(int32(u.high-u.low))) + u.low
}
