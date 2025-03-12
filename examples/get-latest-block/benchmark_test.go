package main

import (
	"b46/b46/_sys_init"
	"testing"
	"time"
)

func BenchmarkFunction1(b *testing.B) {
	_ = _sys_init.NewEnviroSetup()
	// If the API has a rate limit, say 10 RPS, we can enforce ~10 calls per second.
	const rps = 5
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	for i := 0; i < b.N; i++ {
		<-ticker.C // Wait for the next tick to respect the RPS limit
		GetLatestBlockRpc()
	}
}
func BenchmarkFunction2(b *testing.B) {
	_ = _sys_init.NewEnviroSetup()
	// If the API has a rate limit, say 10 RPS, we can enforce ~10 calls per second.
	const rps = 5
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	for i := 0; i < b.N; i++ {
		<-ticker.C // Wait for the next tick to respect the RPS limit
		GetLatestBlock()
	}
}

// go test -bench=. -count 5
// go test -bench=Function1 -count 5
// go test -bench=Function2 -count 5
