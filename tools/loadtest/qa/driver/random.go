package main

import "math/rand/v2"

// The driver's randomness picks hosts, labels and queries and spaces arrivals; none of it protects anything, so
// math/rand/v2 is the right tool (and the schedule's generator can be seeded to replay a run).

func randIntN(n int) int   { return rand.IntN(n) }   //nolint:gosec // load-test choices, not security
func randFloat() float64   { return rand.Float64() } //nolint:gosec // load-test choices, not security
func randPerm(n int) []int { return rand.Perm(n) }   //nolint:gosec // load-test choices, not security
func newSeeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // a replayable schedule, not security
}
func randSeed() uint64 { return rand.Uint64() } //nolint:gosec // a replayable schedule, not security
