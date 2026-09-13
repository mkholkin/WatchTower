package testutil

import (
	"hash/fnv"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

// Shuffle complements go test -shuffle, which does not shuffle table subtests.
func Shuffle[T any](t *testing.T, cases []T) []T {
	t.Helper()
	seedText := os.Getenv("LAB1_SEED")
	if seedText == "" {
		return cases
	}
	seed, err := strconv.ParseInt(seedText, 10, 64)
	if err != nil {
		t.Fatalf("invalid LAB1_SEED: %v", err)
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(t.Name()))
	result := append([]T(nil), cases...)
	rand.New(rand.NewSource(seed^int64(hash.Sum64()))).Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	return result
}
